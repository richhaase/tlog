# tlog - Append-Only Task Log for AI Coding Agents

An AI-first task tracker with dependency management. Inspired by [Steve Yegge's beads](https://github.com/steveyegge/beads).

## Design Philosophy

- **AI-first**: JSON input/output, designed for coding agents to read and write
- **Append-only**: All mutations are appends. No updates, no deletes. Current state is computed by replaying events.
- **Git-native**: `.tlog/` directory commits and merges like code. Hash-based IDs prevent conflicts.
- **Zero infrastructure**: No database, no daemon, no server. Just files.

## Installation

```bash
# Install
go install github.com/rdh/tlog/cmd/tlog@latest

# Or build from source
go build -o tlog ./cmd/tlog
```

## Quick Start

```bash
# Initialize in your project
tlog init

# Create tasks
tlog create "Set up authentication"
tlog create "Build login page" --dep tl-abc123

# See what's ready to work on
tlog ready

# Mark done
tlog done tl-abc123

# Commit to git
tlog sync
```

## Commands

### Task Management

```bash
tlog create "title"              # Create task
tlog create "title" -d ID        # Create with dependency
tlog create "title" -b ID        # Create, blocking another task
tlog done ID                     # Mark complete
tlog reopen ID                   # Reopen completed task
tlog update ID --title "new"     # Update title/notes
```

### Querying

```bash
tlog list                        # List open tasks
tlog list --status all           # List all tasks
tlog list --status done          # List completed
tlog show ID                     # Show task details
tlog ready                       # Tasks ready to work on
```

### Dependencies

```bash
tlog dep ID --on OTHER           # ID depends on OTHER
tlog dep ID --on OTHER --remove  # Remove dependency
tlog block ID --blocks OTHER     # ID blocks OTHER
tlog graph                       # Dependency graph (JSON)
tlog graph --format mermaid      # Mermaid diagram
```

### AI Agent Integration

```bash
tlog prime                       # Context blob for agent injection (~1-2k tokens)
```

### Git Integration

```bash
tlog sync                        # git add + commit .tlog/
tlog sync -m "message"           # Custom commit message
```

## Storage Format

All data lives in `.tlog/events/` as append-only JSONL files:

```
.tlog/
├── config.json
└── events/
    ├── 2026-01-14.jsonl
    └── 2026-01-15.jsonl
```

Each line is an event:

```jsonl
{"id":"tl-a1b2c3d4","ts":"2026-01-14T10:00:00Z","type":"create","title":"Set up auth","status":"open","deps":[]}
{"id":"tl-a1b2c3d4","ts":"2026-01-14T11:00:00Z","type":"status","status":"done"}
```

Current state is computed by replaying events. Latest event wins per task ID.

## Why Append-Only?

1. **Conflict-free merges**: Two branches can create tasks independently. Hash IDs never collide.
2. **Full history**: Every change is recorded. Debug by reading the event log.
3. **Simple implementation**: No update/delete logic. Just append and replay.
4. **Git-friendly**: Append-only files merge trivially.

## Agent Integration

Add to your `AGENTS.md` or system prompt:

```markdown
## Task Tracking

Use `tlog` for task management:
- `tlog prime` - Get current context
- `tlog ready` - See available work  
- `tlog create "title"` - Create task
- `tlog done ID` - Complete task
- `tlog sync` - Commit changes
```

## License

MIT
