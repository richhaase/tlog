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
func CmdCreate(root, title string, deps, blocks, labels []string, notes string) (map[string]interface{}, error) {
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
		ID:        id,
		Timestamp: now,
		Type:      EventCreate,
		Title:     title,
		Status:    StatusOpen,
		Deps:      deps,
		Blocks:    blocks,
		Labels:    labels,
		Notes:     notes,
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
func CmdDone(root, id string) (map[string]interface{}, error) {
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
		Status:    StatusDone,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":        id,
		"status":    StatusDone,
		"completed": now,
	}, nil
}

// CmdReopen reopens a completed task
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

// CmdUpdate updates a task's title, notes, or labels
func CmdUpdate(root, id, title, notes string, labels []string) (map[string]interface{}, error) {
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
		Type:      EventUpdate,
		Title:     title,
		Notes:     notes,
		Labels:    labels,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      id,
		"updated": now,
	}, nil
}

// CmdList lists tasks with optional status filter
func CmdList(root string, statusFilter string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)

	var taskList []*Task
	for _, task := range tasks {
		if statusFilter == "all" ||
			(statusFilter == "open" && task.Status == StatusOpen) ||
			(statusFilter == "done" && task.Status == StatusDone) {
			taskList = append(taskList, task)
		}
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

	// Get dependency status
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

	return map[string]interface{}{
		"task":       task,
		"dep_status": depStatus,
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
		status := "○"
		if node.Status == StatusDone {
			status = "●"
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
func CmdPrime(root string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	ready := GetReadyTasks(tasks)

	// Get recent completed (last 5)
	var completed []*Task
	for _, task := range tasks {
		if task.Status == StatusDone {
			completed = append(completed, task)
		}
	}
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].Updated.After(completed[j].Updated)
	})
	if len(completed) > 5 {
		completed = completed[:5]
	}

	// Get blocked tasks
	var blocked []*Task
	readyIDs := make(map[string]bool)
	for _, t := range ready {
		readyIDs[t.ID] = true
	}
	for _, task := range tasks {
		if task.Status == StatusOpen && !readyIDs[task.ID] {
			blocked = append(blocked, task)
		}
	}

	// Build summary
	openCount := 0
	doneCount := 0
	for _, task := range tasks {
		if task.Status == StatusOpen {
			openCount++
		} else {
			doneCount++
		}
	}

	summary := fmt.Sprintf("%d open, %d done, %d ready", openCount, doneCount, len(ready))

	return map[string]interface{}{
		"summary":          summary,
		"ready_tasks":      ready,
		"recent_completed": completed,
		"blocked_tasks":    blocked,
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
