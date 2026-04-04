package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/api"
	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/memory"
	"github.com/tonypk/ai-management-brain/internal/report"
	"github.com/tonypk/ai-management-brain/internal/scheduler"
)

// createSchedulerCallbacks builds the schedulerCallbacks that wire remind/chase/summary to real operations.
func createSchedulerCallbacks(svc *services) *schedulerCallbacks {
	return &schedulerCallbacks{
		remindFn: func(ctx context.Context) error {
			slog.Info("remind job: sending check-in questions")
			tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}

			if tenant.OnboardingCompletedAt == nil {
				slog.Debug("skipping scheduler job for tenant still in onboarding", "tenant_id", tenant.ID)
				return nil
			}

			// Get mentor's questions (supports blending)
			engine, err := engineForTenant(svc.engineFactory, tenant, "default")
			if err != nil {
				return fmt.Errorf("load engine for remind: %w", err)
			}
			questions := engine.GetCheckinQuestions()

			emps, err := svc.reportDB.ListActiveEmployees(ctx, tenant.ID)
			if err != nil {
				return fmt.Errorf("list employees: %w", err)
			}
			if len(emps) == 0 {
				slog.Info("remind job: no employees to remind")
				return nil
			}
			for _, emp := range emps {
				_, firstQ, err := svc.collector.StartWithQuestions(ctx, emp.ID, questions)
				if err != nil {
					slog.Error("start collection", "employee_id", emp.ID, "error", err)
					continue
				}
				msg := fmt.Sprintf("Good morning %s! Time for your daily check-in.\n\n%s", emp.Name, firstQ)
				if err := svc.tgBot.SendMessage(emp.TelegramID, msg); err != nil {
					slog.Error("send remind", "employee_id", emp.ID, "error", err)
				}
			}
			slog.Info("remind job: completed", "employees_reminded", len(emps), "mentor", tenant.MentorID)
			return nil
		},
		chaseFn: func(ctx context.Context) error {
			slog.Info("chase job: chasing non-submitters")
			tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}

			if tenant.OnboardingCompletedAt == nil {
				slog.Debug("skipping scheduler job for tenant still in onboarding", "tenant_id", tenant.ID)
				return nil
			}

			today := time.Now().In(svc.loc).Format("2006-01-02")
			return svc.chaser.ChaseAll(ctx, tenant.ID, today, tenant.MentorID)
		},
		summaryFn: func(ctx context.Context) error {
			slog.Info("summary job: generating daily summary")
			tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}

			if tenant.OnboardingCompletedAt == nil {
				slog.Debug("skipping scheduler job for tenant still in onboarding", "tenant_id", tenant.ID)
				return nil
			}

			engine, err := engineForTenant(svc.engineFactory, tenant, "default")
			if err != nil {
				return fmt.Errorf("load engine for summary: %w", err)
			}
			today := time.Now().In(svc.loc).Format("2006-01-02")
			result, err := svc.summarizer.Generate(ctx, tenant.ID, today, engine)
			if err != nil {
				return fmt.Errorf("generate summary: %w", err)
			}
			header := fmt.Sprintf("Daily Summary (%s)\nMentor: %s\nSubmission rate: %.0f%%\n\n", today, tenant.MentorID, result.SubmissionRate*100)
			if err := svc.tgBot.SendMessage(svc.cfg.BossTelegramID, header+result.Content); err != nil {
				return fmt.Errorf("send summary to boss: %w", err)
			}
			slog.Info("summary job: completed", "submission_rate", result.SubmissionRate, "mentor", tenant.MentorID)

			// Run trigger rules after summary
			bossEmp := report.EmployeeInfo{
				ID: "boss", Name: "Boss",
				TelegramID:       svc.cfg.BossTelegramID,
				PreferredChannel: "telegram",
			}
			triggerResults, err := svc.triggerChecker.CheckAll(ctx, tenant.ID, tenant.MentorID, bossEmp)
			if err != nil {
				slog.Error("trigger check failed", "error", err)
			} else if len(triggerResults) > 0 {
				slog.Info("triggers fired", "count", len(triggerResults))
			}

			return nil
		},
	}
}

