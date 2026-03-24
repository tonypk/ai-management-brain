import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";
import type { CommandResult, EmployeeProfile } from "../types.js";

export async function listEmployees(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.post<CommandResult>(
      "/api/v1/openclaw/command",
      { command: "list employees" },
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getEmployeeProfile(
  client: ApiClient,
  name: string,
): Promise<CallToolResult> {
  if (!name.trim()) {
    return {
      content: [{ type: "text", text: "Employee name cannot be empty." }],
      isError: true,
    };
  }
  try {
    const data = await client.get<EmployeeProfile>(
      `/api/v1/employees/profile/${encodeURIComponent(name)}`,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    if (error instanceof APIError && error.statusCode === 404) {
      return {
        content: [
          {
            type: "text",
            text: `No employee found matching '${name}'.`,
          },
        ],
        isError: true,
      };
    }
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
