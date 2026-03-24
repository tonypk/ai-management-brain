import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";
import type {
  BoardDiscussResponse,
  SeatChatResponse,
  SeatChatInactiveResponse,
} from "../types.js";

const MAX_INPUT_LEN = 4000;

export async function boardDiscuss(
  client: ApiClient,
  topic: string,
): Promise<CallToolResult> {
  if (!topic.trim()) {
    return {
      content: [{ type: "text", text: "Topic cannot be empty." }],
      isError: true,
    };
  }
  if (topic.length > MAX_INPUT_LEN) {
    return {
      content: [
        {
          type: "text",
          text: `Topic too long (max ${MAX_INPUT_LEN} characters).`,
        },
      ],
      isError: true,
    };
  }
  try {
    const data = await client.post<BoardDiscussResponse>(
      "/api/v1/seats/board/discuss",
      { topic },
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function chatWithSeat(
  client: ApiClient,
  seatType: string,
  message: string,
): Promise<CallToolResult> {
  if (!message.trim()) {
    return {
      content: [{ type: "text", text: "Message cannot be empty." }],
      isError: true,
    };
  }
  if (message.length > MAX_INPUT_LEN) {
    return {
      content: [
        {
          type: "text",
          text: `Message too long (max ${MAX_INPUT_LEN} characters).`,
        },
      ],
      isError: true,
    };
  }
  try {
    const data = await client.post<
      SeatChatResponse | SeatChatInactiveResponse
    >("/api/v1/seats/chat", { seat_type: seatType, message });

    // Check if seat is inactive (response has "message" field instead of "response")
    if ("message" in data) {
      return {
        content: [{ type: "text", text: data.message }],
      };
    }

    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    if (error instanceof APIError && error.statusCode === 400) {
      return {
        content: [
          {
            type: "text",
            text: "Unknown seat type. Valid types: ceo, cfo, cmo, cto, chro, coo",
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
