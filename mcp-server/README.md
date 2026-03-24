# @tonypk/management-brain-mcp

MCP server for [AI Management Brain](https://manageaibrain.com) — 9 tools for team management, C-Suite board discussions, and employee insights.

## Install

Add to your Claude Code MCP config (`~/.claude.json` or project `.mcp.json`):

```json
{
  "mcpServers": {
    "management-brain": {
      "command": "npx",
      "args": ["-y", "@tonypk/management-brain-mcp"],
      "env": {
        "MANAGEMENT_BRAIN_API_KEY": "your-api-key"
      }
    }
  }
}
```

Zero local dependencies. One environment variable.

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

| Variable | Required | Description |
|----------|----------|-------------|
| `MANAGEMENT_BRAIN_API_KEY` | Yes | API key from manageaibrain.com |
| `MANAGEMENT_BRAIN_BASE_URL` | No | Override API URL (default: `https://manageaibrain.com`) |

## License

MIT
