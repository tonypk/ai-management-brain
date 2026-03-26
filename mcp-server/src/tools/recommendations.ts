import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";

export async function getRecommendations(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const recs = await client.get<unknown[]>("/api/v1/recommendations?status=pending");
    if (!Array.isArray(recs) || recs.length === 0) {
      return { content: [{ type: "text", text: "No pending recommendations." }] };
    }
    return {
      content: [{ type: "text", text: JSON.stringify(recs, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function executeRecommendation(
  client: ApiClient,
  args: { recommendation_id: string; action_index?: number },
): Promise<CallToolResult> {
  if (!args.recommendation_id) {
    return {
      content: [{ type: "text", text: "recommendation_id is required." }],
      isError: true,
    };
  }
  try {
    const data = await client.post<unknown>(
      `/api/v1/recommendations/${args.recommendation_id}/execute`,
      { action_index: args.action_index ?? 0 },
    );
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
