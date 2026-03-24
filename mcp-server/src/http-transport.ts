import express, {
  type Request,
  type Response,
  type NextFunction,
} from "express";
import cors from "cors";
import rateLimit from "express-rate-limit";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { createServer } from "./server.js";

const PORT = parseInt(process.env.MCP_PORT ?? "3100", 10);
const API_KEY = process.env.MCP_HTTP_API_KEY ?? "";
const CORS_ORIGINS = process.env.MCP_CORS_ORIGINS
  ? process.env.MCP_CORS_ORIGINS.split(",").map((s) => s.trim())
  : "*";

function authMiddleware(req: Request, res: Response, next: NextFunction): void {
  // Skip auth for health check
  if (req.path === "/health") {
    next();
    return;
  }

  if (!API_KEY) {
    res.status(500).json({ error: "MCP_HTTP_API_KEY not configured" });
    return;
  }

  const authHeader = req.headers.authorization;
  if (!authHeader?.startsWith("Bearer ")) {
    res.status(401).json({ error: "Missing or invalid Authorization header" });
    return;
  }

  const token = authHeader.slice(7);
  if (token !== API_KEY) {
    res.status(401).json({ error: "Invalid API key" });
    return;
  }

  next();
}

export async function startHttpServer(): Promise<void> {
  const app = express();

  // Rate limiting
  app.use(
    rateLimit({
      windowMs: 60_000,
      limit: 120,
      standardHeaders: "draft-7",
      legacyHeaders: false,
      message: { error: "Too many requests, please try again later." },
    }),
  );

  // CORS
  app.use(cors({ origin: CORS_ORIGINS }));

  // Auth
  app.use(authMiddleware);

  // Health check
  app.get("/health", (_req: Request, res: Response) => {
    res.json({ status: "ok", transport: "http", timestamp: new Date().toISOString() });
  });

  // MCP endpoint — stateless mode (one transport per request)
  app.all("/mcp", async (req: Request, res: Response) => {
    const transport = new StreamableHTTPServerTransport({
      sessionIdGenerator: undefined, // stateless
    });

    const server = createServer();
    await server.connect(transport);

    await transport.handleRequest(req, res);

    await transport.close();
    await server.close();
  });

  const httpServer = app.listen(PORT, "0.0.0.0", () => {
    console.log(`MCP HTTP server listening on port ${PORT}`);
  });

  // Graceful shutdown
  const shutdown = () => {
    console.log("Shutting down MCP HTTP server...");
    httpServer.close(() => {
      console.log("MCP HTTP server stopped");
      process.exit(0);
    });
  };

  process.on("SIGTERM", shutdown);
  process.on("SIGINT", shutdown);
}
