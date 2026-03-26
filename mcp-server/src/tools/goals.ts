import { ApiClient, APIError } from "../api-client.js";
import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}

export async function getGoalState(
  client: ApiClient,
  cycle?: string,
): Promise<CallToolResult> {
  try {
    const params = cycle ? `?cycle=${encodeURIComponent(cycle)}` : "";
    const data = await client.get<Record<string, unknown>[]>(
      `/api/v1/openclaw/goals${params}`,
    );
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  } catch (error) {
    return errorResult(error);
  }
}
