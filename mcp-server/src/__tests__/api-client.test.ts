import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ApiClient, APIError } from "../api-client.js";

describe("ApiClient", () => {
  let client: ApiClient;
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    client = new ApiClient("https://example.com", "test-key");
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("sends authorization header", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 200,
      json: () => Promise.resolve({ result: "ok" }),
    });

    await client.get("/api/v1/test");

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "https://example.com/api/v1/test",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer test-key",
        }),
      }),
    );
  });

  it("extracts .data from wrapped responses", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 200,
      json: () => Promise.resolve({ data: { name: "test" } }),
    });

    const result = await client.get("/api/v1/wrapped");
    expect(result).toEqual({ name: "test" });
  });

  it("passes through flat responses", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 200,
      json: () => Promise.resolve({ date: "2026-03-24", submitted: 4 }),
    });

    const result = await client.get<{ date: string; submitted: number }>(
      "/api/v1/flat",
    );
    expect(result).toEqual({ date: "2026-03-24", submitted: 4 });
  });

  it("throws APIError on 401", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 401,
      json: () => Promise.resolve({ error: "unauthorized" }),
    });

    await expect(client.get("/api/v1/test")).rejects.toThrow(APIError);
    await expect(client.get("/api/v1/test")).rejects.toThrow(
      "Invalid API key",
    );
  });

  it("throws APIError on 429", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 429,
      json: () => Promise.resolve({ error: "rate limited" }),
    });

    await expect(client.get("/api/v1/test")).rejects.toThrow(
      "rate limited",
    );
  });

  it("retries once on 5xx", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ status: 500, json: () => Promise.resolve({}) })
      .mockResolvedValueOnce({
        status: 200,
        json: () => Promise.resolve({ result: "ok" }),
      });
    globalThis.fetch = fetchMock;

    const result = await client.get("/api/v1/test");
    expect(result).toEqual({ result: "ok" });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("throws after two 5xx failures", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 500,
      json: () => Promise.resolve({}),
    });

    await expect(client.get("/api/v1/test")).rejects.toThrow(
      "Server error. Please try again.",
    );
  });

  it("sends POST body as JSON", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      status: 200,
      json: () => Promise.resolve({ result: "ok" }),
    });

    await client.post("/api/v1/test", { command: "switch mentor musk" });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "https://example.com/api/v1/test",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ command: "switch mentor musk" }),
      }),
    );
  });
});
