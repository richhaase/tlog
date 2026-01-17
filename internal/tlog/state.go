package tlog

import "fmt"

// ComputeState replays events to build current task state
func ComputeState(events []Event) map[string]*Task {
	tasks := make(map[string]*Task)

	for _, event := range events {
		switch event.Type {
		case EventCreate:
			priority := PriorityMedium
			if event.Priority != nil {
				priority = *event.Priority
			}
			status := StatusOpen
			if event.Status != "" {
				status = event.Status
			}
			tasks[event.ID] = &Task{
				ID:          event.ID,
				Title:       event.Title,
				Status:      status,
				Resolution:  event.Resolution,
				Priority:    priority,
				Deps:        event.Deps,
				Created:     event.Timestamp,
				Updated:     event.Timestamp,
				Labels:      event.Labels,
				Description: event.Description,
				Notes:       event.Notes,
			}
			if tasks[event.ID].Deps == nil {
				tasks[event.ID].Deps = []string{}
			}
			if tasks[event.ID].Labels == nil {
				tasks[event.ID].Labels = []string{}
			}

		case EventStatus:
			if task, ok := tasks[event.ID]; ok {
				task.Status = event.Status
				task.Resolution = event.Resolution
				if event.Notes != "" {
					task.Notes = appendNote(task.Notes, event.Notes)
				}
				task.Updated = event.Timestamp
			}

		case EventDep:
			if task, ok := tasks[event.ID]; ok {
				switch event.Action {
				case "add":
					task.Deps = appendUnique(task.Deps, event.Dep)
				case "remove":
					task.Deps = removeItem(task.Deps, event.Dep)
				}
				task.Updated = event.Timestamp
			}

		case EventUpdate:
			if task, ok := tasks[event.ID]; ok {
				if event.Title != "" {
					task.Title = event.Title
				}
				if event.Description != "" {
					task.Description = event.Description
				}
				if event.Notes != "" {
					task.Notes = appendNote(task.Notes, event.Notes)
				}
				if event.Labels != nil {
					task.Labels = event.Labels
				}
				if event.Priority != nil {
					task.Priority = *event.Priority
				}
				task.Updated = event.Timestamp
			}

		case EventDelete:
			if task, ok := tasks[event.ID]; ok {
				task.Deleted = true
				if event.Notes != "" {
					task.Notes = appendNote(task.Notes, event.Notes)
				}
				task.Updated = event.Timestamp
			}
		}
	}

	return tasks
}

// GetReadyTasks returns tasks that are open, have all deps done, and are not backlog priority
func GetReadyTasks(tasks map[string]*Task) []*Task {
	var ready []*Task
	for _, task := range tasks {
		// Exclude deleted tasks
		if task.Deleted {
			continue
		}

		if task.Status != StatusOpen {
			continue
		}

		// Exclude backlog priority tasks
		if task.Priority == PriorityBacklog {
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
	}

	return Graph{Nodes: nodes, Edges: edges}
}

// Helper functions

// appendNote appends a new note to existing notes, separated by newlines
func appendNote(existing, newNote string) string {
	if existing == "" {
		return newNote
	}
	return existing + "\n" + newNote
}

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

// WouldCreateCycle checks if adding depID as a dependency of taskID would create a cycle.
// Returns true if adding the dependency would create a circular dependency.
func WouldCreateCycle(tasks map[string]*Task, taskID, depID string) bool {
	// Self-dependency is a cycle
	if taskID == depID {
		return true
	}

	// Check if taskID is reachable from depID (i.e., depID already depends on taskID)
	// If so, adding depID as a dependency of taskID would create a cycle
	visited := make(map[string]bool)
	return isReachable(tasks, depID, taskID, visited)
}

// isReachable checks if targetID is reachable from startID via dependencies
func isReachable(tasks map[string]*Task, startID, targetID string, visited map[string]bool) bool {
	if startID == targetID {
		return true
	}

	if visited[startID] {
		return false
	}
	visited[startID] = true

	task, ok := tasks[startID]
	if !ok {
		return false
	}

	for _, depID := range task.Deps {
		if isReachable(tasks, depID, targetID, visited) {
			return true
		}
	}

	return false
}

// ResolveID resolves a prefix to a full task ID.
// Accepts full ID or prefix. Returns error if no match or ambiguous.
// Deleted tasks are excluded from resolution.
func ResolveID(tasks map[string]*Task, prefix string) (string, error) {
	var matches []string
	for id, task := range tasks {
		if task.Deleted {
			continue
		}
		if id == prefix {
			// Exact match
			return id, nil
		}
		if len(prefix) <= len(id) && id[:len(prefix)] == prefix {
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
