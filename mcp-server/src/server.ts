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

const NO_KEY_MSG =
  "Please set MANAGEMENT_BRAIN_API_KEY environment variable.";

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
    version: "1.0.0",
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
        .describe("Time period: 'weekly' for last 7 days, 'monthly' for last 30 days"),
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
        .describe("The C-Suite role to consult: ceo, cfo, cmo, cto, chro, or coo"),
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
        .describe(
          "Period in YYYY-MM format, e.g. '2026-03' for March 2026",
        ),
    },
    async ({ period }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getIncentiveScores(client, period);
    },
  );

  return server;
}
