import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";

// --- Consulting Engagement Tools ---

export async function startConsultingEngagement(
  client: ApiClient,
  args: { problem: string; mentor_id?: string; culture_code?: string },
): Promise<CallToolResult> {
  if (!args.problem) {
    return {
      content: [{ type: "text", text: "problem is required." }],
      isError: true,
    };
  }
  try {
    const data = await client.post<unknown>("/api/v1/consulting/start", {
      problem: args.problem,
      mentor_id: args.mentor_id ?? "",
      culture_code: args.culture_code ?? "",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function answerConsultingQuestion(
  client: ApiClient,
  args: { engagement_id: string; answer: string },
): Promise<CallToolResult> {
  if (!args.engagement_id || !args.answer) {
    return {
      content: [{ type: "text", text: "engagement_id and answer are required." }],
      isError: true,
    };
  }
  try {
    const data = await client.post<unknown>(
      `/api/v1/consulting/${args.engagement_id}/answer`,
      { answer: args.answer },
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function listConsultingEngagements(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<unknown[]>("/api/v1/consulting");
    if (!Array.isArray(data) || data.length === 0) {
      return { content: [{ type: "text", text: "No consulting engagements found." }] };
    }
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getConsultingEngagement(
  client: ApiClient,
  args: { engagement_id: string },
): Promise<CallToolResult> {
  if (!args.engagement_id) {
    return {
      content: [{ type: "text", text: "engagement_id is required." }],
      isError: true,
    };
  }
  try {
    const data = await client.get<unknown>(
      `/api/v1/consulting/${args.engagement_id}`,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function reviewConsultingAction(
  client: ApiClient,
  args: { action_id: string; approved: boolean },
): Promise<CallToolResult> {
  if (!args.action_id) {
    return {
      content: [{ type: "text", text: "action_id is required." }],
      isError: true,
    };
  }
  try {
    const data = await client.post<unknown>(
      `/api/v1/consulting/actions/${args.action_id}/review`,
      { approved: args.approved },
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function executeConsultingActions(
  client: ApiClient,
  args: { engagement_id: string },
): Promise<CallToolResult> {
  if (!args.engagement_id) {
    return {
      content: [{ type: "text", text: "engagement_id is required." }],
      isError: true,
    };
  }
  try {
    const data = await client.post<unknown>(
      `/api/v1/consulting/${args.engagement_id}/execute`,
      {},
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function checkConsultingProgress(
  client: ApiClient,
  args: { engagement_id: string },
): Promise<CallToolResult> {
  if (!args.engagement_id) {
    return {
      content: [{ type: "text", text: "engagement_id is required." }],
      isError: true,
    };
  }
  try {
    const data = await client.get<unknown>(
      `/api/v1/consulting/${args.engagement_id}/progress`,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function closeConsultingEngagement(
  client: ApiClient,
  args: { engagement_id: string },
): Promise<CallToolResult> {
  if (!args.engagement_id) {
    return {
      content: [{ type: "text", text: "engagement_id is required." }],
      isError: true,
    };
  }
  try {
    const data = await client.post<unknown>(
      `/api/v1/consulting/${args.engagement_id}/close`,
      {},
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function listConsultingActions(
  client: ApiClient,
  args: { engagement_id: string },
): Promise<CallToolResult> {
  if (!args.engagement_id) {
    return {
      content: [{ type: "text", text: "engagement_id is required." }],
      isError: true,
    };
  }
  try {
    const data = await client.get<unknown>(
      `/api/v1/consulting/${args.engagement_id}/actions`,
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
