import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { ApiClient } from "./api-client.js";
import { getTeamStatus, getReport, getAlerts } from "./tools/team.js";
import { switchMentor, listMentors } from "./tools/mentor.js";
import { boardDiscuss, chatWithSeat } from "./tools/csuite.js";
import { listEmployees, getEmployeeProfile } from "./tools/employee.js";
import {
  sendCheckin,
  chaseEmployee,
  sendSummary,
  sendMessage,
} from "./tools/actions.js";
import {
  getRecommendations,
  executeRecommendation,
} from "./tools/recommendations.js";
import {
  getCompanyState,
  getExecutionSignals,
  getCommunicationEvents,
  getTopRisks,
  getWorkingMemory,
  getKPIDashboard,
  getOverdueTasks,
  getTaskStats,
  getIncentiveScores,
} from "./tools/brain.js";
import { getCompanyContext, updateContext } from "./tools/context.js";
import { getGoalState } from "./tools/goals.js";
import { createExecutionPlan } from "./tools/planning.js";
import { ingestMetric } from "./tools/metrics-write.js";
import { calculateIncentives } from "./tools/incentives-calc.js";
import {
  getSyncManifest,
  reportSyncResult,
  configureSync,
} from "./tools/sync.js";
import {
  startConsultingEngagement,
  answerConsultingQuestion,
  listConsultingEngagements,
  getConsultingEngagement,
  reviewConsultingAction,
  executeConsultingActions,
  checkConsultingProgress,
  closeConsultingEngagement,
  listConsultingActions,
} from "./tools/consulting.js";
import { getWorldModel, getEmployeeWorldModel } from "./tools/worldmodel.js";

const NO_KEY_MSG = "Please set MANAGEMENT_BRAIN_API_KEY environment variable.";

function makeClient(): ApiClient | null {
  const apiKey = process.env.MANAGEMENT_BRAIN_API_KEY ?? "";
  const baseUrl =
    process.env.MANAGEMENT_BRAIN_BASE_URL ?? "https://manageaibrain.com";
  if (!apiKey) return null;
  return new ApiClient(baseUrl, apiKey);
}

