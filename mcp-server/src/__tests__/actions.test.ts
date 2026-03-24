import { describe, it, expect, vi, afterEach } from "vitest";
import { ApiClient } from "../api-client.js";
import {
  sendCheckin,
  chaseEmployee,
  sendSummary,
  sendMessage,
} from "../tools/actions.js";

function mockClient(response: unknown): ApiClient {
  globalThis.fetch = vi.fn().mockResolvedValue({
    status: 200,
    json: () => Promise.resolve(response),
  });
  return new ApiClient("https://example.com", "test-key");
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("sendCheckin", () => {
  it("sends to all employees when no name given", async () => {
    const client = mockClient({
      sent_to: ["Alice", "Bob"],
      skipped: ["Carol (already submitted)"],
    });
    const result = await sendCheckin(client);
    expect(result.isError).toBeUndefined();
    const parsed = JSON.parse((result.content[0] as { text: string }).text);
    expect(parsed.sent_to).toEqual(["Alice", "Bob"]);
    expect(parsed.skipped).toEqual(["Carol (already submitted)"]);
  });

  it("sends to specific employee when name given", async () => {
    const client = mockClient({ sent_to: ["Alice"], skipped: [] });
    const result = await sendCheckin(client, "Alice");
    expect(result.isError).toBeUndefined();
    const parsed = JSON.parse((result.content[0] as { text: string }).text);
    expect(parsed.sent_to).toEqual(["Alice"]);
  });

  it("handles API error", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 400,
      json: () => Promise.resolve({ error: "no employee found matching 'xyz'" }),
    });
    const client = new ApiClient("https://example.com", "test-key");
    const result = await sendCheckin(client, "xyz");
    expect(result.isError).toBe(true);
  });
});

describe("chaseEmployee", () => {
  it("chases all when no name given", async () => {
    const client = mockClient({
      chased: ["John (step 2)"],
      skipped: ["Alice (already submitted)"],
    });
    const result = await chaseEmployee(client);
    expect(result.isError).toBeUndefined();
    const parsed = JSON.parse((result.content[0] as { text: string }).text);
    expect(parsed.chased).toEqual(["John (step 2)"]);
  });

  it("chases specific employee", async () => {
    const client = mockClient({ chased: ["John"], skipped: [] });
    const result = await chaseEmployee(client, "John");
    expect(result.isError).toBeUndefined();
  });
});

describe("sendSummary", () => {
  it("returns summary with submission rate", async () => {
    const client = mockClient({
      summary: "Team had a productive day...",
      submission_rate: 0.83,
      sent_to: "boss",
    });
    const result = await sendSummary(client);
    expect(result.isError).toBeUndefined();
    const parsed = JSON.parse((result.content[0] as { text: string }).text);
    expect(parsed.submission_rate).toBe(0.83);
    expect(parsed.sent_to).toBe("boss");
  });
});

describe("sendMessage", () => {
  it("rejects empty employee name", async () => {
    const client = mockClient({});
    const result = await sendMessage(client, "", "hello");
    expect(result.isError).toBe(true);
    expect((result.content[0] as { text: string }).text).toContain("Employee name");
  });

  it("rejects empty message", async () => {
    const client = mockClient({});
    const result = await sendMessage(client, "John", "");
    expect(result.isError).toBe(true);
    expect((result.content[0] as { text: string }).text).toContain("Message");
  });

  it("rejects whitespace-only employee name", async () => {
    const client = mockClient({});
    const result = await sendMessage(client, "   ", "hello");
    expect(result.isError).toBe(true);
  });

  it("sends message successfully", async () => {
    const client = mockClient({
      sent_to: "John Santos",
      channel: "telegram",
    });
    const result = await sendMessage(client, "John", "Hey, can we sync at 3pm?");
    expect(result.isError).toBeUndefined();
    const parsed = JSON.parse((result.content[0] as { text: string }).text);
    expect(parsed.sent_to).toBe("John Santos");
    expect(parsed.channel).toBe("telegram");
  });
});
