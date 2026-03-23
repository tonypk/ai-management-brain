# Boss Memory Recall — Design Spec

**Date:** 2026-03-23
**Status:** Draft

## Problem

When the boss chats with the AI mentor, mentioning an employee by name should trigger automatic recall of that employee's memories (insights, strategies, knowledge, profile). Currently the code has a TODO placeholder that matches employee names but skips the actual recall because `RosterEntry` lacks the employee ID.

## Solution

Add the employee ID to `RosterEntry`, then implement the memory recall loop in `HandleBoss` to call the existing `RecallForMentor` API for each matched employee.

## Design

### Data Changes

**`RosterEntry` struct** (`internal/brain/chat.go`):
- Add `ID string` field

**`fetchBossContext`** (`cmd/brain/main.go`):
- Include the employee's UUID when building the roster entry

### Logic Changes

**`HandleBoss`** (`internal/brain/chat.go`):
- Match ALL mentioned employees (not just the first — remove `break`)
- Cap at 3 matched employees to control token budget
- For each matched employee, call `engine.MemoryEngine().RecallForMentor(ctx, tenantID, emp.ID, text)`
- Format each result with `memory.FormatForPrompt()` and concatenate
- Inject the combined memory section into `BuildBossContext.MemorySection`

### Token Budget

- Each `RecallForMentor` call is capped at 800 tokens by the Retriever
- With max 3 employees, worst case ~2400 tokens of memory injection
- This is within acceptable limits for a boss system prompt

### Error Handling

- If recall fails for one employee, log warning and skip (don't fail the entire message)
- If no memories found for a matched employee, the formatted result is empty — no injection

## Files Changed

- `internal/brain/chat.go` — Add ID to RosterEntry, implement memory recall loop
- `cmd/brain/main.go` — Pass employee ID in fetchBossContext

## Testing

- Update `matchEmployeeName` tests if needed (already has tests)
- Add test for boss memory recall with mock memory engine
- Verify multi-employee matching works correctly

## Out of Scope

- No new API endpoints
- No frontend changes
- No database changes
- No new dependencies
