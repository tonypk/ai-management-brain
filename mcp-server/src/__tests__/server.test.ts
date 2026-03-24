import { describe, it, expect, vi, beforeEach } from "vitest";
import { createServer } from "../server.js";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";

beforeEach(() => {
  vi.unstubAllEnvs();
});

describe("createServer", () => {
  it("returns an McpServer instance", () => {
    const server = createServer();
    expect(server).toBeDefined();
    expect(typeof server.connect).toBe("function");
    expect(typeof server.close).toBe("function");
  });

  it("can be called multiple times independently", () => {
    const s1 = createServer();
    const s2 = createServer();
    expect(s1).not.toBe(s2);
  });
});
