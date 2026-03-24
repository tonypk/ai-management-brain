import { describe, it, expect, vi, afterEach } from "vitest";
import { ApiClient } from "../api-client.js";
import { getTeamStatus, getReport } from "../tools/team.js";
import { switchMentor } from "../tools/mentor.js";
import { boardDiscuss, chatWithSeat } from "../tools/csuite.js";
import { getEmployeeProfile } from "../tools/employee.js";

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

describe("team tools", () => {
  it("getTeamStatus returns formatted response", async () => {
    const client = mockClient({ date: "2026-03-24", submitted: 4 });
    const result = await getTeamStatus(client);
    expect(result.isError).toBeUndefined();
    expect(result.content[0].type).toBe("text");
    const parsed = JSON.parse((result.content[0] as { text: string }).text);
    expect(parsed.submitted).toBe(4);
  });

  it("getReport validates period", async () => {
    const client = mockClient({});
    const result = await getReport(client, "invalid");
    expect(result.isError).toBe(true);
  });
});

describe("mentor tools", () => {
  it("switchMentor rejects empty input", async () => {
    const client = mockClient({});
    const result = await switchMentor(client, "  ");
    expect(result.isError).toBe(true);
    expect((result.content[0] as { text: string }).text).toContain("empty");
  });
});

describe("csuite tools", () => {
  it("boardDiscuss rejects empty topic", async () => {
    const client = mockClient({});
    const result = await boardDiscuss(client, "");
    expect(result.isError).toBe(true);
  });

  it("boardDiscuss rejects oversized topic", async () => {
    const client = mockClient({});
    const result = await boardDiscuss(client, "x".repeat(4001));
    expect(result.isError).toBe(true);
    expect((result.content[0] as { text: string }).text).toContain("4000");
  });

  it("chatWithSeat rejects empty message", async () => {
    const client = mockClient({});
    const result = await chatWithSeat(client, "ceo", "");
    expect(result.isError).toBe(true);
  });

  it("chatWithSeat handles inactive seat", async () => {
    const client = mockClient({
      data: { message: "The CEO seat is currently inactive." },
    });
    const result = await chatWithSeat(client, "ceo", "hello");
    expect(result.isError).toBeUndefined();
    expect((result.content[0] as { text: string }).text).toContain("inactive");
  });
});

describe("employee tools", () => {
  it("getEmployeeProfile rejects empty name", async () => {
    const client = mockClient({});
    const result = await getEmployeeProfile(client, "");
    expect(result.isError).toBe(true);
  });
});
