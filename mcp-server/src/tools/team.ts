import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";
import type { TeamStatus, TeamReport, Alerts } from "../types.js";

export async function getTeamStatus(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<TeamStatus>("/api/v1/openclaw/status");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getReport(
  client: ApiClient,
  period: string,
): Promise<CallToolResult> {
  if (period !== "weekly" && period !== "monthly") {
    return {
      content: [
        { type: "text", text: 'Period must be "weekly" or "monthly".' },
      ],
      isError: true,
    };
  }
  try {
    const data = await client.get<TeamReport>(
      `/api/v1/openclaw/report?period=${period}`,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getAlerts(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<Alerts>("/api/v1/openclaw/alerts");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}
