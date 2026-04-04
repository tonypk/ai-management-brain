package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/tonypk/ai-management-brain/internal/bot"
	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/events"
	"github.com/tonypk/ai-management-brain/internal/report"
)

// registerTelegramTextHandler registers the raw text handler for report collection,
// mentor chat, and group @mentions on the Telegram bot.
func registerTelegramTextHandler(svc *services) {
	svc.tgBot.RegisterRawTextHandler(func(c tele.Context) error {
		ctx := context.Background()
		senderID := c.Sender().ID
		text := c.Text()
		sendReply := func(msg string) error { return c.Send(msg) }

		// === GROUP CHAT HANDLING ===
		chatType := string(c.Chat().Type)
		if chatType == "group" || chatType == "supergroup" {
			return handleTelegramGroupMessage(svc, c, ctx, text)
		}

		// === PRIVATE CHAT HANDLING ===

		// Check if sender is the boss FIRST (boss may not be in employees table)
		if senderID == svc.cfg.BossTelegramID {
			return handleTelegramBossMessage(svc, c, ctx, senderID, text, sendReply)
		}

		// Look up employee by telegram_id
		emp, err := svc.botDB.GetEmployeeByTelegramID(ctx, senderID)
		if err != nil {
			return nil
		}

		return handleTelegramEmployeeMessage(svc, ctx, emp, text, sendReply, senderID)
	})
}

// handleTelegramGroupMessage handles @mention messages in group chats.
func handleTelegramGroupMessage(svc *services, c tele.Context, ctx context.Context, text string) error {
	botUsername := "@" + c.Bot().Me.Username
	if !strings.Contains(text, botUsername) {
		return nil // ignore non-mention messages
	}

	// Strip the @mention from the text
	cleanText := strings.ReplaceAll(text, botUsername, "")
	cleanText = strings.TrimSpace(cleanText)
	if cleanText == "" {
		return c.Reply("有什么我可以帮你的吗？")
	}

	chatID := fmt.Sprintf("%d", c.Chat().ID)
	gc, err := svc.queries.GetGroupChatByPlatformID(ctx, sqlc.GetGroupChatByPlatformIDParams{
		Platform:       "telegram",
		PlatformChatID: chatID,
	})
	if err != nil {
		slog.Debug("group message from unregistered group", "chat_id", chatID)
		return nil
	}
	if !gc.IsActive {
		return nil
	}

	// Load mentor engine
	tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
	if err != nil {
		slog.Error("group chat: get tenant", "error", err)
		return nil
	}

	engine, err := engineForTenant(svc.engineFactory, tenant, "default")
	if err != nil {
		slog.Error("group chat: load engine", "error", err)
		return nil
	}

	// Get latest summary for team context
	summaryText := ""
	if summary, err := svc.queries.GetLatestSummary(ctx, gc.TenantID); err == nil {
		summaryText = summary.Content
	}

	// Build group reply prompt
	systemPrompt := brain.BuildGroupReplyPrompt(
		engine.MentorName(),
		gc.GroupType,
		summaryText,
		cleanText,
	)

	if svc.chatService == nil || svc.chatService.LLM() == nil {
		return c.Reply(brain.AIDisabledMessage())
	}

	// Use LLM single-turn Chat
	response, err := svc.chatService.LLM().Chat(ctx, systemPrompt, cleanText)
	if err != nil {
		slog.Error("group reply LLM failed", "error", err, "group", gc.Name)
		return nil
	}

	return c.Reply(response)
}