export function createServer(): McpServer {
  const server = new McpServer({
    name: "management-brain",
    version: "1.1.0",
  });

  // --- Group 1: Daily Operations (read) ---

  server.tool(
    "get_team_status",
    "Check today's daily check-in progress: how many submitted, who is still pending, and how many reminders have been sent. Use this when the user asks about today's team status, attendance, or who hasn't reported yet.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getTeamStatus(client);
    },
  );

  server.tool(
    "get_report",
    "Generate a team performance report with employee rankings by check-in rate and personalized 1:1 meeting suggestions. Use this to prepare for weekly/monthly reviews or to understand team trends over time.",
    {
      period: z
        .enum(["weekly", "monthly"])
        .describe(
          "Time period: 'weekly' for last 7 days, 'monthly' for last 30 days",
        ),
    },
    async ({ period }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getReport(client, period);
    },
  );

  server.tool(
    "get_alerts",
    "Get urgent alerts for employees who have missed check-ins for multiple consecutive days. Returns severity levels and missed day counts. Use this to identify team members who may need immediate attention or a wellness check.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getAlerts(client);
    },
  );

  // --- Group 2: Management Philosophy ---

  server.tool(
    "switch_mentor",
    "Change the AI management mentor that shapes all advice and analysis. Each mentor brings a distinct leadership philosophy: inamori (servant leadership), dalio (radical transparency), grove (OKRs), musk (first principles), jobs (product obsession), bezos (customer obsession), ma (ecosystem thinking), ren (wolf culture), son (300-year vision). Use this when the user wants different management perspectives.",
    {
      mentor: z
        .string()
        .describe(
          "Mentor ID: inamori, dalio, grove, musk, jobs, bezos, ma, ren, son, buffett, leijun, zhangyiming, caodewang, chushijian, meyer, trout",
        ),
    },
    async ({ mentor }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return switchMentor(client, mentor);
    },
  );

  server.tool(
    "list_mentors",
    "List all available management mentors with their names, companies, core philosophies, domain expertise, and recommended C-Suite seat configurations. Use this when the user wants to explore available mentors before switching.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return listMentors(client);
    },
  );

  // --- Group 3: AI C-Suite Board ---

  server.tool(
    "board_discuss",
    "Convene a virtual board meeting where AI-powered C-Suite executives (CEO, CFO, CMO, CTO, CHRO, COO) each analyze a topic from their domain expertise, followed by a unified synthesis. Use this for strategic decisions like market expansion, budget allocation, org restructuring, product launches, or any cross-functional question.",
    {
      topic: z
        .string()
        .min(1)
        .max(4000)
        .describe(
          "The strategic question or business topic for the board to discuss, e.g. 'Should we expand to the Japan market?' or 'How to reduce employee turnover by 20%?'",
        ),
    },
    async ({ topic }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return boardDiscuss(client, topic);
    },
  );

  server.tool(
    "chat_with_seat",
    "Have a direct conversation with one AI C-Suite executive. Each seat has deep domain expertise: CEO (strategy & vision), CFO (finance & budgets), CMO (marketing & growth), CTO (technology & architecture), CHRO (people & culture), COO (operations & efficiency). Use this for domain-specific questions rather than cross-functional topics.",
    {
      seat_type: z
        .string()
        .describe(
          "The C-Suite role to consult: ceo, cfo, cmo, cto, chro, or coo",
        ),
      message: z
        .string()
        .min(1)
        .max(4000)
        .describe(
          "Your question or topic for this executive, e.g. 'What's our burn rate outlook for Q3?' (CFO) or 'How should we structure the engineering team for microservices?' (CTO)",
        ),
    },
    async ({ seat_type, message }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return chatWithSeat(client, seat_type, message);
    },
  );

  // --- Group 4: People & Employees ---

  server.tool(
    "list_employees",
    "List all active employees with their names and roles. Use this to get an overview of the team composition or to find an employee's exact name before looking up their profile.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return listEmployees(client);
    },
  );

  server.tool(
    "get_employee_profile",
    "Look up a specific employee's detailed profile including check-in submission rate, sentiment trend over time, consecutive missed days, and recent daily reports. Supports fuzzy name matching. Use this to prepare for 1:1 meetings or to understand an individual's engagement and wellbeing.",
    {
      name: z
        .string()
        .describe(
          "Employee name to search for (case-insensitive, fuzzy match supported), e.g. 'John' or 'john doe'",
        ),
    },
    async ({ name }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getEmployeeProfile(client, name);
    },
  );

  // --- Group 5: Actions (write — sends messages via bot/channels) ---

  server.tool(
    "send_checkin",
    "Trigger daily check-in questions for all employees or a specific person. This SENDS messages via Telegram/Slack/Lark to employees, starting a check-in conversation. Use when the boss wants to manually trigger check-ins outside the scheduled time.",
    {
      employee_name: z
        .string()
        .optional()
        .describe(
          "Optional employee name (fuzzy match). If omitted, sends to ALL active employees who haven't submitted today.",
        ),
    },
    async ({ employee_name }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return sendCheckin(client, employee_name);
    },
  );

  server.tool(
    "chase_employee",
    "Send chase reminders to employees who haven't submitted their daily report. This SENDS reminder messages via their preferred channel. Use when the boss wants to nudge non-responders.",
    {
      employee_name: z
        .string()
        .optional()
        .describe(
          "Optional employee name (fuzzy match). If omitted, chases ALL employees who haven't submitted today.",
        ),
    },
    async ({ employee_name }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return chaseEmployee(client, employee_name);
    },
  );

  server.tool(
    "send_summary",
    "Generate today's team daily summary and send it to the boss via Telegram. Includes submission rate, key highlights, and blockers shaped by the active mentor's perspective. Use when the boss wants an immediate summary instead of waiting for the scheduled one.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return sendSummary(client);
    },
  );

  server.tool(
    "send_message",
    "Send a custom message to a specific employee via their preferred messaging channel (Telegram/Slack/Lark/Signal). Use this when the boss wants to communicate directly with a team member through the management system.",
    {
      employee_name: z
        .string()
        .describe(
          "Employee name (fuzzy match supported), e.g. 'John' or 'john doe'",
        ),
      message: z
        .string()
        .min(1)
        .max(4000)
        .describe(
          "The message to send, e.g. 'Hey, can we sync at 3pm?' or 'Great work on the release!'",
        ),
    },
    async ({ employee_name, message }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return sendMessage(client, employee_name, message);
    },
  );

  // --- Group 6: Brain Layer — Execution Intelligence ---

  server.tool(
    "get_company_state",
    "Get a comprehensive snapshot of the company's current operational state: top execution risks, overdue tasks, task status breakdown, communication event counts, blocked projects, and AI working memory. Use this for situational awareness before making management decisions.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getCompanyState(client);
    },
  );

  server.tool(
    "get_execution_signals",
    "Get AI-generated execution risk signals: overload risk, delivery risk, engagement drops, blocker cascades, performance spikes, and metric anomalies. Each signal has a severity score (0-1) and evidence-based reasons. Use this to identify which people or projects need attention.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getExecutionSignals(client);
    },
  );

  server.tool(
    "get_communication_events",
    "Get structured management events extracted from daily check-in reports: blockers reported, tasks completed, commitments made, delays, escalations, and proactive updates. Each event has a confidence score. Use this to understand team communication patterns and key signals.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getCommunicationEvents(client);
    },
  );

  server.tool(
    "get_top_risks",
    "Get the highest-severity execution risk signals across the organization. Returns signals sorted by score, highlighting the most urgent issues that need management attention. Use this for quick risk assessment.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getTopRisks(client);
    },
  );

  server.tool(
    "get_working_memory",
    "Get the AI's latest working memory snapshot: focus areas, risk summary, team momentum (positive/neutral/negative), pending decisions, recent wins, and action items. This is the AI manager's situational awareness context.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getWorkingMemory(client);
    },
  );

  server.tool(
    "get_kpi_dashboard",
    "Get all KPI metrics with their latest values and targets. Shows metric name, unit, current value, target value, and owner. Use this to monitor business performance and identify metrics that are off-track.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getKPIDashboard(client);
    },
  );

  server.tool(
    "get_overdue_tasks",
    "Get all tasks that are past their due date. Shows task title, priority, assignee, and how overdue they are. Use this to identify delivery risks and follow up with task owners.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getOverdueTasks(client);
    },
  );

  server.tool(
    "get_task_stats",
    "Get task status breakdown: counts of tasks in each status (todo, in_progress, in_review, done, blocked). Use this for a quick pulse on team workload and throughput.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getTaskStats(client);
    },
  );

  server.tool(
    "get_incentive_scores",
    "Get incentive evaluation scores for a specific period. Shows per-employee scores, breakdowns, payout weights, and whether human review is needed. Use this for compensation decisions and performance recognition.",
    {
      period: z
        .string()
        .describe("Period in YYYY-MM format, e.g. '2026-03' for March 2026"),
    },
    async ({ period }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getIncentiveScores(client, period);
    },
  );

  // --- Group: AI Recommendations ---

  server.tool(
    "get_recommendations",
    "Get pending AI management recommendations with suggested actions. Recommendations are generated from daily AI scans (enriched with employee memory patterns) and real-time triggers (check-in sentiment, memory-based stress/blocker/growth patterns). Each includes title, description, priority, category, evidence (including memory_evidence), and executable actions.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getRecommendations(client);
    },
  );

  server.tool(
    "execute_recommendation",
    "Execute a specific action on an AI recommendation. Takes a recommendation ID and optional action index (defaults to the first action). Returns the execution result including success status and any messages.",
    {
      recommendation_id: z
        .string()
        .describe("The recommendation UUID to execute"),
      action_index: z
        .number()
        .optional()
        .describe("Index of the action to execute (default 0)"),
    },
    async ({ recommendation_id, action_index }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return executeRecommendation(client, { recommendation_id, action_index });
    },
  );

  // --- Group 7: Brain Layer v3 — Context, Goals, Planning, Metrics, Incentives ---

  server.tool(
    "get_company_context",
    "Get the complete company context: organization profile, strategic priorities, key risks, team composition, active goals, KPI metrics, top execution risks, and HR insights. This is the foundation for all management reasoning — call this before making recommendations.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getCompanyContext(client);
    },
  );

  server.tool(
    "update_context",
    "Update company context: strategic priorities, key risks, or management style weights. Use this during onboarding or when the boss shares new strategic information. Only the fields provided will be updated; omitted fields are left unchanged.",
    {
      updates: z
        .object({
          strategic_priorities: z
            .array(z.string())
            .optional()
            .describe(
              "List of strategic priorities, e.g. ['Increase ARR by 30%', 'Launch APAC']",
            ),
          key_risks: z
            .array(z.string())
            .optional()
            .describe(
              "List of key risks, e.g. ['Key person dependency on Alice', 'Cash runway < 6 months']",
            ),
          management_style_weights: z
            .record(z.string(), z.number())
            .optional()
            .describe(
              "Management style weight map, e.g. { 'inamori': 0.7, 'grove': 0.3 }",
            ),
        })
        .describe("Fields to update on the organization context"),
    },
    async ({ updates }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return updateContext(client, updates as Record<string, unknown>);
    },
  );

  server.tool(
    "get_goal_state",
    "Get OKR and KPI progress: all goals with linked key results, current metric values vs targets, completion percentages, and owners. Use this to understand strategic alignment and goal health.",
    {
      cycle: z
        .string()
        .describe(
          "Goal cycle identifier, e.g. '2026-Q1' or '2026-H1'. Required by the API.",
        ),
    },
    async ({ cycle }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getGoalState(client, cycle);
    },
  );

  server.tool(
    "create_execution_plan",
    "Generate a prioritized action plan based on current company context, goals, signals, and metrics. Returns recommended next actions with owners, priorities, deadlines, and evidence-based reasoning. Requires ANTHROPIC_API_KEY on the server. Use this for proactive management planning.",
    {
      focus_area: z
        .string()
        .optional()
        .describe(
          "Optional focus area to narrow the plan: 'risks', 'goals', 'tasks', or 'overall' (default)",
        ),
    },
    async ({ focus_area }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return createExecutionPlan(client, { focus_area });
    },
  );

  server.tool(
    "ingest_metric",
    "Record a KPI data point. Use this to import metric values from external sources (spreadsheets, reports, dashboards). Specify the metric UUID and the observed value.",
    {
      metric_id: z.string().describe("Metric UUID (from get_kpi_dashboard)"),
      value: z.number().describe("The observed numeric value"),
      observed_at: z
        .string()
        .optional()
        .describe(
          "ISO 8601 timestamp of when the value was observed, defaults to now",
        ),
      source: z
        .string()
        .optional()
        .describe("Data source label, e.g. 'sheets', 'manual', 'api'"),
    },
    async ({ metric_id, value, observed_at, source }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return ingestMetric(client, { metric_id, value, observed_at, source });
    },
  );

  server.tool(
    "calculate_incentives",
    "Calculate incentive scores for all active employees in a given period using active incentive rules, execution data, goal attribution, and communication quality. Returns per-employee scores with breakdowns and human-review flags. Requires ANTHROPIC_API_KEY on the server.",
    {
      period: z
        .string()
        .describe("Period in YYYY-MM format, e.g. '2026-03' for March 2026"),
    },
    async ({ period }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return calculateIncentives(client, { period });
    },
  );

  // --- Group 8: Sync Tools ---

  server.tool(
    "get_sync_manifest",
    "Get a list of data changes since the last sync. Returns changed tasks, goals, projects, and metrics with their current values and timestamps.",
    {
      storage_type: z
        .enum(["notion", "sheets"])
        .describe("Which storage to get manifest for"),
    },
    async ({ storage_type }) => {
      const client = makeClient();
      if (!client)
        return {
          content: [{ type: "text", text: NO_KEY_MSG }],
          isError: true,
        };
      return getSyncManifest(client, storage_type);
    },
  );

  server.tool(
    "report_sync_result",
    "Report the result of a sync operation. Call this after completing a sync to record stats and write pulled items back to manageaibrain.",
    {
      storage_type: z.enum(["notion", "sheets"]),
      items_pushed: z
        .number()
        .describe("Number of items pushed to external storage"),
      items_pulled: z
        .number()
        .describe("Number of items pulled from external storage"),
      conflicts: z.number().describe("Number of conflicts detected"),
      errors: z.array(z.string()).optional().describe("Error messages if any"),
      pulled_items: z
        .array(
          z.object({
            entity_type: z.string(),
            external_id: z.string(),
            data: z.record(z.string(), z.any()),
          }),
        )
        .optional()
        .describe(
          "Items pulled from external storage to update in manageaibrain",
        ),
    },
    async ({
      storage_type,
      items_pushed,
      items_pulled,
      conflicts,
      errors,
      pulled_items,
    }) => {
      const client = makeClient();
      if (!client)
        return {
          content: [{ type: "text", text: NO_KEY_MSG }],
          isError: true,
        };
      return reportSyncResult(client, {
        storage_type,
        items_pushed,
        items_pulled,
        conflicts,
        errors,
        pulled_items,
      });
    },
  );

  server.tool(
    "configure_sync",
    "Configure sync settings for Notion or Google Sheets. Set which data types to sync, frequency, and storage-specific config.",
    {
      storage_type: z.enum(["notion", "sheets"]),
      is_enabled: z.boolean(),
      entity_types: z.array(z.enum(["tasks", "goals", "projects", "metrics"])),
      sync_frequency_minutes: z
        .number()
        .optional()
        .describe("Sync interval in minutes, default 30"),
      config: z
        .record(z.string(), z.any())
        .optional()
        .describe("Storage-specific config: Notion database IDs or Sheet IDs"),
    },
    async (params) => {
      const client = makeClient();
      if (!client)
        return {
          content: [{ type: "text", text: NO_KEY_MSG }],
          isError: true,
        };
      return configureSync(client, params);
    },
  );

  // --- Group 9: AI Consulting Engine ---

  server.tool(
    "start_consulting",
    "Start a new AI management consulting engagement. Describe a business problem and the AI consultant will classify its complexity, assign a category, and begin a structured diagnostic conversation. Like hiring McKinsey — the AI asks focused questions to diagnose root causes before recommending actions.",
    {
      problem: z
        .string()
        .min(1)
        .max(4000)
        .describe(
          "The business problem or challenge to consult on, e.g. 'Our sales team is underperforming this quarter' or 'We need to restructure the engineering organization for scale'",
        ),
      mentor_id: z
        .string()
        .optional()
        .describe(
          "Optional mentor to shape the consulting style (inamori, dalio, grove, musk, etc.)",
        ),
      culture_code: z
        .string()
        .optional()
        .describe(
          "Optional culture code (default, philippines, singapore, etc.)",
        ),
    },
    async ({ problem, mentor_id, culture_code }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return startConsultingEngagement(client, {
        problem,
        mentor_id,
        culture_code,
      });
    },
  );

  server.tool(
    "answer_consulting_question",
    "Answer a diagnostic question from the AI consultant during an active consulting engagement. The consultant will either ask the next question or, when enough information is gathered, automatically generate a root cause analysis and action plan.",
    {
      engagement_id: z.string().describe("The consulting engagement UUID"),
      answer: z
        .string()
        .min(1)
        .max(4000)
        .describe("Your answer to the consultant's diagnostic question"),
    },
    async ({ engagement_id, answer }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return answerConsultingQuestion(client, { engagement_id, answer });
    },
  );

  server.tool(
    "list_consulting_engagements",
    "List all consulting engagements (active and closed). Shows engagement title, tier, category, current phase, and progress percentage.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return listConsultingEngagements(client);
    },
  );

  server.tool(
    "get_consulting_engagement",
    "Get full details of a specific consulting engagement including diagnosis data, analysis, plan, and progress.",
    {
      engagement_id: z.string().describe("The consulting engagement UUID"),
    },
    async ({ engagement_id }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getConsultingEngagement(client, { engagement_id });
    },
  );

  server.tool(
    "review_consulting_action",
    "Approve or reject a specific action in a consulting engagement's action plan. Each action must be individually reviewed before it can be executed.",
    {
      action_id: z.string().describe("The engagement action UUID to review"),
      approved: z
        .boolean()
        .describe("true to approve the action, false to reject it"),
    },
    async ({ action_id, approved }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return reviewConsultingAction(client, { action_id, approved });
    },
  );

  server.tool(
    "execute_consulting_actions",
    "Execute all approved actions for a consulting engagement. This dispatches the actions (create tasks, schedule meetings, send messages, flag risks) and moves the engagement to tracking phase.",
    {
      engagement_id: z
        .string()
        .describe(
          "The consulting engagement UUID whose approved actions to execute",
        ),
    },
    async ({ engagement_id }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return executeConsultingActions(client, { engagement_id });
    },
  );

  server.tool(
    "check_consulting_progress",
    "Check progress on an active consulting engagement. Returns a progress report with completion percentage, what's on track, what's at risk, and recommended next steps.",
    {
      engagement_id: z.string().describe("The consulting engagement UUID"),
    },
    async ({ engagement_id }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return checkConsultingProgress(client, { engagement_id });
    },
  );

  server.tool(
    "close_consulting_engagement",
    "Close a consulting engagement with a retrospective. Generates an effectiveness assessment, lessons learned, and stores insights as organizational memory for future engagements.",
    {
      engagement_id: z
        .string()
        .describe("The consulting engagement UUID to close"),
    },
    async ({ engagement_id }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return closeConsultingEngagement(client, { engagement_id });
    },
  );

  server.tool(
    "list_consulting_actions",
    "List all actions for a consulting engagement with their current status (pending, approved, rejected, done, failed) and linked task/meeting IDs.",
    {
      engagement_id: z.string().describe("The consulting engagement UUID"),
    },
    async ({ engagement_id }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return listConsultingActions(client, { engagement_id });
    },
  );

  // --- Group 10: Team World Model ---

  server.tool(
    "get_world_model",
    "Get the team's World Model — skills, collaborations, blockers, growth events, and AI insights extracted from daily check-ins. Use this to understand the team's collective capabilities, collaboration patterns, and recurring challenges.",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getWorldModel(client);
    },
  );

  server.tool(
    "get_employee_world_model",
    "Get a specific employee's World Model — skills, growth trajectory, blockers, and collaborations. Use this to prepare for 1:1s or understand an individual's development.",
    { name: z.string().describe("Employee name (fuzzy match)") },
    async ({ name }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getEmployeeWorldModel(client, name);
    },
  );

  return server;
}
