# Agent Instructions

## Task Tracking with tlog

This project uses `tlog` for task management. Use it to track work across sessions.

### Starting a Session

```bash
tlog prime   # Get current context - what's done, what's ready
tlog ready   # See tasks available to work on
```

### During Work

```bash
tlog create "Task description"           # Create new task
tlog create "Subtask" --dep tl-PARENT    # Create dependent task
tlog show tl-ID                          # Get task details
```

### Completing Work

```bash
tlog done tl-ID    # Mark task complete
tlog sync          # Commit changes to git
```

### Key Commands

| Command | Purpose |
|---------|---------|
| `tlog prime` | Context summary for session start |
| `tlog ready` | List tasks ready to work on |
| `tlog create "title"` | Create new task |
| `tlog done ID` | Mark task complete |
| `tlog list` | List all open tasks |
| `tlog graph` | Show dependency graph |
| `tlog sync` | Git commit .tlog/ |

### Workflow

1. Run `tlog prime` at session start
2. Pick a task from `tlog ready`
3. Create subtasks as needed with `--dep`
4. Mark tasks done as you complete them
5. Run `tlog sync` before ending session

### Output Format

All commands output JSON for easy parsing. Example:

```json
{
  "id": "tl-a1b2c3d4",
  "title": "Implement feature X",
  "status": "open",
  "deps": ["tl-e5f6g7h8"]
}
```
