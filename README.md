# tlog

Task tracking for AI coding agents. Helps humans and agents collaborate on work.

> **Warning:** This is an experiment. tlog is a drastically simplified step cousin to [steveyegge/beads](https://github.com/steveyegge/beads), testing the thesis that beads does too much for one tool. tlog intentionally does less to enable more flexibility in workflows. If you want battle-tested, use beads.

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
tlog init                    # initialize in current directory
tlog create "task title"     # create a task
tlog ready                   # list tasks ready to work on
tlog claim <id>              # claim a task (mark in_progress)
tlog done <id>               # mark task complete
tlog unclaim <id>            # release task back to open

tlog list                    # list open tasks
tlog list --status all       # list all tasks
tlog list --priority high    # filter by priority
tlog backlog                 # list backlog tasks
tlog graph                   # show dependency tree
tlog show <id>               # show task details

tlog dep <id> <dep-id>       # add dependency
tlog update <id> --title "x" # update task
tlog sync                    # commit .tlog to git
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
