# tlog

Task tracking for AI coding agents. Helps humans and agents collaborate on work.

> **Warning:** This is an experiment. tlog is a drastically simplified step cousin to [steveyegge/beads](https://github.com/steveyegge/beads), testing the thesis that beads does too much. tlog intentionally does less to enable more flexibility in workflows. If you want battle-tested, use beads.

## What it does

- Agents track their work with simple CLI commands
- Tasks have dependencies (do X before Y)
- State persists across sessions and git branches
- Humans can see what agents are doing and what's next

## Install

```bash
go install github.com/richhaase/tlog/cmd/tlog@latest
```

## Usage

```bash
tlog help
```

## For agents

Add to your `CLAUDE.md` or `AGENTS.md`:

```markdown
This project uses tlog for task tracking. Run `tlog prime` to get started.
```

## Inspiration

[steveyegge/beads](https://github.com/steveyegge/beads)

## License

MIT
