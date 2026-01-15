package tlog

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// CmdInit initializes a new tlog repository
func CmdInit(path string) (map[string]interface{}, error) {
	if err := Initialize(path); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":  "initialized",
		"path":    path + "/" + TlogDir,
		"message": "tlog initialized. Add .tlog/ to git.",
	}, nil
}

// CmdCreate creates a new task
func CmdCreate(root, title string, deps, blocks, labels []string, description, notes string) (map[string]interface{}, error) {
	id := GenerateID()
	now := NowISO()

	if deps == nil {
		deps = []string{}
	}
	if blocks == nil {
		blocks = []string{}
	}
	if labels == nil {
		labels = []string{}
	}

	event := Event{
		ID:          id,
		Timestamp:   now,
		Type:        EventCreate,
		Title:       title,
		Status:      StatusOpen,
		Deps:        deps,
		Blocks:      blocks,
		Labels:      labels,
		Description: description,
		Notes:       notes,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      id,
		"title":   title,
		"status":  StatusOpen,
		"deps":    deps,
		"blocks":  blocks,
		"created": now,
	}, nil
}

// CmdDone marks a task as done
func CmdDone(root, id string, resolution Resolution, notes string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	if _, ok := tasks[id]; !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	if resolution == "" {
		resolution = ResolutionCompleted
	}

	now := NowISO()
	event := Event{
		ID:         id,
		Timestamp:  now,
		Type:       EventStatus,
		Status:     StatusDone,
		Resolution: resolution,
		Notes:      notes,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":         id,
		"status":     StatusDone,
		"resolution": resolution,
		"completed":  now,
	}, nil
}

// CmdClaim marks a task as in_progress
func CmdClaim(root, id, notes string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	task, ok := tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	if task.Status != StatusOpen {
		return nil, fmt.Errorf("can only claim open tasks, task is %s", task.Status)
	}

	now := NowISO()
	event := Event{
		ID:        id,
		Timestamp: now,
		Type:      EventStatus,
		Status:    StatusInProgress,
		Notes:     notes,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      id,
		"status":  StatusInProgress,
		"claimed": now,
	}, nil
}

// CmdUnclaim releases a claimed task back to open
func CmdUnclaim(root, id, notes string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	task, ok := tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	if task.Status != StatusInProgress {
		return nil, fmt.Errorf("can only unclaim in_progress tasks, task is %s", task.Status)
	}

	now := NowISO()
	event := Event{
		ID:        id,
		Timestamp: now,
		Type:      EventStatus,
		Status:    StatusOpen,
		Notes:     notes,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":        id,
		"status":    StatusOpen,
		"unclaimed": now,
	}, nil
}

// CmdReopen reopens a task (from done or in_progress back to open)
func CmdReopen(root, id string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	if _, ok := tasks[id]; !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	now := NowISO()
	event := Event{
		ID:        id,
		Timestamp: now,
		Type:      EventStatus,
		Status:    StatusOpen,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":       id,
		"status":   StatusOpen,
		"reopened": now,
	}, nil
}

// CmdUpdate updates a task's title, description, notes, or labels
func CmdUpdate(root, id, title, description, notes string, labels []string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	if _, ok := tasks[id]; !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	now := NowISO()
	event := Event{
		ID:          id,
		Timestamp:   now,
		Type:        EventUpdate,
		Title:       title,
		Description: description,
		Notes:       notes,
		Labels:      labels,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      id,
		"updated": now,
	}, nil
}

// CmdList lists tasks with optional status and label filters
func CmdList(root string, statusFilter string, labelFilter string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)

	var taskList []*Task
	for _, task := range tasks {
		// Check status filter
		statusMatch := statusFilter == "all" ||
			(statusFilter == "open" && task.Status == StatusOpen) ||
			(statusFilter == "in_progress" && task.Status == StatusInProgress) ||
			(statusFilter == "done" && task.Status == StatusDone)
		if !statusMatch {
			continue
		}

		// Check label filter
		if labelFilter != "" {
			hasLabel := false
			for _, label := range task.Labels {
				if label == labelFilter {
					hasLabel = true
					break
				}
			}
			if !hasLabel {
				continue
			}
		}

		taskList = append(taskList, task)
	}

	// Sort by created time descending
	sort.Slice(taskList, func(i, j int) bool {
		return taskList[i].Created.After(taskList[j].Created)
	})

	return map[string]interface{}{
		"tasks": taskList,
		"count": len(taskList),
	}, nil
}

// CmdShow shows details of a single task
func CmdShow(root, id string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	task, ok := tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	// Get dependency status (tasks this task depends on)
	depStatus := make([]map[string]interface{}, 0)
	for _, depID := range task.Deps {
		if depTask, ok := tasks[depID]; ok {
			depStatus = append(depStatus, map[string]interface{}{
				"id":     depID,
				"title":  depTask.Title,
				"status": depTask.Status,
			})
		}
	}

	// Get blocked_by (tasks that have this task in their blocks array)
	blockedBy := make([]map[string]interface{}, 0)
	for _, other := range tasks {
		for _, blockID := range other.Blocks {
			if blockID == id {
				blockedBy = append(blockedBy, map[string]interface{}{
					"id":     other.ID,
					"title":  other.Title,
					"status": other.Status,
				})
				break
			}
		}
	}

	// Get dependents (tasks that have this task in their deps array)
	dependents := make([]map[string]interface{}, 0)
	for _, other := range tasks {
		for _, depID := range other.Deps {
			if depID == id {
				dependents = append(dependents, map[string]interface{}{
					"id":     other.ID,
					"title":  other.Title,
					"status": other.Status,
				})
				break
			}
		}
	}

	return map[string]interface{}{
		"task":       task,
		"dep_status": depStatus,
		"blocked_by": blockedBy,
		"dependents": dependents,
	}, nil
}

