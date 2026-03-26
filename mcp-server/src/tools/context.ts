import { ApiClient, APIError } from "../api-client.js";
import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}

export async function getCompanyContext(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<Record<string, unknown>>(
      "/api/v1/openclaw/context",
    );
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  } catch (error) {
    return errorResult(error);
  }
}

export async function updateContext(
  client: ApiClient,
  updates: Record<string, unknown>,
): Promise<CallToolResult> {
  try {
    const data = await client.put<Record<string, unknown>>(
      "/api/v1/openclaw/context",
      updates,
    );
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  } catch (error) {
    return errorResult(error);
  }
}
