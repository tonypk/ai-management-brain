# MCP Tools Reference

All 33 MCP tools for Team Operations Mode. Use these directly — no manual API calls needed.

## Read Tools — Daily Operations (9)

| Tool | What it does |
|------|-------------|
| `get_team_status` | Today's check-in progress: submitted, pending, reminders sent |
| `get_report` | Weekly/monthly performance report with rankings and 1:1 suggestions |
| `get_alerts` | Alerts for employees with consecutive missed check-ins |
| `switch_mentor` | Change active management mentor philosophy |
| `list_mentors` | List all 16 mentors with expertise and recommended C-Suite seats |
| `board_discuss` | Convene AI C-Suite board meeting (CEO/CFO/CMO/CTO/CHRO/COO) on any topic |
| `chat_with_seat` | Direct conversation with one AI C-Suite executive |
| `list_employees` | List all active employees with roles |
| `get_employee_profile` | Employee profile with sentiment trend and submission history |

## Read Tools — Execution Intelligence (9)

| Tool | What it does |
|------|-------------|
| `get_company_state` | Full operational snapshot: risks, overdue tasks, event counts, blocked projects, working memory |
| `get_execution_signals` | AI-generated risk signals: overload, delivery, engagement, blockers, spikes, anomalies |
| `get_communication_events` | Structured events extracted from check-ins: blockers, completions, commitments, delays |
| `get_top_risks` | Highest-severity execution risks sorted by urgency score |
| `get_working_memory` | AI's situational awareness: focus areas, momentum, pending decisions, action items |
| `get_kpi_dashboard` | All KPI metrics with latest values vs targets |
| `get_overdue_tasks` | Tasks past their due date with priority and assignee |
| `get_task_stats` | Task status breakdown: todo, in_progress, in_review, done, blocked |
| `get_incentive_scores` | Per-employee incentive scores for a period with breakdowns and review flags |

## Read Tools — Brain Context (3)

| Tool | What it does |
|------|-------------|
| `get_company_context` | Complete company context: organization profile, strategic priorities, key risks, team composition, HR insights — the foundation for all management reasoning |
| `get_goal_state` | OKR and KPI progress: goals with linked key results, metric values vs targets, completion percentages, owners |
| `create_execution_plan` | Generate a prioritized action plan based on current context, goals, signals, and metrics with evidence-based reasoning |

## Write Tools (4 — sends messages to employees)

| Tool | What it does |
|------|-------------|
| `send_checkin` | Trigger daily check-in questions for all or a specific employee |
| `chase_employee` | Send chase reminders to employees who haven't submitted today |
| `send_summary` | Generate and send today's team daily summary to the boss |
| `send_message` | Send a custom message to an employee via their preferred channel |

Write tools actively send messages via Telegram/Slack/Lark/Signal.

## Write Tools — Context (2)

| Tool | What it does |
|------|-------------|
| `ingest_metric` | Record a KPI data point from external sources (spreadsheets, reports, dashboards) |
| `update_context` | Update company context: strategic priorities, key risks, management style weights |

## AI Recommendations (2)

| Tool | What it does |
|------|-------------|
| `get_recommendations` | Get pending AI management recommendations with suggested actions, priority, evidence |
| `execute_recommendation` | Execute a specific action on a recommendation (send message, schedule meeting, etc.) |

The recommendation engine runs a daily scan (10:30 AM) analyzing team data through the active mentor's lens, plus real-time triggers on events like consecutive missed check-ins, sentiment drops, and overdue tasks.

## Write Tools — Incentives (1)

| Tool | What it does |
|------|-------------|
| `calculate_incentives` | Calculate incentive scores for all employees in a given period using execution data, goal attribution, and active rules |

## Sync Tools (3 — bidirectional Notion/Sheets sync)

| Tool | What it does |
|------|-------------|
| `get_sync_manifest` | Get data changes since last sync — returns changed tasks, goals, projects, metrics for push to Notion/Sheets |
| `report_sync_result` | Report sync completion — records stats (items pushed/pulled/conflicts) and writes pulled items back |
| `configure_sync` | Configure sync settings: storage type (Notion/Sheets), entity types, frequency, storage-specific config |

## Tool Categories Summary

| Category | Count | Impact |
|----------|-------|--------|
| Read — Daily Ops | 9 | Query only |
| Read — Intelligence | 9 | Query only |
| Read — Brain Context | 3 | Query only |
| Write — Messages | 4 | Sends messages to employees |
| Write — Context | 2 | Updates company data |
| AI Recommendations | 2 | Suggests + executes management actions |
| Write — Incentives | 1 | Calculates scores |
| Sync | 3 | Reads/writes Notion/Sheets |
| **Total** | **33** | |
