import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";
import type {
  SwitchMentorSuccess,
  SwitchMentorError,
  Mentor,
} from "../types.js";

export async function switchMentor(
  client: ApiClient,
  mentor: string,
): Promise<CallToolResult> {
  if (!mentor.trim()) {
    return {
      content: [{ type: "text", text: "Mentor name cannot be empty." }],
      isError: true,
    };
  }
  try {
    const data = await client.post<SwitchMentorSuccess | SwitchMentorError>(
      "/api/v1/openclaw/command",
      { command: `switch mentor ${mentor}` },
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function listMentors(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<Mentor[]>("/api/v1/seats/mentors");
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
