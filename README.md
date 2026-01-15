# tlog

Task tracking for AI coding agents. Helps humans and agents collaborate on work.

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

Heavily inspired by [steveyegge/beads](https://github.com/steveyegge/beads), which is fantastic. tlog aims to be a lighter-weight alternative.

## License

MIT
