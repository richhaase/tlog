package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rdh/tlog/internal/tlog"
)

func outputJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func errorJSON(msg string) {
	outputJSON(map[string]string{"error": msg})
	os.Exit(1)
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
    --notes <text>        Add notes
  done <id>               Mark task as done
  reopen <id>             Reopen a completed task
  update <id>             Update task
    --title <text>        New title
    --notes <text>        New notes
    --label <label>       Set labels (repeatable)
  list [--status <s>]     List tasks (open|done|all, default: open)
  show <id>               Show task details
  ready                   List tasks ready to work on
  dep <id> <dep-id>       Add dependency
    --remove              Remove instead of add
  block <id> <block-id>   Add blocking relationship
    --remove              Remove instead of add
  graph [--format <f>]    Show dependency graph (json|mermaid)
  prime                   Get AI agent context
  sync [--message <m>]    Commit .tlog to git
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
		var notes string

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
		result, err := tlog.CmdCreate(root, title, deps, blocks, labels, notes)
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
		result, err := tlog.CmdDone(root, args[0])
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
		result, err := tlog.CmdReopen(root, args[0])
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "update":
		if len(args) < 1 {
			errorJSON("update requires a task ID")
		}
		id := args[0]
		var title, notes string
		var labels []string

		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--title":
				if i+1 < len(args) {
					title = args[i+1]
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

		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		result, err := tlog.CmdUpdate(root, id, title, notes, labels)
		if err != nil {
			errorJSON(err.Error())
		}
		outputJSON(result)

	case "list":
		status := "open"
		for i := 0; i < len(args); i++ {
			if args[i] == "--status" && i+1 < len(args) {
				status = args[i+1]
				i++
			}
		}

		root, err := tlog.RequireTlog()
		if err != nil {
			errorJSON(err.Error())
		}
		result, err := tlog.CmdList(root, status)
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
		result, err := tlog.CmdShow(root, args[0])
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
		result, err := tlog.CmdDep(root, args[0], args[1], action)
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
		result, err := tlog.CmdBlock(root, args[0], args[1], action)
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
		outputJSON(result)

	case "sync":
		message := ""
		for i := 0; i < len(args); i++ {
			if args[i] == "--message" && i+1 < len(args) {
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
