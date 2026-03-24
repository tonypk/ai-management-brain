# @tonykk/management-brain-mcp

MCP server for [AI Management Brain](https://manageaibrain.com) — 9 tools for team management, C-Suite board discussions, and employee insights.

Supports both **stdio** (Claude Code, OpenClaw) and **HTTP** (ChatGPT, Gemini, remote clients) transports.

## Install (stdio mode)

Add to your Claude Code MCP config (`~/.claude.json` or project `.mcp.json`):

```json
{
  "mcpServers": {
    "management-brain": {
      "command": "npx",
      "args": ["-y", "@tonykk/management-brain-mcp"],
      "env": {
        "MANAGEMENT_BRAIN_API_KEY": "your-api-key"
      }
    }
  }
}
```

Zero local dependencies. One environment variable.

## HTTP Mode (for ChatGPT / remote clients)

Start the server in HTTP mode:

```bash
TRANSPORT=http \
MCP_HTTP_API_KEY=your-secret \
MANAGEMENT_BRAIN_API_KEY=your-api-key \
node dist/index.js
```

Or use the npm script:

```bash
npm run start:http
```

The server listens on port 3100 (configurable via `MCP_PORT`).

### Docker

```bash
docker build -t management-brain-mcp ./mcp-server
docker run -p 3100:3100 \
  -e MCP_HTTP_API_KEY=your-secret \
  -e MANAGEMENT_BRAIN_API_KEY=your-api-key \
  management-brain-mcp
```

### Testing the HTTP endpoint

```bash
# Health check
curl http://localhost:3100/health

# MCP initialize
curl -X POST http://localhost:3100/mcp \
  -H "Authorization: Bearer your-secret" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
```

### Production deployment

The MCP HTTP server is included in `docker-compose.prod.yml`. Set `MCP_HTTP_API_KEY` in your `.env` file and rebuild:

```bash
docker compose -f docker-compose.prod.yml up -d --build mcp frontend
```

The endpoint is available at `https://manageaibrain.com/mcp`.

## Tools

| Tool | Description |
|------|-------------|
| `get_team_status` | Today's check-in status — submission rate, pending employees |
| `get_report` | Team performance report (weekly/monthly) with ranking |
| `get_alerts` | Alerts for employees with consecutive missed days |
| `switch_mentor` | Switch management mentor (musk, inamori, dalio, etc.) |
| `list_mentors` | List all mentors with expertise and recommended seats |
| `board_discuss` | Board discussion across all C-Suite seats on a topic |
| `chat_with_seat` | Chat with a specific C-Suite seat (CEO, CFO, etc.) |
| `list_employees` | List all active employees |
| `get_employee_profile` | Employee profile with sentiment and submission history |

## Usage Examples

In Claude Code:
- "How's my team doing today?" → `get_team_status`
- "Show me the weekly report" → `get_report`
- "Should we expand to Japan?" → `board_discuss`
- "Ask the CFO about Q2 budget" → `chat_with_seat`
- "How is John doing?" → `get_employee_profile`
- "Switch to Inamori management style" → `switch_mentor`

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MANAGEMENT_BRAIN_API_KEY` | Yes | — | API key from manageaibrain.com |
| `MANAGEMENT_BRAIN_BASE_URL` | No | `https://manageaibrain.com` | Override API URL |
| `TRANSPORT` | No | `stdio` | Transport mode: `stdio` or `http` |
| `MCP_HTTP_API_KEY` | HTTP mode | — | Bearer token for HTTP authentication |
| `MCP_PORT` | No | `3100` | HTTP server port |
| `MCP_CORS_ORIGINS` | No | `*` | Comma-separated allowed CORS origins |

## Architecture

```
Claude Code / OpenClaw          ChatGPT / Gemini
    │                               │
    │ stdio (local)                  │ HTTPS
    │                               │
  npx management-brain-mcp    manageaibrain.com/mcp
                                    │
                               nginx → mcp:3100
                                    │
                              MCP HTTP Container
                                    │
                          http://brain:8080 (internal)
```

## License

MIT
