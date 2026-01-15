package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/richhaase/tlog/internal/tlog"
)

func outputJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func errorJSON(msg string) {
	outputJSON(map[string]string{"error": msg})
	os.Exit(1)
}

// resolveID resolves a prefix to a full task ID
func resolveID(root, prefix string) string {
	events, err := tlog.LoadAllEvents(root)
	if err != nil {
		errorJSON(err.Error())
	}
	tasks := tlog.ComputeState(events)
	id, err := tlog.ResolveID(tasks, prefix)
	if err != nil {
		errorJSON(err.Error())
	}
	return id
}

func usage() {
	fmt.Println(`tlog - append-only task tracking for AI agents

Usage: tlog <command> [options]

Commands:
  init                    Initialize tlog in current directory
  create <title>          Create a new task
    --dep <id>            Add dependency (repeatable)
    --blocks <id>         Add blocking relationship (repeatable)
    --label <label>       Add label (repeatable)
    --description <text>  Set description (what this task is)
    --notes <text>        Add notes (what happened)
  done <id>               Mark task as done (resolution: completed)
    --wontfix             Resolution: wontfix
    --duplicate           Resolution: duplicate
    --note <text>         Append closing note
  claim <id>              Mark task as in_progress
    --note <text>         Append note
  unclaim <id>            Release claimed task back to open
    --note <text>         Append note
  reopen <id>             Reopen task (from done or in_progress)
  update <id>             Update task
    --title <text>        New title
    --description <text>  Set description (overwrites)
    --notes <text>        Append notes
    --label <label>       Set labels (repeatable)
  list                    List tasks
    --status <s>          Filter by status (open|in_progress|done|all, default: open)
    --label <label>       Filter by label
  show <id>               Show task details
  ready                   List tasks ready to work on
  dep <id> <dep-id>       Add dependency
    --remove              Remove instead of add
  block <id> <block-id>   Add blocking relationship
    --remove              Remove instead of add
  graph [--format <f>]    Show dependency graph (json|mermaid)
  prime                   Get AI agent context
  labels                  Show labels in use and conventions
  sync [-m|--message <m>] Commit .tlog to git
  version                 Show version information`)
	os.Exit(0)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "help", "-h", "--help":
		usage()

	case "version", "-v", "--version":
		fmt.Println(buildVersionString())

	case "init":
		cwd, _ := os.Getwd()
		result, err := tlog.CmdInit(cwd)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "create":
		if len(args) < 1 {
			errorJSON("create requires a title")
		}
		title := args[0]
		var deps, blocks, labels []string
		var description, notes string

		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--dep":
				if i+1 < len(args) {
					deps = append(deps, args[i+1])
					i++
				}
			case "--blocks":
				if i+1 < len(args) {
					blocks = append(blocks, args[i+1])
					i++
				}
			case "--label":
				if i+1 < len(args) {
					labels = append(labels, args[i+1])
					i++
				}
			case "--description":
				if i+1 < len(args) {
					description = args[i+1]
					i++
				}
			case "--notes":
				if i+1 < len(args) {
					notes = args[i+1]
					i++
				}
			}
		}

		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		result, err := tlog.CmdCreate(root, title, deps, blocks, labels, description, notes)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "done":
		if len(args) < 1 {
			errorJSON("done requires a task ID")
		}
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		id := resolveID(root, args[0])

		var resolution tlog.Resolution
		var notes string
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--wontfix":
				resolution = tlog.ResolutionWontfix
			case "--duplicate":
				resolution = tlog.ResolutionDuplicate
			case "--note":
				if i+1 < len(args) {
					notes = args[i+1]
					i++
				}
			}
		}

		result, err := tlog.CmdDone(root, id, resolution, notes)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "claim":
		if len(args) < 1 {
			errorJSON("claim requires a task ID")
		}
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		id := resolveID(root, args[0])
		var notes string
		for i := 1; i < len(args); i++ {
			if args[i] == "--note" && i+1 < len(args) {
				notes = args[i+1]
				i++
			}
		}
		result, err := tlog.CmdClaim(root, id, notes)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "unclaim":
		if len(args) < 1 {
			errorJSON("unclaim requires a task ID")
		}
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		id := resolveID(root, args[0])
		var notes string
		for i := 1; i < len(args); i++ {
			if args[i] == "--note" && i+1 < len(args) {
				notes = args[i+1]
				i++
			}
		}
		result, err := tlog.CmdUnclaim(root, id, notes)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "reopen":
		if len(args) < 1 {
			errorJSON("reopen requires a task ID")
		}
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		id := resolveID(root, args[0])
		result, err := tlog.CmdReopen(root, id)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "update":
		if len(args) < 1 {
			errorJSON("update requires a task ID")
		}
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		id := resolveID(root, args[0])
		var title, description, notes string
		var labels []string

		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--title":
				if i+1 < len(args) {
					title = args[i+1]
					i++
				}
			case "--description":
				if i+1 < len(args) {
					description = args[i+1]
					i++
				}
			case "--notes":
				if i+1 < len(args) {
					notes = args[i+1]
					i++
				}
			case "--label":
				if i+1 < len(args) {
					labels = append(labels, args[i+1])
					i++
				}
			}
		}

		result, err := tlog.CmdUpdate(root, id, title, description, notes, labels)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "list":
		status := "open"
		label := ""
		for i := 0; i < len(args); i++ {
			if args[i] == "--status" && i+1 < len(args) {
				status = args[i+1]
				i++
			} else if args[i] == "--label" && i+1 < len(args) {
				label = args[i+1]
				i++
			}
		}

		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		result, err := tlog.CmdList(root, status, label)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "show":
		if len(args) < 1 {
			errorJSON("show requires a task ID")
		}
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		id := resolveID(root, args[0])
		result, err := tlog.CmdShow(root, id)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "ready":
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		result, err := tlog.CmdReady(root)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "dep":
		if len(args) < 2 {
			errorJSON("dep requires task ID and dependency ID")
		}
		action := "add"
		for _, a := range args {
			if a == "--remove" {
				action = "remove"
			}
		}
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		id := resolveID(root, args[0])
		depID := resolveID(root, args[1])
		result, err := tlog.CmdDep(root, id, depID, action)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "block":
		if len(args) < 2 {
			errorJSON("block requires task ID and block ID")
		}
		action := "add"
		for _, a := range args {
			if a == "--remove" {
				action = "remove"
			}
		}
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		id := resolveID(root, args[0])
		blockID := resolveID(root, args[1])
		result, err := tlog.CmdBlock(root, id, blockID, action)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "graph":
		format := "json"
		for i := 0; i < len(args); i++ {
			if args[i] == "--format" && i+1 < len(args) {
				format = args[i+1]
				i++
			}
		}

		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		result, err := tlog.CmdGraph(root, format)
		if err != nil {
			errorJSON(err.Error())
		}
		if format == "mermaid" {
			fmt.Println(result)
		} else {
			outputJSON(result)
		}

	case "prime":
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		result, err := tlog.CmdPrime(root)
		if err != nil {
			errorJSON(err.Error())
		}
		fmt.Print(result)

	case "labels":
		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		result, err := tlog.CmdLabels(root)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "sync":
		message := ""
		for i := 0; i < len(args); i++ {
			if (args[i] == "--message" || args[i] == "-m") && i+1 < len(args) {
				message = args[i+1]
				i++
			}
		}

		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		result, err := tlog.CmdSync(root, message)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	default:
		errorJSON(fmt.Sprintf("unknown command: %s", cmd))
	}
}
