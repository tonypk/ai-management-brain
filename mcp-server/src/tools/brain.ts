import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";
import type {
  CompanyState,
  ExecutionSignal,
  CommunicationEvent,
  MetricWithValue,
  IncentiveScore,
} from "../types.js";

export async function getCompanyState(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<CompanyState>("/api/v1/openclaw/state");
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getExecutionSignals(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<ExecutionSignal[]>(
      "/api/v1/openclaw/state/signals?limit=20",
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getCommunicationEvents(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<CommunicationEvent[]>(
      "/api/v1/openclaw/state/events?limit=20",
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getTopRisks(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<ExecutionSignal[]>(
      "/api/v1/openclaw/state/risks",
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getWorkingMemory(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<Record<string, unknown>>(
      "/api/v1/openclaw/state/memory",
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getKPIDashboard(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<MetricWithValue[]>(
      "/api/v1/openclaw/kpis",
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getOverdueTasks(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<Record<string, unknown>[]>(
      "/api/v1/openclaw/tasks/overdue",
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getTaskStats(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<Record<string, unknown>[]>(
      "/api/v1/openclaw/tasks/stats",
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getIncentiveScores(
  client: ApiClient,
  period: string,
): Promise<CallToolResult> {
  try {
    const data = await client.get<IncentiveScore[]>(
      `/api/v1/openclaw/incentives/scores?period=${encodeURIComponent(period)}`,
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
