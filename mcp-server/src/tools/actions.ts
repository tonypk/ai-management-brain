import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";
import type {
  CheckinResult,
  ChaseResult,
  SummaryActionResult,
  MessageResult,
} from "../types.js";

export async function sendCheckin(
  client: ApiClient,
  employeeName?: string,
): Promise<CallToolResult> {
  try {
    const body: Record<string, unknown> = {};
    if (employeeName && employeeName.trim()) {
      body.employee_name = employeeName.trim();
    }
    const data = await client.post<CheckinResult>(
      "/api/v1/openclaw/checkin",
      body,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function chaseEmployee(
  client: ApiClient,
  employeeName?: string,
): Promise<CallToolResult> {
  try {
    const body: Record<string, unknown> = {};
    if (employeeName && employeeName.trim()) {
      body.employee_name = employeeName.trim();
    }
    const data = await client.post<ChaseResult>(
      "/api/v1/openclaw/chase",
      body,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function sendSummary(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.post<SummaryActionResult>(
      "/api/v1/openclaw/summary",
      {},
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function sendMessage(
  client: ApiClient,
  employeeName: string,
  message: string,
): Promise<CallToolResult> {
  if (!employeeName || !employeeName.trim()) {
    return {
      content: [{ type: "text", text: "Employee name is required." }],
      isError: true,
    };
  }
  if (!message || !message.trim()) {
    return {
      content: [{ type: "text", text: "Message is required." }],
      isError: true,
    };
  }
  try {
    const data = await client.post<MessageResult>(
      "/api/v1/openclaw/message",
      {
        employee_name: employeeName.trim(),
        message: message.trim(),
      },
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
