#!/usr/bin/env node
import { createServer } from "./server.js";

const isHttpMode =
  process.argv.includes("--http") || process.env.TRANSPORT === "http";

async function main() {
  if (isHttpMode) {
    const { startHttpServer } = await import("./http-transport.js");
    await startHttpServer();
  } else {
    const { StdioServerTransport } = await import(
      "@modelcontextprotocol/sdk/server/stdio.js"
    );
    const server = createServer();
    const transport = new StdioServerTransport();
    await server.connect(transport);
  }
}

main().catch((error) => {
  console.error("MCP server failed to start:", error);
  process.exit(1);
});
