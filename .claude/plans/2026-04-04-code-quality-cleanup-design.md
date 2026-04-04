# Code Quality Cleanup — Design Spec

> **Date:** 2026-04-04 | **Status:** Approved

## Goal

Reduce the two largest files in the codebase to under 800 lines each, commit pending frontend bugfixes, and clean up billing TODOs. Pure refactoring — zero behavior changes.

## Constraints

| Dimension | Decision |
|---|---|
| Behavior changes | Zero — all tests must pass before and after |
| New packages | Zero — files stay in same package (`cmd/brain`, `internal/bot`) |
| Database | Zero migrations |
| Frontend features | Zero — only commit existing uncommitted fixes |
| API | Zero endpoint changes |

## 1. Split `cmd/brain/main.go` (2741 → ~500 lines)

### Current Structure

`main.go` contains 42 functions covering: type definitions, adapters, migrations (inline SQL), dependency bootstrap, message handlers, event subscribers, scheduler job definitions, AI role manager setup, API router, health check, graceful shutdown, and world model trigger evaluation.

### Extraction Plan

All new files remain in `cmd/brain/` package `main`. No new types or interfaces — just moving existing code.

| New File | Content | Approx Lines |
|---|---|---|
| `migrations.go` | `runMigrations()` + all migration SQL constants | ~800 |
| `bootstrap.go` | `setupServices()` — extracted from main() lines that create DB, Redis, LLM, memory engine, seat service, onboarding, brain engines, consulting, report components, world model, command handler, channel router, dispatcher, recommender | ~400 |
| `handlers.go` | Telegram raw text handler closure + unified handler closure (the two big anonymous functions in main) — extracted as named functions taking dependencies | ~350 |
| `scheduler_jobs.go` | All cron job registration callbacks (proactive actions, memory consolidation, group mentor, goal snapshots, brain signals, incentive calculation, recommendation scan, engagement tracker) | ~400 |
| `adapters.go` | `redactHandler`, `schedulerCallbacks`, `redisWrapper`, `groupDBAdapter`, `seatServiceAdapter`, `onboardingAdapter`, `consultingBotAdapter` — all adapter types and methods | ~250 |
| `utils.go` | `engineForTenant()`, `fetchBossContext()`, `parseUUIDForChat()`, `numericFromFloat()`, `formatPgUUID()`, `numericToFloat64()` | ~100 |
| `world_model_triggers.go` | `evaluateWorldModelTriggers()` | ~100 |

### `main.go` After Refactor (~500 lines)

Keeps only:
- `main()` function skeleton — calls `runMigrations()`, `setupServices()`, registers handlers/subscribers/scheduler, starts servers, waits for shutdown
- Graceful shutdown logic
- Health check endpoint
- Imports

### Dependency Passing

The current `main()` uses local variables extensively. To avoid a massive "context bag" struct:
- `bootstrap.go` returns a `services` struct containing all initialized services
- `handlers.go` functions accept the services they need as parameters
- `scheduler_jobs.go` functions accept `schedulerCallbacks` + specific services
- No globals — all dependencies explicit

### `services` Struct

```go
type services struct {
    cfg            *config.Config
    logger         *slog.Logger
    db             *pgxpool.Pool
    queries        *sqlc.Queries
    redis          *redis.Client
    llm            *brain.LLMService
    memoryEngine   *memory.Engine
    seatService    *seats.Service
    onboarding     *onboarding.Service
    orgEngine      *brain.OrgEngine
    orgWizard      *brain.OrgWizard
    consultEngine  *brain.ConsultingEngine
    collector      *report.Collector
    summarizer     *report.Summarizer
    chaser         *report.Chaser
    alertChecker   *report.AlertChecker
    actionExec     *report.ActionExecutor
    recommender    *brain.Recommender
    dispatcher     *brain.Dispatcher
    wmService      *worldmodel.Service
    wmExtractor    *worldmodel.Extractor
    cmdHandler     *bot.CommandHandler
    channelRouter  *channel.Router
    eventBus       *events.Bus
    mentorEngine   func(string) *brain.MentorEngine
}
```

## 2. Split `internal/bot/commands.go` (955 → ~150 lines)

### Extraction Plan

All files stay in `internal/bot/` package `bot`. `CommandHandler` struct and its interfaces stay in `commands.go`.

| New File | Functions | Approx Lines |
|---|---|---|
| `commands_onboarding.go` | `HandleStart`, `HandleJoin`, `generateInviteCode` | ~120 |
| `commands_team.go` | `HandleStatus`, `HandleAddEmployee`, `HandleProfile` | ~170 |
| `commands_config.go` | `HandleMentor`, `HandleBlend`, `HandleCulture` | ~160 |
| `commands_seats.go` | `HandleTalk`, `HandleBoard`, `HandleTeam`, `HandleAssign`, `listPersonas`, `defaultTitleForSeatType` | ~170 |
| `commands_consulting.go` | `HandleConsult`, `resolveTenantID` | ~110 |

### `commands.go` After Refactor (~150 lines)

Keeps:
- `CommandHandler` struct definition
- Interface definitions (`CommandQuerier`, `GroupQuerier`, `SeatServicer`, etc.)
- Constructor `NewCommandHandler()`
- Setter methods (`SetGroupDB`, `SetSeatService`, etc.)
- `HandleHelp`, `HandleDiagnostics`
- `mentorDescriptions` map

## 3. Commit Frontend Bugfixes

Two existing uncommitted changes:

1. **`frontend/src/App.vue`**: Added `router.isReady()` guard with `routerReady` ref — prevents layout flash on initial load
2. **`frontend/src/api/client.ts`**: Added `pathname !== '/login'` check in 401 handler — prevents infinite redirect loop

Both are small, correct fixes. Commit and deploy.

## 4. Clean Up Billing TODOs

4 TODOs in `internal/api/billing.go` (webhook handler) and 1 in `internal/report/chaser.go`:

- Change `// TODO:` to `// FUTURE:` with brief explanation
- Makes intent clear: these are planned features, not forgotten bugs

## 5. Verification

After all changes:
1. `go build ./...` — must compile
2. `go vet ./...` — must pass
3. `go test ./... -count=1` — all existing tests must pass
4. `wc -l` on refactored files — all under 800 lines
5. Frontend build — must succeed
6. Deploy to server — health check must pass

## File Change Summary

### Modified (2 files)
- `cmd/brain/main.go` — reduced from 2741 to ~500 lines
- `internal/bot/commands.go` — reduced from 955 to ~150 lines

### Created (12 files)
- `cmd/brain/migrations.go`
- `cmd/brain/bootstrap.go`
- `cmd/brain/handlers.go`
- `cmd/brain/scheduler_jobs.go`
- `cmd/brain/adapters.go`
- `cmd/brain/utils.go`
- `cmd/brain/world_model_triggers.go`
- `internal/bot/commands_onboarding.go`
- `internal/bot/commands_team.go`
- `internal/bot/commands_config.go`
- `internal/bot/commands_seats.go`
- `internal/bot/commands_consulting.go`

### Committed (2 files, already modified)
- `frontend/src/App.vue`
- `frontend/src/api/client.ts`

### Minor Edit (2 files)
- `internal/api/billing.go` — TODO → FUTURE
- `internal/report/chaser.go` — TODO → FUTURE