// handleTelegramBossMessage handles private messages from the boss.
func handleTelegramBossMessage(svc *services, c tele.Context, ctx context.Context, senderID int64, text string, sendReply func(string) error) error {
	if svc.chatService == nil {
		return sendReply(brain.AIDisabledMessage())
	}

	// Onboarding intercept: if not complete, route all messages through onboarding
	if svc.onboardingSvc != nil {
		tenant, err := svc.botDB.GetTenantByBossChatID(ctx, senderID)
		if err == nil && tenant.OnboardingCompletedAt == nil {
			svc.tgAdapter.Bot().Notify(tele.ChatID(senderID), tele.Typing)
			uid, pErr := parseUUIDForChat(tenant.ID)
			if pErr == nil {
				resp, oErr := svc.onboardingSvc.HandleMessage(ctx, uid, "telegram",
					strconv.FormatInt(senderID, 10), text)
				if oErr != nil {
					slog.Error("onboarding message failed", "error", oErr)
					return sendReply("Something went wrong. Please try again.")
				}
				return sendReply(resp)
			}
		}
	}

	// C-Suite seat routing: if boss has an active seat via /talk, route to seat chat
	if svc.seatSvc != nil {
		tenant, err := svc.botDB.GetTenantByBossChatID(ctx, senderID)
		if err == nil {
			activeSeat := svc.seatSvc.GetActiveSeat(ctx, tenant.ID, senderID)
			if activeSeat != "" {
				svc.tgAdapter.Bot().Notify(tele.ChatID(senderID), tele.Typing)
				reply, seatErr := svc.seatSvc.Chat(ctx, tenant.ID, activeSeat, "default", text)
				if seatErr != nil {
					slog.Error("seat chat failed", "seat", activeSeat, "error", seatErr)
					return sendReply("Seat chat failed. Use /talk off to return to default mode.")
				}
				return sendReply(reply)
			}
		}
	}

	svc.tgAdapter.Bot().Notify(tele.ChatID(senderID), tele.Typing)
	tenant, err := svc.botDB.GetTenantByBossChatID(ctx, senderID)
	if err != nil {
		slog.Error("boss chat: get tenant", "error", err)
		return sendReply(brain.AIErrorMessage())
	}
	bossCtx := fetchBossContext(ctx, svc.queries, tenant.ID, svc.loc)
	resp, err := svc.chatService.HandleBoss(ctx, tenant.ID, tenant.MentorID, "default", text, bossCtx)
	if err != nil {
		slog.Error("boss chat failed", "error", err)
		return sendReply(brain.AIErrorMessage())
	}
	return sendReply(resp)
}

// handleTelegramEmployeeMessage handles private messages from employees (report collection + mentor chat).
func handleTelegramEmployeeMessage(svc *services, ctx context.Context, emp *bot.Employee, text string, sendReply func(string) error, senderID int64) error {
	empID := emp.ID
	state := svc.collector.GetState(ctx, empID)
	lower := strings.ToLower(strings.TrimSpace(text))

	switch state {
	case report.StateConfirming:
		if lower == "confirm" {
			answers := svc.collector.GetAnswers(ctx, empID)
			cState, msg, err := svc.collector.Confirm(ctx, empID)
			if err != nil {
				slog.Error("confirm report", "employee_id", empID, "error", err)
				return sendReply("Error confirming report. Please try again.")
			}
			if cState == report.StateComplete && answers != nil {
				today := time.Now().In(svc.loc).Format("2006-01-02")
				if err := svc.reportDB.CreateReport(ctx, emp.TenantID, empID, today, answers); err != nil {
					slog.Error("save report", "employee_id", empID, "error", err)
					return sendReply("Report confirmed but failed to save. Please contact your manager.")
				}
				slog.Info("report saved", "employee_id", empID, "date", today)

				// Publish report submitted event
				_ = svc.eventBus.PublishPayload(ctx, events.ReportSubmitted, emp.TenantID, events.ReportSubmittedPayload{
					EmployeeID:   empID,
					EmployeeName: emp.Name,
					ReportDate:   today,
					Channel:      "telegram",
				})

				// Run async blocker/sentiment analysis
				go func(eid, tid, date string) {
					if err := svc.analyzer.Analyze(context.Background(), eid, date); err != nil {
						slog.Error("report analysis failed", "employee_id", eid, "error", err)
					}
				}(empID, emp.TenantID, today)

				// World Model extraction + trigger evaluation
				go func(tid, eid, eName string, ans map[string]string) {
					answersJSON, _ := json.Marshal(ans)
					if err := svc.wmExtractor.ExtractFromReport(context.Background(), tid, eid, string(answersJSON)); err != nil {
						slog.Error("world model extraction failed", "employee_id", eid, "error", err)
						return
					}
					evaluateWorldModelTriggers(context.Background(), tid, eid, eName, svc.queries, svc.recommender)
				}(emp.TenantID, empID, emp.Name, answers)
			}
			return sendReply(msg)
		}
		if lower == "edit" {
			_, firstQ, err := svc.collector.Start(ctx, empID)
			if err != nil {
				return sendReply("Error restarting. Please try again.")
			}
			return sendReply("Let's start over.\n\n" + firstQ)
		}
		return sendReply("Please reply 'confirm' to submit or 'edit' to start over.")

	case report.StateCollecting:
		cState, nextMsg, err := svc.collector.HandleAnswer(ctx, empID, text)
		if err != nil {
			slog.Error("handle answer", "employee_id", empID, "error", err)
			return sendReply("Error processing your answer. Please try again.")
		}
		_ = cState
		if nextMsg != "" {
			return sendReply(nextMsg)
		}

	default:
		// Mentor chat — idle state
		if svc.chatService == nil {
			return nil
		}
		svc.tgAdapter.Bot().Notify(tele.ChatID(senderID), tele.Typing)
		tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
		if err != nil {
			slog.Warn("mentor chat: get tenant", "error", err)
			return nil
		}
		resp, err := svc.chatService.HandleEmployee(ctx, brain.EmployeeChatRequest{
			EmployeeID:  empID,
			TenantID:    emp.TenantID,
			Name:        emp.Name,
			MentorID:    tenant.MentorID,
			CultureCode: emp.CultureCode,
			Text:        text,
		})
		if err != nil {
			slog.Error("mentor chat failed", "employee_id", empID, "error", err)
			return nil
		}
		if resp != "" {
			return sendReply(resp)
		}
	}

	return nil
}

