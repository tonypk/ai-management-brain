import { ApiClient, APIError } from "../api-client.js";
import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}

export async function ingestMetric(
  client: ApiClient,
  params: {
    metric_id: string;
    value: number;
    observed_at?: string;
    source?: string;
  },
): Promise<CallToolResult> {
  try {
    const body: Record<string, unknown> = { value: params.value };
    if (params.observed_at) body.observed_at = params.observed_at;
    if (params.source) body.source_ref = params.source;

    const data = await client.post<Record<string, unknown>>(
      `/api/v1/openclaw/kpis/${params.metric_id}/values`,
      body,
    );
    return { content: [{ type: "text", text: JSON.stringify(data, null, 2) }] };
  } catch (error) {
    return errorResult(error);
  }
}
