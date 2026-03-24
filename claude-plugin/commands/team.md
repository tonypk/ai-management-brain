---
description: Add, list, or remove team members
argument-hint: <add|list|remove> [options]
allowed-tools: [Read, Write]
---

# Team Management

Manage team members for Boss AI Agent.

**Subcommands:**

- `/team list` — show all employees
- `/team add <name> --channel <slack|telegram|email> --id <channel-id> [--culture <code>] [--role <role>]` — add an employee
- `/team remove <name>` — deactivate an employee

**Process:**

1. Read `~/.boss-ai-agent/data/employees.json`.

2. For **list**: display a table of all employees with name, channel, culture, role, active status.

3. For **add**:
   - Parse the employee details from the arguments.
   - Validate channel is one of: slack, telegram, email.
   - If culture not specified, use the global culture from config.
   - If role not specified, default to "member".
   - Append to employees.json.
   - Confirm: "Added {name} ({channel}, culture: {culture})."

4. For **remove**:
   - Find employee by name (case-insensitive).
   - Set `active: false` (soft delete).
   - Confirm: "Removed {name} from active team."

**Employee record format:**
```json
{
  "name": "John Santos",
  "channel": "telegram",
  "channelId": "123456789",
  "culture": "philippines",
  "role": "member",
  "active": true,
  "addedAt": "2026-03-24"
}
```

**Available cultures:** default, philippines, singapore, indonesia, srilanka, malaysia, china, usa, india