// createUnifiedHandler creates the unified message handler for non-Telegram channels (Slack, Lark).
func createUnifiedHandler(svc *services) *channel.UnifiedHandler {
	return channel.NewUnifiedHandler(channel.UnifiedHandlerConfig{
		Queries: svc.queries,
		Sender:  svc.channelSender,
		OnText: func(ctx context.Context, employeeID, tenantID, text, channelType, empName, empJobTitle, empResponsibilities, empCountry, empLanguage, empCultureCode string) (string, error) {
			state := svc.collector.GetState(ctx, employeeID)
			lower := strings.ToLower(strings.TrimSpace(text))

			switch state {
			case report.StateConfirming:
				if lower == "confirm" {
					answers := svc.collector.GetAnswers(ctx, employeeID)
					cState, msg, err := svc.collector.Confirm(ctx, employeeID)
					if err != nil {
						return "Error confirming report. Please try again.", nil
					}
					if cState == report.StateComplete && answers != nil {
						today := time.Now().In(svc.loc).Format("2006-01-02")
						if err := svc.reportDB.CreateReport(ctx, tenantID, employeeID, today, answers); err != nil {
							return "Report confirmed but failed to save.", nil
						}
						_ = svc.eventBus.PublishPayload(ctx, events.ReportSubmitted, tenantID, events.ReportSubmittedPayload{
							EmployeeID:   employeeID,
							EmployeeName: "",
							ReportDate:   today,
							Channel:      channelType,
						})
						go func() {
							if err := svc.analyzer.Analyze(context.Background(), employeeID, today); err != nil {
								slog.Error("report analysis failed", "employee_id", employeeID, "error", err)
							}
						}()

						// World Model extraction + trigger evaluation
						go func(tid, eid, eName string, ans map[string]string) {
							answersJSON, _ := json.Marshal(ans)
							if err := svc.wmExtractor.ExtractFromReport(context.Background(), tid, eid, string(answersJSON)); err != nil {
								slog.Error("world model extraction failed", "employee_id", eid, "error", err)
								return
							}
							evaluateWorldModelTriggers(context.Background(), tid, eid, eName, svc.queries, svc.recommender)
						}(tenantID, employeeID, empName, answers)
					}
					return msg, nil
				}
				if lower == "edit" {
					_, firstQ, err := svc.collector.Start(ctx, employeeID)
					if err != nil {
						return "Error restarting. Please try again.", nil
					}
					return "Let's start over.\n\n" + firstQ, nil
				}
				return "Please reply 'confirm' to submit or 'edit' to start over.", nil

			case report.StateCollecting:
				_, nextMsg, err := svc.collector.HandleAnswer(ctx, employeeID, text)
				if err != nil {
					return "Error processing your answer. Please try again.", nil
				}
				return nextMsg, nil

			default:
				// Mentor chat — idle state
				if svc.chatService == nil {
					return "", nil
				}
				tenant, err := svc.botDB.GetTenantByBossChatID(ctx, svc.cfg.BossTelegramID)
				if err != nil {
					return "", nil
				}
				resp, err := svc.chatService.HandleEmployee(ctx, brain.EmployeeChatRequest{
					EmployeeID:       employeeID,
					TenantID:         tenantID,
					Name:             empName,
					JobTitle:         empJobTitle,
					Responsibilities: empResponsibilities,
					Country:          empCountry,
					Language:         empLanguage,
					MentorID:         tenant.MentorID,
					CultureCode:      empCultureCode,
					Text:             text,
				})
				if err != nil {
					slog.Error("unified mentor chat failed", "employee_id", employeeID, "error", err)
					return "", nil
				}
				return resp, nil
			}
		},
	})
}