// registerSchedulerJobs registers all scheduled jobs with the scheduler.
func registerSchedulerJobs(svc *services, sched *scheduler.Scheduler) {
	// World Model cron jobs
	if err := sched.AddJob("wm_decay", "0 3 * * *", func(ctx context.Context) error {
		return svc.wmDecay.RunForAllTenants(ctx)
	}); err != nil {
		slog.Error("failed to add wm_decay job", "error", err)
	}
	if err := sched.AddJob("wm_insights", "15 19 * * *", func(ctx context.Context) error {
		return svc.wmInsights.GenerateForAllTenants(ctx)
	}); err != nil {
		slog.Error("failed to add wm_insights job", "error", err)
	}

	// Boss employee info for proactive actions (channel-agnostic)
	bossEmployeeInfo := report.EmployeeInfo{
		ID: "boss", Name: "Boss",
		TelegramID:       svc.cfg.BossTelegramID,
		PreferredChannel: "telegram",
	}

	// Register proactive action jobs
	if err := sched.AddJob("weekly_actions", "0 18 * * 5", func(ctx context.Context) error {
		slog.Info("weekly actions job: running proactive actions")
		tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
		if err != nil {
			return fmt.Errorf("get tenant: %w", err)
		}
		return svc.actionExecutor.RunWeekly(ctx, tenant.ID, tenant.MentorID, bossEmployeeInfo)
	}); err != nil {
		slog.Error("failed to register weekly actions job", "error", err)
	}

	if err := sched.AddJob("monthly_actions", "0 18 1 * *", func(ctx context.Context) error {
		slog.Info("monthly actions job: running proactive actions")
		tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
		if err != nil {
			return fmt.Errorf("get tenant: %w", err)
		}
		return svc.actionExecutor.RunMonthly(ctx, tenant.ID, tenant.MentorID, bossEmployeeInfo)
	}); err != nil {
		slog.Error("failed to register monthly actions job", "error", err)
	}
	slog.Info("proactive action jobs registered", "weekly", "Friday 18:00", "monthly", "1st 18:00")

	// Memory consolidation jobs
	if svc.memEngine != nil {
		if err := sched.AddJob("memory-clean", "0 2 * * *", func(ctx context.Context) error {
			slog.Info("memory-clean job: cleaning expired memories")
			return svc.memEngine.RunConsolidation(ctx, memory.ConsolidationClean)
		}); err != nil {
			slog.Error("failed to register memory-clean job", "error", err)
		}

		if err := sched.AddJob("memory-consolidate", "0 3 * * 0", func(ctx context.Context) error {
			slog.Info("memory-consolidate job: merging short-term memories")
			return svc.memEngine.RunConsolidation(ctx, memory.ConsolidationMerge)
		}); err != nil {
			slog.Error("failed to register memory-consolidate job", "error", err)
		}

		if err := sched.AddJob("memory-profiles", "0 4 1 * *", func(ctx context.Context) error {
			slog.Info("memory-profiles job: rebuilding employee profiles")
			return svc.memEngine.RunConsolidation(ctx, memory.ConsolidationRebuild)
		}); err != nil {
			slog.Error("failed to register memory-profiles job", "error", err)
		}

		slog.Info("memory consolidation jobs registered",
			"clean", "daily 02:00",
			"consolidate", "weekly Sunday 03:00",
			"profiles", "monthly 1st 04:00",
		)
	}

	// Group mentor autonomous posting job
	if svc.chatService != nil && svc.chatService.LLM() != nil {
		if err := sched.AddJob("group_mentor", "0 10 * * *", func(ctx context.Context) error {
			slog.Info("group_mentor job: running autonomous posting decisions")
			tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}

			var tenantUUID pgtype.UUID
			if err := tenantUUID.Scan(tenant.ID); err != nil {
				return fmt.Errorf("parse tenant UUID: %w", err)
			}

			groups, err := svc.queries.ListActiveGroupChatsByTenant(ctx, tenantUUID)
			if err != nil {
				return fmt.Errorf("list active groups: %w", err)
			}
			if len(groups) == 0 {
				slog.Info("group_mentor job: no active groups")
				return nil
			}

			engine, err := engineForTenant(svc.engineFactory, tenant, "default")
			if err != nil {
				return fmt.Errorf("load engine: %w", err)
			}

			// Collect team data
			today := time.Now().In(svc.loc)
			weekday := today.Weekday().String()
			todayDate := pgtype.Date{Time: today.Truncate(24 * time.Hour), Valid: true}

			submissionRate := "N/A"
			emps, _ := svc.queries.ListActiveEmployees(ctx, tenantUUID)
			if len(emps) > 0 {
				submitted, _ := svc.queries.CountReportsByTenantDate(ctx, sqlc.CountReportsByTenantDateParams{
					TenantID:   tenantUUID,
					ReportDate: todayDate,
				})
				pct := float64(submitted) / float64(len(emps)) * 100
				submissionRate = fmt.Sprintf("%.0f%% (%d/%d)", pct, submitted, len(emps))
			}

			summaryText := ""
			if summary, err := svc.queries.GetLatestSummary(ctx, tenantUUID); err == nil {
				summaryText = summary.Content
				if len(summaryText) > 500 {
					summaryText = summaryText[:500] + "..."
				}
			}

			llmClient := svc.chatService.LLM()

			for _, gc := range groups {
				groupID := formatPgUUID(gc.ID)

				// Anti-spam: check Redis for last post time
				antiSpamKey := fmt.Sprintf("group:last_post:%s", groupID)
				if _, err := svc.redisClient.Get(ctx, antiSpamKey); err == nil {
					slog.Debug("group_mentor: skipping (posted recently)", "group", gc.Name)
					continue
				}

				// Build decision prompt
				prompt := brain.BuildGroupDecisionPrompt(
					engine.MentorName(),
					gc.GroupType,
					brain.GroupTeamData{
						SubmissionRate: submissionRate,
						LatestSummary:  summaryText,
						Weekday:        weekday,
					},
				)

				response, err := llmClient.Chat(ctx, prompt, "Decide whether to post.")
				if err != nil {
					slog.Error("group_mentor: LLM decision failed", "group", gc.Name, "error", err)
					continue
				}

				if brain.IsSkipDecision(response) {
					slog.Debug("group_mentor: AI decided SKIP", "group", gc.Name)
					continue
				}

				// Send message to group
				chatID, _ := strconv.ParseInt(gc.PlatformChatID, 10, 64)
				if chatID == 0 {
					slog.Error("group_mentor: invalid chat ID", "platform_chat_id", gc.PlatformChatID)
					continue
				}

				if err := svc.tgBot.SendMessage(chatID, response); err != nil {
					slog.Error("group_mentor: send failed", "group", gc.Name, "error", err)
					continue
				}

				// Set anti-spam key (24h TTL)
				_ = svc.redisClient.Set(ctx, antiSpamKey, "1", 24*time.Hour)
				slog.Info("group_mentor: posted to group", "group", gc.Name, "message_len", len(response))
			}

			return nil
		}); err != nil {
			slog.Error("failed to register group_mentor job", "error", err)
		} else {
			slog.Info("group_mentor job registered", "schedule", "daily 10:00")
		}
	}

	// Goal snapshot cron job (daily at 23:00)
	if err := sched.AddJob("goal_snapshots", "0 23 * * *", func(ctx context.Context) error {
		slog.Info("goal_snapshots job: calculating daily progress")

		tenantIDs, err := svc.queries.ListTenantsWithActiveGoals(ctx)
		if err != nil {
			return fmt.Errorf("list tenants with active goals: %w", err)
		}

		today := pgtype.Date{Time: time.Now().Truncate(24 * time.Hour), Valid: true}
		var snapshotCount int

		for _, tenantID := range tenantIDs {
			goals, err := svc.queries.ListActiveGoalsByTenant(ctx, tenantID)
			if err != nil {
				slog.Error("goal_snapshots: list active goals", "error", err)
				continue
			}

			for _, goal := range goals {
				krs, err := svc.queries.GetKeyResultsByGoal(ctx, goal.ID)
				if err != nil {
					slog.Error("goal_snapshots: get key results", "goal_id", goal.ID, "error", err)
					continue
				}

				progress := api.CalculateGoalProgress(krs)

				if err := svc.queries.CreateGoalSnapshot(ctx, sqlc.CreateGoalSnapshotParams{
					GoalID:          goal.ID,
					OverallProgress: numericFromFloat(progress),
					SnapshotDate:    today,
				}); err != nil {
					slog.Error("goal_snapshots: create snapshot", "goal_id", goal.ID, "error", err)
					continue
				}
				snapshotCount++
			}
		}

		slog.Info("goal_snapshots job: done", "tenants", len(tenantIDs), "snapshots", snapshotCount)
		return nil
	}); err != nil {
		slog.Error("failed to register goal_snapshots job", "error", err)
	} else {
		slog.Info("goal_snapshots job registered", "schedule", "daily 23:00")
	}

	// Brain Layer v2: Daily signal generation + working memory job
	if svc.stateEngine != nil {
		if err := sched.AddJob("brain_signals", "0 22 * * *", func(ctx context.Context) error {
			slog.Info("brain_signals job: generating execution signals + working memory")
			tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}
			var tenantUUID pgtype.UUID
			if err := tenantUUID.Scan(tenant.ID); err != nil {
				return fmt.Errorf("parse tenant UUID: %w", err)
			}

			// Generate signals for each active employee
			employees, err := svc.queries.ListActiveEmployees(ctx, tenantUUID)
			if err != nil {
				slog.Error("brain_signals: list employees", "error", err)
			} else {
				for _, emp := range employees {
					_, sigErr := svc.stateEngine.GenerateSignals(ctx, tenantUUID, "employee", emp.ID, emp.Name)
					if sigErr != nil {
						slog.Error("brain_signals: generate signal", "employee", emp.Name, "error", sigErr)
					}
				}
			}

			// Generate working memory snapshot
			contextJSON, err := svc.contextService.FormatContextForPrompt(ctx, tenantUUID)
			if err != nil {
				slog.Error("brain_signals: format context", "error", err)
			} else {
				_, err = svc.stateEngine.GenerateWorkingMemory(ctx, tenantUUID, contextJSON)
				if err != nil {
					slog.Error("brain_signals: generate working memory", "error", err)
				}
			}

			slog.Info("brain_signals job: done")
			return nil
		}); err != nil {
			slog.Error("failed to register brain_signals job", "error", err)
		} else {
			slog.Info("brain_signals job registered", "schedule", "daily 22:00")
		}
	}

	// Brain Layer v2: Monthly incentive calculation job
	if svc.incentiveEngine != nil {
		if err := sched.AddJob("incentive_calc", "0 6 1 * *", func(ctx context.Context) error {
			slog.Info("incentive_calc job: calculating monthly incentive scores")
			tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
			if err != nil {
				return fmt.Errorf("get tenant: %w", err)
			}
			var tenantUUID pgtype.UUID
			if err := tenantUUID.Scan(tenant.ID); err != nil {
				return fmt.Errorf("parse tenant UUID: %w", err)
			}

			// Calculate for the previous month
			now := time.Now().In(svc.loc)
			prevMonth := now.AddDate(0, -1, 0)
			period := prevMonth.Format("2006-01")

			employees, err := svc.queries.ListActiveEmployees(ctx, tenantUUID)
			if err != nil {
				return fmt.Errorf("list employees: %w", err)
			}

			var totalScores int
			for _, emp := range employees {
				scores, err := svc.incentiveEngine.Calculate(ctx, tenantUUID, period, emp.ID, emp.Name)
				if err != nil {
					slog.Error("incentive_calc: calculate", "employee", emp.Name, "error", err)
					continue
				}
				totalScores += len(scores)
			}

			slog.Info("incentive_calc job: done", "period", period, "employees", len(employees), "scores", totalScores)
			return nil
		}); err != nil {
			slog.Error("failed to register incentive_calc job", "error", err)
		} else {
			slog.Info("incentive_calc job registered", "schedule", "monthly 1st 06:00")
		}
	}

	// Recommendation daily scan job
	if svc.recommender != nil {
		if err := sched.AddJob("recommendation_scan", "30 10 * * *", func(ctx context.Context) error {
			slog.Info("recommendation_scan: starting")
			tenants, err := svc.queries.ListActiveTenants(ctx)
			if err != nil {
				return fmt.Errorf("list tenants: %w", err)
			}
			for _, tenant := range tenants {
				mentorID := tenant.MentorID
				if mentorID == "" {
					mentorID = "inamori"
				}
				if err := svc.recommender.DailyScan(ctx, tenant.ID, mentorID, "default"); err != nil {
					slog.Error("recommendation_scan: tenant failed", "tenant", formatPgUUID(tenant.ID), "error", err)
					continue
				}
			}
			return nil
		}); err != nil {
			slog.Error("failed to register recommendation_scan job", "error", err)
		} else {
			slog.Info("recommendation_scan job registered", "schedule", "daily 10:30")
		}
	}

	// Engagement tracker: check active consulting engagements and send progress reports
	if svc.consultingEngine != nil {
		if err := sched.AddJob("engagement_tracker", "0 11 * * *", func(ctx context.Context) error {
			slog.Info("engagement_tracker: starting")
			engagements, err := svc.queries.ListEngagementsForTracking(ctx)
			if err != nil {
				return fmt.Errorf("list engagements for tracking: %w", err)
			}
			if len(engagements) == 0 {
				slog.Info("engagement_tracker: no active engagements to track")
				return nil
			}
			for _, eng := range engagements {
				report, err := svc.consultingEngine.CheckProgress(ctx, eng.ID)
				if err != nil {
					slog.Error("engagement_tracker: check progress failed",
						"engagement_id", formatPgUUID(eng.ID), "error", err)
					continue
				}
				msg := fmt.Sprintf("Consulting Update: %s\n\n%s", eng.Title, report)
				if err := svc.tgBot.SendMessage(svc.cfg.BossTelegramID, msg); err != nil {
					slog.Error("engagement_tracker: send report failed",
						"engagement_id", formatPgUUID(eng.ID), "error", err)
				}
			}
			slog.Info("engagement_tracker: completed", "engagements_checked", len(engagements))
			return nil
		}); err != nil {
			slog.Error("failed to register engagement_tracker job", "error", err)
		} else {
			slog.Info("engagement_tracker job registered", "schedule", "daily 11:00")
		}
	}

}
