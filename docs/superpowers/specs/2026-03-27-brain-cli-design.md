# Brain CLI — Design Spec

## Goal

Standalone `brain` CLI tool that calls manageaibrain.com API directly, giving bosses terminal-native access to daily management operations without Claude Code or a browser.

## Architecture

```
brain CLI (Go binary)
  │
  ├── Config (~/.brain/config.yaml or env vars)
  │     ├── api_key: MANAGEMENT_BRAIN_API_KEY
  │     └── server: https://manageaibrain.com (default)
  │
  └── HTTP Client → /api/v1/openclaw/* endpoints
        ├── GET endpoints (status, report, alerts, risks, kpis, employees, profile)
        └── POST endpoints (checkin, chase, summary)
```

**Key principle**: The CLI is a thin HTTP client. All business logic lives on the server. The CLI's job is authentication, request formatting, and terminal output.

## Authentication

Priority order:
1. `MANAGEMENT_BRAIN_API_KEY` environment variable
2. `~/.brain/config.yaml` → `api_key` field
3. If neither found → prompt "Run `brain login` to configure"

The API key is sent as `Authorization: Bearer <key>` header (matching existing API key auth in the backend).

## Config File

Path: `~/.brain/config.yaml`

```yaml
server: https://manageaibrain.com
api_key: sk-brain-xxxxxxxxxxxx
```

Created by `brain login`. User can also edit manually.

## Commands (12)

### Setup

| Command | Action |
|---------|--------|
| `brain login` | Interactive: prompt for API key + server URL → write `~/.brain/config.yaml` |
| `brain version` | Print version and server URL |

### Read Operations (7)

| Command | HTTP | Endpoint | Output |
|---------|------|----------|--------|
| `brain status` | GET | /api/v1/openclaw/status | Check-in progress table: submitted/pending/chased |
| `brain report [weekly\|monthly]` | GET | /api/v1/openclaw/report?period=X | Performance rankings table + 1:1 suggestions |
| `brain alerts` | GET | /api/v1/openclaw/alerts | Consecutive missed days, sorted by severity |
| `brain risks` | GET | /api/v1/openclaw/state/risks | Top execution risks with scores |
| `brain kpis` | GET | /api/v1/openclaw/kpis | KPI metrics vs targets table |
| `brain employees` | GET | /api/v1/openclaw/status | Employee list with roles |
| `brain profile <name>` | GET | /api/v1/employees/profile/:name | Detailed employee profile + sentiment trend |

### Write Operations (3)

| Command | HTTP | Endpoint | Effect |
|---------|------|----------|--------|
| `brain checkin [name]` | POST | /api/v1/openclaw/checkin | Send check-in to one or all employees |
| `brain chase [name]` | POST | /api/v1/openclaw/chase | Chase non-submitters |
| `brain summary` | POST | /api/v1/openclaw/summary | Generate + send daily summary to boss |

Write commands print a confirmation before executing (e.g., "Send check-in to 5 employees? [y/N]"). Pass `--yes` or `-y` to skip confirmation.

## File Structure

```
cmd/brain-cli/
├── main.go           # cobra root command, version, subcommand registration
├── config.go         # loadConfig(): env → file → error; saveConfig()
├── client.go         # BrainClient: base URL, API key, doGet/doPost helpers
├── format.go         # table(), bold(), color(), printJSON() terminal helpers
├── cmd_login.go      # brain login
├── cmd_status.go     # brain status
├── cmd_report.go     # brain report
├── cmd_alerts.go     # brain alerts
├── cmd_checkin.go    # brain checkin
├── cmd_chase.go      # brain chase
├── cmd_summary.go    # brain summary
├── cmd_risks.go      # brain risks
├── cmd_kpis.go       # brain kpis
├── cmd_employees.go  # brain employees
├── cmd_profile.go    # brain profile
└── cmd_version.go    # brain version
```

Each `cmd_*.go` file: ~40-60 lines (cobra command + HTTP call + format output).

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — config file parsing
- Standard library only for HTTP, JSON, terminal output

No lipgloss/glamour — keep it minimal. Use ANSI escape codes directly in `format.go` for bold/color.

## API Response Handling

All `/api/v1/openclaw/*` endpoints return the standard envelope:

```json
{
  "success": true,
  "data": { ... },
  "error": null
}
```

The CLI checks `success` field. On `false`, print `error` message and exit with code 1. On HTTP 401, print "Invalid API key. Run `brain login` to reconfigure."

## Terminal Output Examples

### `brain status`

```
Team Status — 2026-03-27

Submitted  3/5 (60%)
Pending    Alice, Bob
Chased     0

Recent check-ins:
  Charlie   positive   Finished login page, no blockers
  Dave      neutral    Working on API docs
  Eve       anxious    Blocked by CI issue (day 2)
```

### `brain kpis`

```
KPI Dashboard

Name              Value   Target  Status
Revenue MRR       $45K    $50K    ⚠ 90%
Sprint Velocity   42      40      ✓ 105%
Bug Count         12      <10     ✗ 120%
NPS Score         72      80      ⚠ 90%
```

### `brain risks`

```
Top Risks

Score  Type        Employee   Evidence
0.85   overload    Alice      3 tasks overdue, sentiment: frustrated
0.72   delivery    Backend    Sprint behind 40%, 2 blocked PRs
0.65   engagement  Bob        3 consecutive missed check-ins
```

## Build & Install

```bash
# Build
CGO_ENABLED=0 go build -o brain ./cmd/brain-cli

# Install globally
go install ./cmd/brain-cli
# Binary name: brain-cli (rename to brain)

# Cross-compile for Linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o brain-linux ./cmd/brain-cli
```

Future: publish to Homebrew tap or GitHub Releases.

## Out of Scope

- Interactive TUI (ncurses-style) — this is a simple CLI, not a TUI app
- Offline mode — CLI always needs server connection
- Auto-update — users manually download new versions
- Shell completion — can add later via cobra's built-in completion
- Piping/JSON mode (`--json` flag) — can add later
