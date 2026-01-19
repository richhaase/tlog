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
# Setup
tlog init                    # initialize in current directory
tlog prime                   # get AI agent context (start here)

# Task lifecycle
tlog create "task title"     # create a task
tlog claim <id>              # claim a task (mark in_progress)
tlog done <id>               # mark task complete
tlog done <id> --commit abc  # mark done and record commit SHA
tlog unclaim <id>            # release task back to open
tlog reopen <id>             # reopen a done/in_progress task
tlog delete <id>             # soft-delete task (removed on prune)

# Querying
tlog ready                   # list tasks ready to work on
tlog list                    # list open tasks
tlog list --status all       # list all tasks
tlog list --priority high    # filter by priority
tlog backlog                 # list backlog tasks
tlog show <id>               # show task details
tlog graph                   # show dependency tree

# Task metadata
tlog create "x" --for <parent>         # create subtask
tlog create "x" --priority high        # set priority
tlog update <id> --note "what happened"  # append note
tlog dep <id> --needs <dep-id>         # add dependency
tlog dep <id> --remove <dep-id>        # remove dependency

# Maintenance
tlog sync "message"          # commit .tlog to git
tlog prune                   # compact files and remove done tasks
tlog labels                  # show labels in use
```

## For agents

Add to your `CLAUDE.md` or `AGENTS.md`:

```markdown
This project uses tlog for task tracking. Run `tlog prime` to get started.
```

## Development

```bash
make help           # show available targets
make build          # build to bin/tlog
make install        # install to GOBIN
make test           # run tests
make check          # run all quality checks
```

## Inspiration

[steveyegge/beads](https://github.com/steveyegge/beads)

## License

MIT
