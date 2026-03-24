import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { ApiClient } from "./api-client.js";
import { getTeamStatus, getReport, getAlerts } from "./tools/team.js";
import { switchMentor, listMentors } from "./tools/mentor.js";
import { boardDiscuss, chatWithSeat } from "./tools/csuite.js";
import { listEmployees, getEmployeeProfile } from "./tools/employee.js";

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

  // --- Group 1: Core Management ---

  server.tool(
    "get_team_status",
    "Get today's team check-in status — submission rate, pending employees, chase counts",
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
    "Get team performance report with ranking and 1:1 suggestions",
    { period: z.enum(["weekly", "monthly"]).describe("Report period") },
    async ({ period }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getReport(client, period);
    },
  );

  server.tool(
    "get_alerts",
    "Get active alerts for employees with consecutive missed check-in days",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getAlerts(client);
    },
  );

  // --- Group 2: Mentors ---

  server.tool(
    "switch_mentor",
    'Switch the active management mentor philosophy (e.g., "musk", "inamori")',
    {
      mentor: z
        .string()
        .describe('Mentor ID or name, e.g. "musk", "inamori", "dalio"'),
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
    "List all available mentors with domain expertise and recommended C-Suite seats",
    {},
    async () => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return listMentors(client);
    },
  );

  // --- Group 3: C-Suite ---

  server.tool(
    "board_discuss",
    "Run a board discussion across all active C-Suite seats on a topic. Each seat responds from their expertise, followed by a synthesis.",
    {
      topic: z
        .string()
        .min(1)
        .max(4000)
        .describe("The topic for the board to discuss"),
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
    "Chat directly with a specific C-Suite seat (e.g., ask the CFO about budget)",
    {
      seat_type: z
        .string()
        .describe(
          'C-Suite seat type, e.g. "ceo", "cfo", "cmo", "cto", "chro", "coo"',
        ),
      message: z
        .string()
        .min(1)
        .max(4000)
        .describe("Your message to the C-Suite seat"),
    },
    async ({ seat_type, message }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return chatWithSeat(client, seat_type, message);
    },
  );

  // --- Group 4: Employees ---

  server.tool(
    "list_employees",
    "List all active employees with their roles",
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
    "Get an employee's profile with submission history, sentiment trends, and recent reports",
    {
      name: z
        .string()
        .describe("Employee name (case-insensitive fuzzy match)"),
    },
    async ({ name }) => {
      const client = makeClient();
      if (!client)
        return { content: [{ type: "text", text: NO_KEY_MSG }], isError: true };
      return getEmployeeProfile(client, name);
    },
  );

  return server;
}
