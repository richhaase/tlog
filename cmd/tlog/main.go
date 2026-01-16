package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/richhaase/tlog/internal/tlog"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tlog",
	Short: "Append-only task tracking for AI agents",
	Long:  `tlog - append-only task tracking for AI agents`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Version command
	rootCmd.AddCommand(&cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(buildVersionString())
		},
	})

	// Init command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize tlog in current directory",
		Run: func(cmd *cobra.Command, args []string) {
			cwd, _ := os.Getwd()
			result, err := tlog.CmdInit(cwd)
			if err != nil {
				exitError(err.Error())
			}
			fmt.Printf("Initialized: %s\n", result["path"])
		},
	})

	// Create command
	createCmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			title := args[0]
			deps, _ := cmd.Flags().GetStringSlice("dep")
			labels, _ := cmd.Flags().GetStringSlice("label")
			description, _ := cmd.Flags().GetString("description")
			notes, _ := cmd.Flags().GetString("notes")
			priorityStr, _ := cmd.Flags().GetString("priority")
			forParent, _ := cmd.Flags().GetString("for")

			var priority *tlog.Priority
			if priorityStr != "" {
				p := tlog.ParsePriority(priorityStr)
				priority = &p
			}

			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}

			// Resolve forParent ID if provided
			if forParent != "" {
				forParent = resolveID(root, forParent)
			}

			result, err := tlog.CmdCreate(root, title, deps, labels, description, notes, priority, forParent)
			if err != nil {
				exitError(err.Error())
			}
			fmt.Printf("Created: %s %q\n", result["id"], result["title"])
		},
	}
	createCmd.Flags().StringSlice("dep", nil, "Add dependency (repeatable)")
	createCmd.Flags().StringSlice("label", nil, "Add label (repeatable)")
	createCmd.Flags().String("description", "", "Set description (what this task is)")
	createCmd.Flags().String("notes", "", "Add notes (what happened)")
	createCmd.Flags().String("priority", "", "Set priority (critical|high|medium|low|backlog)")
	createCmd.Flags().String("for", "", "Add as subtask of parent task (parent will depend on this task)")
	rootCmd.AddCommand(createCmd)

	// Done command
	doneCmd := &cobra.Command{
		Use:   "done <id>",
		Short: "Mark task as done",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			id := resolveID(root, args[0])

			var resolution tlog.Resolution
			if wontfix, _ := cmd.Flags().GetBool("wontfix"); wontfix {
				resolution = tlog.ResolutionWontfix
			} else if duplicate, _ := cmd.Flags().GetBool("duplicate"); duplicate {
				resolution = tlog.ResolutionDuplicate
			}
			notes, _ := cmd.Flags().GetString("note")

			result, err := tlog.CmdDone(root, id, resolution, notes)
			if err != nil {
				exitError(err.Error())
			}
			fmt.Printf("Done: %s (%s)\n", result["id"], result["resolution"])
		},
	}
	doneCmd.Flags().Bool("wontfix", false, "Resolution: wontfix")
	doneCmd.Flags().Bool("duplicate", false, "Resolution: duplicate")
	doneCmd.Flags().String("note", "", "Append closing note")
	rootCmd.AddCommand(doneCmd)

	// Claim command
	claimCmd := &cobra.Command{
		Use:   "claim <id>",
		Short: "Mark task as in_progress",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			id := resolveID(root, args[0])
			notes, _ := cmd.Flags().GetString("note")

			result, err := tlog.CmdClaim(root, id, notes)
			if err != nil {
				exitError(err.Error())
			}
			fmt.Printf("Claimed: %s\n", result["id"])
		},
	}
	claimCmd.Flags().String("note", "", "Append note")
	rootCmd.AddCommand(claimCmd)

	// Unclaim command
	unclaimCmd := &cobra.Command{
		Use:   "unclaim <id>",
		Short: "Release claimed task back to open",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			id := resolveID(root, args[0])
			notes, _ := cmd.Flags().GetString("note")

			result, err := tlog.CmdUnclaim(root, id, notes)
			if err != nil {
				exitError(err.Error())
			}
			fmt.Printf("Unclaimed: %s\n", result["id"])
		},
	}
	unclaimCmd.Flags().String("note", "", "Append note")
	rootCmd.AddCommand(unclaimCmd)

	// Reopen command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "reopen <id>",
		Short: "Reopen task (from done or in_progress)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			id := resolveID(root, args[0])
			result, err := tlog.CmdReopen(root, id)
			if err != nil {
				exitError(err.Error())
			}
			fmt.Printf("Reopened: %s\n", result["id"])
		},
	})

	// Update command
	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			id := resolveID(root, args[0])

			title, _ := cmd.Flags().GetString("title")
			description, _ := cmd.Flags().GetString("description")
			notes, _ := cmd.Flags().GetString("notes")
			labels, _ := cmd.Flags().GetStringSlice("label")
			priorityStr, _ := cmd.Flags().GetString("priority")

			var priority *tlog.Priority
			if priorityStr != "" {
				p := tlog.ParsePriority(priorityStr)
				priority = &p
			}

			result, err := tlog.CmdUpdate(root, id, title, description, notes, labels, priority)
			if err != nil {
				exitError(err.Error())
			}
			fmt.Printf("Updated: %s\n", result["id"])
		},
	}
	updateCmd.Flags().String("title", "", "New title")
	updateCmd.Flags().String("description", "", "Set description (overwrites)")
	updateCmd.Flags().String("notes", "", "Append notes")
	updateCmd.Flags().StringSlice("label", nil, "Set labels (repeatable)")
	updateCmd.Flags().String("priority", "", "Set priority (critical|high|medium|low|backlog)")
	rootCmd.AddCommand(updateCmd)

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Run: func(cmd *cobra.Command, args []string) {
			status, _ := cmd.Flags().GetString("status")
			label, _ := cmd.Flags().GetString("label")
			priority, _ := cmd.Flags().GetString("priority")

			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			result, err := tlog.CmdList(root, status, label, priority)
			if err != nil {
				exitError(err.Error())
			}
			tasks := result["tasks"].([]*tlog.Task)
			if len(tasks) == 0 {
				fmt.Println("No tasks")
			} else {
				for _, t := range tasks {
					extra := ""
					if t.Priority != tlog.PriorityMedium {
						extra = " !" + t.Priority.String()
					}
					if len(t.Labels) > 0 {
						extra += " [" + strings.Join(t.Labels, ", ") + "]"
					}
					fmt.Printf("%s  %s (%s)%s\n", t.ID, t.Title, t.Status, extra)
				}
			}
		},
	}
	listCmd.Flags().String("status", "open", "Filter by status (open|in_progress|done|all)")
	listCmd.Flags().String("label", "", "Filter by label")
	listCmd.Flags().String("priority", "", "Filter by priority (critical|high|medium|low|backlog)")
	rootCmd.AddCommand(listCmd)

	// Show command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "show <id>",
		Short: "Show task details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			id := resolveID(root, args[0])
			result, err := tlog.CmdShow(root, id)
			if err != nil {
				exitError(err.Error())
			}
			task := result["task"].(*tlog.Task)
			fmt.Printf("%s: %s\n", task.ID, task.Title)
			fmt.Printf("Status: %s\n", task.Status)
			fmt.Printf("Priority: %s\n", task.Priority)
			if task.Description != "" {
				fmt.Printf("Description: %s\n", task.Description)
			}
			if len(task.Labels) > 0 {
				fmt.Printf("Labels: %s\n", strings.Join(task.Labels, ", "))
			}
			if deps, ok := result["dep_status"].([]map[string]interface{}); ok && len(deps) > 0 {
				fmt.Print("Deps:")
				for _, d := range deps {
					fmt.Printf(" %s(%s)", d["id"], d["status"])
				}
				fmt.Println()
			}
			if task.Notes != "" {
				fmt.Printf("Notes: %s\n", task.Notes)
			}
		},
	})

	// Ready command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "ready",
		Short: "List tasks ready to work on",
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			result, err := tlog.CmdReady(root)
			if err != nil {
				exitError(err.Error())
			}
			tasks := result["tasks"].([]*tlog.Task)
			if len(tasks) == 0 {
				fmt.Println("No tasks ready")
			} else {
				for _, t := range tasks {
					extra := ""
					if t.Priority != tlog.PriorityMedium {
						extra = " !" + t.Priority.String()
					}
					if len(t.Labels) > 0 {
						extra += " [" + strings.Join(t.Labels, ", ") + "]"
					}
					fmt.Printf("%s  %s%s\n", t.ID, t.Title, extra)
				}
			}
		},
	})

	// Backlog command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "backlog",
		Short: "List backlog tasks",
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			result, err := tlog.CmdList(root, "open", "", "backlog")
			if err != nil {
				exitError(err.Error())
			}
			tasks := result["tasks"].([]*tlog.Task)
			if len(tasks) == 0 {
				fmt.Println("No backlog tasks")
			} else {
				for _, t := range tasks {
					extra := ""
					if len(t.Labels) > 0 {
						extra = " [" + strings.Join(t.Labels, ", ") + "]"
					}
					fmt.Printf("%s  %s%s\n", t.ID, t.Title, extra)
				}
			}
		},
	})

	// Dep command
	depCmd := &cobra.Command{
		Use:   "dep <id> --needs <dep-ids...>",
		Short: "Add or remove dependencies",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			needs, _ := cmd.Flags().GetStringSlice("needs")
			remove, _ := cmd.Flags().GetStringSlice("remove")

			if len(needs) == 0 && len(remove) == 0 {
				exitError("must specify --needs or --remove with one or more task IDs")
			}

			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			id := resolveID(root, args[0])

			// Add dependencies
			for _, dep := range needs {
				depID := resolveID(root, dep)
				result, err := tlog.CmdDep(root, id, depID, "add")
				if err != nil {
					exitError(err.Error())
				}
				fmt.Printf("Dep added: %s -> %s\n", result["id"], result["dep"])
			}

			// Remove dependencies
			for _, dep := range remove {
				depID := resolveID(root, dep)
				result, err := tlog.CmdDep(root, id, depID, "remove")
				if err != nil {
					exitError(err.Error())
				}
				fmt.Printf("Dep removed: %s -> %s\n", result["id"], result["dep"])
			}
		},
	}
	depCmd.Flags().StringSlice("needs", nil, "Add dependencies (task must complete before this one)")
	depCmd.Flags().StringSlice("remove", nil, "Remove dependencies")
	rootCmd.AddCommand(depCmd)

	// Graph command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "graph",
		Short: "Show dependency tree",
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			result, err := tlog.CmdGraph(root)
			if err != nil {
				exitError(err.Error())
			}
			fmt.Print(result)
		},
	})

	// Prime command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "prime",
		Short: "Get AI agent context",
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			result, err := tlog.CmdPrime(root)
			if err != nil {
				exitError(err.Error())
			}
			fmt.Print(result)
		},
	})

	// Labels command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "labels",
		Short: "Show labels in use and conventions",
		Run: func(cmd *cobra.Command, args []string) {
			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			result, err := tlog.CmdLabels(root)
			if err != nil {
				exitError(err.Error())
			}
			inUse := result["in_use"].([]string)
			if len(inUse) > 0 {
				fmt.Println("Labels in use:")
				for _, label := range inUse {
					fmt.Printf("  %s\n", label)
				}
			} else {
				fmt.Println("No labels in use")
			}
		},
	})

	// Sync command
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Commit .tlog to git",
		Run: func(cmd *cobra.Command, args []string) {
			message, _ := cmd.Flags().GetString("message")

			root, err := tlog.RequireTlog()
			if err != nil {
				exitError(err.Error())
			}
			result, err := tlog.CmdSync(root, message)
			if err != nil {
				exitError(err.Error())
			}
			fmt.Printf("Synced: %s\n", result["message"])
		},
	}
	syncCmd.Flags().StringP("message", "m", "", "Commit message")
	rootCmd.AddCommand(syncCmd)
}

func exitError(msg string) {
	fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	os.Exit(1)
}

func resolveID(root, prefix string) string {
	events, err := tlog.LoadAllEvents(root)
	if err != nil {
		exitError(err.Error())
	}
	tasks := tlog.ComputeState(events)
	id, err := tlog.ResolveID(tasks, prefix)
	if err != nil {
		exitError(err.Error())
	}
	return id
}
