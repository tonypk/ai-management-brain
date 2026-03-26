import { ApiClient, APIError } from "../api-client.js";
import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}

export async function calculateIncentives(
  client: ApiClient,
  params: { period: string },
): Promise<CallToolResult> {
  try {
    const data = await client.post<Record<string, unknown>[]>(
      "/api/v1/openclaw/incentives/calculate",
      { period: params.period },
    );
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  } catch (error) {
    return errorResult(error);
  }
}
