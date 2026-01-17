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
func CmdCreate(root, title string, deps, labels []string, description, notes string, priority *Priority, forParent string) (map[string]interface{}, error) {
	id := GenerateID()
	now := NowISO()

	if deps == nil {
		deps = []string{}
	}
	if labels == nil {
		labels = []string{}
	}

	// Load events and compute state if we need to validate deps or forParent
	var tasks map[string]*Task
	if len(deps) > 0 || forParent != "" {
		events, err := LoadAllEvents(root)
		if err != nil {
			return nil, err
		}
		tasks = ComputeState(events)

		// Validate that all dependencies exist
		for _, depID := range deps {
			if _, ok := tasks[depID]; !ok {
				return nil, fmt.Errorf("dependency task not found: %s", depID)
			}
		}

		// Validate that forParent exists
		if forParent != "" {
			if _, ok := tasks[forParent]; !ok {
				return nil, fmt.Errorf("parent task not found: %s", forParent)
			}
		}
	}

	event := Event{
		ID:          id,
		Timestamp:   now,
		Type:        EventCreate,
		Title:       title,
		Status:      StatusOpen,
		Priority:    priority,
		Deps:        deps,
		Labels:      labels,
		Description: description,
		Notes:       notes,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	// If forParent is specified, add this task as a dependency of the parent
	if forParent != "" {
		depEvent := Event{
			ID:        forParent,
			Timestamp: NowISO(),
			Type:      EventDep,
			Dep:       id,
			Action:    "add",
		}
		if err := AppendEvent(root, depEvent); err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{
		"id":        id,
		"title":     title,
		"status":    StatusOpen,
		"deps":      deps,
		"created":   now,
		"forParent": forParent,
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

// CmdDelete marks a task as deleted (tombstone)
func CmdDelete(root, id, notes string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)
	task, ok := tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	if task.Deleted {
		return nil, fmt.Errorf("task already deleted: %s", id)
	}

	now := NowISO()
	event := Event{
		ID:        id,
		Timestamp: now,
		Type:      EventDelete,
		Notes:     notes,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      id,
		"deleted": now,
	}, nil
}

// CmdUpdate updates a task's title, description, notes, or labels
func CmdUpdate(root, id, title, description, notes string, labels []string, priority *Priority) (map[string]interface{}, error) {
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
		Priority:    priority,
	}

	if err := AppendEvent(root, event); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      id,
		"updated": now,
	}, nil
}

// CmdList lists tasks with optional status, label, and priority filters
func CmdList(root string, statusFilter string, labelFilter string, priorityFilter string) (map[string]interface{}, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return nil, err
	}

	tasks := ComputeState(events)

	var taskList []*Task
	for _, task := range tasks {
		// Exclude deleted tasks
		if task.Deleted {
			continue
		}

		// Check status filter
		statusMatch := statusFilter == "all" ||
			(statusFilter == "open" && task.Status == StatusOpen) ||
			(statusFilter == "in_progress" && task.Status == StatusInProgress) ||
			(statusFilter == "done" && task.Status == StatusDone)
		if !statusMatch {
			continue
		}

		// Check priority filter
		if priorityFilter != "" {
			if task.Priority.String() != priorityFilter {
				continue
			}
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

	// Sort by priority (ascending), then created time (descending)
	sort.Slice(taskList, func(i, j int) bool {
		if taskList[i].Priority != taskList[j].Priority {
			return taskList[i].Priority < taskList[j].Priority
		}
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
	if task.Deleted {
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

	// Sort by priority (ascending), then created time (ascending)
	sort.Slice(ready, func(i, j int) bool {
		if ready[i].Priority != ready[j].Priority {
			return ready[i].Priority < ready[j].Priority
		}
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

	// Check for circular dependency when adding
	if action == "add" {
		if WouldCreateCycle(tasks, id, depID) {
			return nil, fmt.Errorf("circular dependency: adding %s as dependency of %s would create a cycle", depID, id)
		}
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

// CmdGraph returns the dependency graph as readable text
func CmdGraph(root string) (string, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return "", err
	}

	tasks := ComputeState(events)
	return FormatDependencyTree(tasks), nil
}

// FormatDependencyTree renders tasks as a goal decomposition tree
// Root = top-level goals (tasks nothing depends on), Leaves = ready tasks
func FormatDependencyTree(tasks map[string]*Task) string {
	var sb strings.Builder

	// Find non-done, non-deleted tasks
	active := make(map[string]*Task)
	for id, t := range tasks {
		if t.Status != StatusDone && !t.Deleted {
			active[id] = t
		}
	}

	if len(active) == 0 {
		return "No active tasks"
	}

	// Build set of tasks that have dependents (are depended on by others)
	hasDependents := make(map[string]bool)
	for _, t := range active {
		for _, depID := range t.Deps {
			if _, ok := active[depID]; ok {
				hasDependents[depID] = true
			}
		}
	}

	// Root tasks: active tasks that no other active task depends on (top-level goals)
	var roots []*Task
	for _, t := range active {
		if !hasDependents[t.ID] {
			roots = append(roots, t)
		}
	}

	// Sort: in_progress first, then by priority, then by created time
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].Status != roots[j].Status {
			return roots[i].Status == StatusInProgress
		}
		if roots[i].Priority != roots[j].Priority {
			return roots[i].Priority < roots[j].Priority
		}
		return roots[i].Created.Before(roots[j].Created)
	})

	// Render each root task with its dependencies (subtasks)
	for i, task := range roots {
		if i > 0 {
			sb.WriteString("\n")
		}
		seen := make(map[string]bool)
		renderTaskTree(&sb, task, active, "", "", seen)
	}

	return sb.String()
}

// renderTaskTree recursively renders a task and its dependencies (subtasks)
func renderTaskTree(sb *strings.Builder, task *Task, active map[string]*Task, prefix string, connector string, seen map[string]bool) {
	// Cycle detection
	if seen[task.ID] {
		return
	}
	seen[task.ID] = true

	// Status symbol
	var status string
	switch task.Status {
	case StatusInProgress:
		status = "◐"
	case StatusDone:
		status = "●"
	default:
		status = "○" // open
	}

	// Render this task
	fmt.Fprintf(sb, "%s%s%s %s  %s\n", prefix, connector, status, task.ID, task.Title)

	// Get active dependencies (subtasks that need to be done first)
	var deps []*Task
	for _, depID := range task.Deps {
		if dep, ok := active[depID]; ok {
			deps = append(deps, dep)
		}
	}
	if len(deps) == 0 {
		return
	}

	// Sort by priority, then by created time
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].Priority != deps[j].Priority {
			return deps[i].Priority < deps[j].Priority
		}
		return deps[i].Created.Before(deps[j].Created)
	})

	// Calculate child prefix based on current connector
	var childPrefix string
	switch connector {
	case "├─ ":
		childPrefix = prefix + "│  "
	case "└─ ":
		childPrefix = prefix + "   "
	default:
		childPrefix = prefix
	}

	for i, dep := range deps {
		isLast := i == len(deps)-1
		childConnector := "├─ "
		if isLast {
			childConnector = "└─ "
		}
		renderTaskTree(sb, dep, active, childPrefix, childConnector, seen)
	}
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
  --note         logs what happened (append-only)
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

	// Collect unique labels (excluding deleted tasks)
	labelSet := make(map[string]bool)
	for _, task := range tasks {
		if task.Deleted {
			continue
		}
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

// CmdCompact compacts old event files into single-line task snapshots
func CmdCompact(root string, dryRun bool) (map[string]interface{}, error) {
	files, err := ListEventFiles(root)
	if err != nil {
		return nil, err
	}

	today := TodayStr() + ".jsonl"

	// Find files to compact (all except today's)
	var filesToCompact []string
	for _, f := range files {
		if f != today {
			filesToCompact = append(filesToCompact, f)
		}
	}

	if len(filesToCompact) == 0 {
		return map[string]interface{}{
			"status":       "nothing to compact",
			"files_before": len(files),
			"files_after":  len(files),
		}, nil
	}

	// Load events from files to compact
	var events []Event
	for _, f := range filesToCompact {
		fileEvents, err := LoadEventsFromFile(root, f)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", f, err)
		}
		events = append(events, fileEvents...)
	}

	// Sort events by timestamp for correct state computation
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	// Compute state from these events
	tasks := ComputeState(events)

	// Generate snapshot create events for non-deleted tasks
	// Use task's Created timestamp to preserve chronological ordering with today's events
	var snapshotEvents []Event
	for _, task := range tasks {
		if task.Deleted {
			continue
		}
		priority := task.Priority
		snapshotEvents = append(snapshotEvents, Event{
			ID:          task.ID,
			Timestamp:   task.Created,
			Type:        EventCreate,
			Title:       task.Title,
			Status:      task.Status,
			Resolution:  task.Resolution,
			Priority:    &priority,
			Deps:        task.Deps,
			Labels:      task.Labels,
			Description: task.Description,
			Notes:       task.Notes,
		})
	}

	if dryRun {
		return map[string]interface{}{
			"status":          "dry run",
			"files_to_remove": filesToCompact,
			"events_before":   len(events),
			"tasks_after":     len(snapshotEvents),
		}, nil
	}

	// Write compacted file
	compactedFilename := "compacted.jsonl"
	if err := WriteEventsToFile(root, compactedFilename, snapshotEvents); err != nil {
		return nil, fmt.Errorf("writing compacted file: %w", err)
	}

	// Delete old files
	var deletedFiles []string
	for _, f := range filesToCompact {
		if err := DeleteEventFile(root, f); err != nil {
			return nil, fmt.Errorf("deleting %s: %w", f, err)
		}
		deletedFiles = append(deletedFiles, f)
	}

	return map[string]interface{}{
		"status":        "compacted",
		"compacted_to":  compactedFilename,
		"files_removed": deletedFiles,
		"events_before": len(events),
		"tasks_after":   len(snapshotEvents),
	}, nil
}