// CmdReady returns tasks ready to be worked on
func CmdReady(root string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	ready := GetReadyTasks(tasks)

	// Sort by created time
	sort.Slice(ready, func(i, j int) bool {
		return ready[i].Created.Before(ready[j].Created)
	})

	return map[string]interface{}{
		"tasks": ready,
		"count": len(ready),
	}, nil
}

// CmdDep adds or removes a dependency
func CmdDep(root, id, depID, action string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	if _, ok := tasks[id]; !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	if _, ok := tasks[depID]; !ok {
		return nil, fmt.Errorf("dependency task not found: %s", depID)
	}

	now := NowISO()
	event := Event{
		ID:        id,
		Timestamp: now,
		Type:      EventDep,
		Dep:       depID,
		Action:    action,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      id,
		"dep":     depID,
		"action":  action,
		"updated": now,
	}, nil
}

// CmdBlock adds or removes a blocking relationship
func CmdBlock(root, id, blockID, action string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	if _, ok := tasks[id]; !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	if _, ok := tasks[blockID]; !ok {
		return nil, fmt.Errorf("blocked task not found: %s", blockID)
	}

	now := NowISO()
	event := Event{
		ID:        id,
		Timestamp: now,
		Type:      EventBlock,
		Block:     blockID,
		Action:    action,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      id,
		"blocks":  blockID,
		"action":  action,
		"updated": now,
	}, nil
}

// CmdGraph returns the dependency graph
func CmdGraph(root string, format string) (interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	graph := BuildDependencyGraph(tasks)

	if format == "mermaid" {
		return GraphToMermaid(graph, tasks), nil
	}

	return graph, nil
}

// GraphToMermaid converts a graph to Mermaid diagram format
func GraphToMermaid(graph Graph, tasks map[string]*Task) string {
	var sb strings.Builder
	sb.WriteString("graph TD\n")

	// Write nodes
	for _, node := range graph.Nodes {
		status := "○" // open
		if node.Status == StatusInProgress {
			status = "◐" // in_progress
		} else if node.Status == StatusDone {
			status = "●" // done
		}
		title := node.Title
		if len(title) > 30 {
			title = title[:30]
		}
		sb.WriteString(fmt.Sprintf("    %s[\"%s %s\"]\n", node.ID, status, title))
	}

	// Write edges
	for _, edge := range graph.Edges {
		sb.WriteString(fmt.Sprintf("    %s --> %s\n", edge.From, edge.To))
	}

	return sb.String()
}

// CmdPrime generates context for AI agents
func CmdPrime(root string) (string, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return "", err
	}

	tasks := ComputeState(events)
	ready := GetReadyTasks(tasks)

	var sb strings.Builder

	sb.WriteString("tlog tracks tasks for AI agents in this project.\n\n")

	sb.WriteString(`Workflow:
1. claim a task before starting (prevents duplicate work)
2. done when finished
3. unclaim if you hit a blocker and need to release it

Commands:
  tlog ready           list tasks ready to work on
  tlog claim <id>      claim a task and start working
  tlog done <id>       mark task complete
  tlog unclaim <id>    release task if blocked
  tlog create "title"  create a new task

Tips:
  --description  sets what the task is (mutable, overwrites)
  --notes        logs what happened (append-only)
`)

	if len(ready) > 0 {
		sb.WriteString("\nReady tasks:\n")
		for _, t := range ready {
			labels := ""
			if len(t.Labels) > 0 {
				labels = " [" + strings.Join(t.Labels, ", ") + "]"
			}
			sb.WriteString(fmt.Sprintf("  %s  %s%s\n", t.ID, t.Title, labels))
		}
	} else {
		sb.WriteString("\nNo tasks ready. Use 'tlog create \"title\"' to create one.\n")
	}

	return sb.String(), nil
}

// CmdLabels shows labels in use and recommended conventions
func CmdLabels(root string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)

	// Collect unique labels
	labelSet := make(map[string]bool)
	for _, task := range tasks {
		for _, label := range task.Labels {
			labelSet[label] = true
		}
	}

	var labels []string
	for label := range labelSet {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	recommended := map[string][]string{
		"priority": {"backlog", "low", "medium", "high", "critical"},
		"type":     {"feature", "bug", "refactor", "chore"},
		"needs":    {"human-review", "agent-review", "discussion", "design"},
	}

	return map[string]interface{}{
		"in_use":      labels,
		"recommended": recommended,
		"note":        "Use feature:<name> for freeform grouping",
	}, nil
}

// CmdSync commits .tlog to git
func CmdSync(root, message string) (map[string]interface{}, error) {
	if message == "" {
		message = "tlog: sync tasks"
	}

	// git add .tlog/
	addCmd := exec.Command("git", "add", root)
	if err := addCmd.Run(); err != nil {
		return nil, fmt.Errorf("git add failed: %w", err)
	}

	// git commit
	commitCmd := exec.Command("git", "commit", "-m", message)
	if err := commitCmd.Run(); err != nil {
		return nil, fmt.Errorf("git commit failed: %w", err)
	}

	return map[string]interface{}{
		"status":  "synced",
		"message": message,
	}, nil
}
