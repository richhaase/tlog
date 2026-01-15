package tlog

import "fmt"

// ComputeState replays events to build current task state
func ComputeState(events []Event) map[string]*Task {
	tasks := make(map[string]*Task)

	for _, event := range events {
		switch event.Type {
		case EventCreate:
			tasks[event.ID] = &Task{
				ID:      event.ID,
				Title:   event.Title,
				Status:  StatusOpen,
				Deps:    event.Deps,
				Blocks:  event.Blocks,
				Created: event.Timestamp,
				Updated: event.Timestamp,
				Labels:  event.Labels,
				Notes:   event.Notes,
			}
			if tasks[event.ID].Deps == nil {
				tasks[event.ID].Deps = []string{}
			}
			if tasks[event.ID].Blocks == nil {
				tasks[event.ID].Blocks = []string{}
			}
			if tasks[event.ID].Labels == nil {
				tasks[event.ID].Labels = []string{}
			}

		case EventStatus:
			if task, ok := tasks[event.ID]; ok {
				task.Status = event.Status
				task.Updated = event.Timestamp
			}

		case EventDep:
			if task, ok := tasks[event.ID]; ok {
				if event.Action == "add" {
					task.Deps = appendUnique(task.Deps, event.Dep)
				} else if event.Action == "remove" {
					task.Deps = removeItem(task.Deps, event.Dep)
				}
				task.Updated = event.Timestamp
			}

		case EventBlock:
			if task, ok := tasks[event.ID]; ok {
				if event.Action == "add" {
					task.Blocks = appendUnique(task.Blocks, event.Block)
				} else if event.Action == "remove" {
					task.Blocks = removeItem(task.Blocks, event.Block)
				}
				task.Updated = event.Timestamp
			}

		case EventUpdate:
			if task, ok := tasks[event.ID]; ok {
				if event.Title != "" {
					task.Title = event.Title
				}
				if event.Notes != "" {
					task.Notes = event.Notes
				}
				if event.Labels != nil {
					task.Labels = event.Labels
				}
				task.Updated = event.Timestamp
			}
		}
	}

	return tasks
}

// GetReadyTasks returns tasks that are open, have all deps done, and are not blocked
func GetReadyTasks(tasks map[string]*Task) []*Task {
	// Build reverse block map: which tasks are blocked by which
	blockedBy := make(map[string][]string)
	for _, task := range tasks {
		for _, blocksID := range task.Blocks {
			blockedBy[blocksID] = append(blockedBy[blocksID], task.ID)
		}
	}

	var ready []*Task
	for _, task := range tasks {
		if task.Status != StatusOpen {
			continue
		}

		// Check if all dependencies are done
		allDepsDone := true
		for _, depID := range task.Deps {
			if depTask, ok := tasks[depID]; ok {
				if depTask.Status != StatusDone {
					allDepsDone = false
					break
				}
			}
		}
		if !allDepsDone {
			continue
		}

		// Check if blocked by any open task
		isBlocked := false
		for _, blockerID := range blockedBy[task.ID] {
			if blocker, ok := tasks[blockerID]; ok {
				if blocker.Status == StatusOpen {
					isBlocked = true
					break
				}
			}
		}
		if isBlocked {
			continue
		}

		ready = append(ready, task)
	}

	return ready
}

// BuildDependencyGraph builds a graph of task dependencies
func BuildDependencyGraph(tasks map[string]*Task) Graph {
	var nodes []GraphNode
	var edges []GraphEdge

	for _, task := range tasks {
		nodes = append(nodes, GraphNode{
			ID:     task.ID,
			Title:  task.Title,
			Status: task.Status,
		})

		for _, depID := range task.Deps {
			edges = append(edges, GraphEdge{
				From: depID,
				To:   task.ID,
				Type: "depends_on",
			})
		}

		for _, blockID := range task.Blocks {
			edges = append(edges, GraphEdge{
				From: task.ID,
				To:   blockID,
				Type: "blocks",
			})
		}
	}

	return Graph{Nodes: nodes, Edges: edges}
}

// Helper functions
func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func removeItem(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// ResolveID resolves a prefix to a full task ID.
// Accepts "tl-abc", "abc", or full ID. Returns error if no match or ambiguous.
func ResolveID(tasks map[string]*Task, prefix string) (string, error) {
	// Normalize: strip "tl-" prefix if present
	search := prefix
	if len(prefix) > 3 && prefix[:3] == "tl-" {
		search = prefix[3:]
	}

	var matches []string
	for id := range tasks {
		// Extract hex part (after "tl-")
		hex := id[3:]
		if hex == search || id == prefix {
			// Exact match
			return id, nil
		}
		if len(search) <= len(hex) && hex[:len(search)] == search {
			matches = append(matches, id)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no task found matching '%s'", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous prefix '%s' matches %d tasks: %v", prefix, len(matches), matches)
	}
}
