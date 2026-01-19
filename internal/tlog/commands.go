package tlog

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
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
func CmdDone(root, id string, resolution Resolution, notes, commit string) (map[string]interface{}, error) {
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
		Commit:     commit,
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
func CmdPrime(root string, cliReference string) (string, error) {
	events, err := LoadAllEvents(root)
	if err != nil {
		return "", err
	}

	tasks := ComputeState(events)

	// Categorize tasks
	var ready, inProgress, blocked, recentDone []*Task
	for _, t := range tasks {
		if t.Deleted {
			continue
		}
		switch t.Status {
		case StatusInProgress:
			inProgress = append(inProgress, t)
		case StatusDone:
			recentDone = append(recentDone, t)
		case StatusOpen:
			if t.Priority == PriorityBacklog {
				continue // skip backlog
			}
			// Check if blocked on deps
			isBlocked := false
			for _, depID := range t.Deps {
				if dep, ok := tasks[depID]; ok && dep.Status != StatusDone {
					isBlocked = true
					break
				}
			}
			if isBlocked {
				blocked = append(blocked, t)
			} else {
				ready = append(ready, t)
			}
		}
	}

	// Sort ready by priority then created
	sortTasksByPriorityCreated(ready)
	sortTasksByPriorityCreated(blocked)

	// Sort recentDone by updated (most recent first), limit to 3
	sort.Slice(recentDone, func(i, j int) bool {
		return recentDone[i].Updated.After(recentDone[j].Updated)
	})
	if len(recentDone) > 3 {
		recentDone = recentDone[:3]
	}

	// Count stats
	var openCount, inProgressCount, doneCount int
	for _, t := range tasks {
		if t.Deleted {
			continue
		}
		switch t.Status {
		case StatusOpen:
			openCount++
		case StatusInProgress:
			inProgressCount++
		case StatusDone:
			doneCount++
		}
	}

	var sb strings.Builder

	sb.WriteString("tlog tracks tasks for AI agents in this project.\n\n")

	// Summary line
	sb.WriteString(fmt.Sprintf("Status: %d open, %d in-progress, %d done\n\n", openCount, inProgressCount, doneCount))

	sb.WriteString(`Workflow:
1. claim a task before starting (prevents duplicate work)
2. decompose large tasks into smaller tasks with dependencies before starting
3. commit changes before marking done
4. done when finished (use --commit to record the commit SHA)
5. unclaim if you hit a blocker and need to release it

`)

	// CLI reference (auto-generated)
	if cliReference != "" {
		sb.WriteString("Commands:\n")
		sb.WriteString(cliReference)
		sb.WriteString("\nTips:\n")
		sb.WriteString("  --description  sets what the task is (mutable, overwrites)\n")
		sb.WriteString("  --note         logs what happened (append-only)\n")
		sb.WriteString("  --for <id>     creates a subtask that blocks the parent\n")
		sb.WriteString("  partial IDs    work if unambiguous (e.g., \"tlog done 4d1\")\n")
		sb.WriteString("  sync -m \"...\" periodically to commit tlog state to git\n")
		sb.WriteString("\nPriority levels (do highest available first):\n")
		sb.WriteString("  [critical]  blocking others or time-sensitive\n")
		sb.WriteString("  [high]      important, do soon\n")
		sb.WriteString("  [medium]    normal priority (default, not shown)\n")
		sb.WriteString("  [low]       nice to have, do when time permits\n")
		sb.WriteString("  [backlog]   not actively prioritized (hidden from ready list)\n")
		sb.WriteString("\nCanonical labels (how to approach):\n")
		sb.WriteString("  spike             timeboxed research — outcome is knowledge/subtasks, not code\n")
		sb.WriteString("  needs-breakdown   too large to work directly — decompose before claiming\n")
		sb.WriteString("  blocked-external  waiting on something outside tlog's control\n")
		sb.WriteString("  wip               partially complete — context exists, needs continuation\n")
	}

	// In-progress tasks (important - shows what's being worked on)
	if len(inProgress) > 0 {
		sb.WriteString("\nIn-progress:\n")
		for _, t := range inProgress {
			sb.WriteString(fmt.Sprintf("  %s  %s%s\n", t.ID, formatPriorityPrefix(t.Priority), t.Title))
		}
	}

	// Ready tasks
	if len(ready) > 0 {
		sb.WriteString("\nReady:\n")
		for _, t := range ready {
			sb.WriteString(fmt.Sprintf("  %s  %s%s\n", t.ID, formatPriorityPrefix(t.Priority), t.Title))
		}
	}

	// Blocked tasks
	if len(blocked) > 0 {
		sb.WriteString("\nBlocked:\n")
		for _, t := range blocked {
			// Find what it's waiting on
			var waitingOn []string
			for _, depID := range t.Deps {
				if dep, ok := tasks[depID]; ok && dep.Status != StatusDone {
					waitingOn = append(waitingOn, depID[:8])
				}
			}
			sb.WriteString(fmt.Sprintf("  %s  %s%s (waiting: %s)\n", t.ID, formatPriorityPrefix(t.Priority), t.Title, strings.Join(waitingOn, ", ")))
		}
	}

	// Recent completions (context)
	if len(recentDone) > 0 {
		sb.WriteString("\nRecent:\n")
		for _, t := range recentDone {
			if t.Commit != "" {
				sb.WriteString(fmt.Sprintf("  %s  %s (%s)\n", t.ID, t.Title, t.Commit))
			} else {
				sb.WriteString(fmt.Sprintf("  %s  %s\n", t.ID, t.Title))
			}
		}
	}

	if len(ready) == 0 && len(inProgress) == 0 && len(blocked) == 0 {
		sb.WriteString("\nNo tasks. Use 'tlog create \"title\"' to create one.\n")
	}

	return sb.String(), nil
}

// sortTasksByPriorityCreated sorts by priority (asc) then created (asc)
func sortTasksByPriorityCreated(tasks []*Task) {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority < tasks[j].Priority
		}
		return tasks[i].Created.Before(tasks[j].Created)
	})
}

// formatPriorityPrefix returns a bracketed priority prefix for display.
// Returns empty string for medium priority (the default) to reduce noise.
func formatPriorityPrefix(p Priority) string {
	if p == PriorityMedium {
		return ""
	}
	return "[" + p.String() + "] "
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

// CmdPrune compacts old event files and optionally removes done tasks.
// It combines compaction and pruning into a single pass for efficiency.
// - keepAll: if true, keep all tasks (equivalent to old compact behavior)
// - saveDays: if > 0 and not keepAll, preserve done tasks from the last N days
func CmdPrune(root string, saveDays int, keepAll bool, dryRun bool) (map[string]interface{}, error) {
	files, err := ListEventFiles(root)
	if err != nil {
		return nil, err
	}

	today := TodayStr() + ".jsonl"

	// Find files to process (all except today's)
	var filesToProcess []string
	for _, f := range files {
		if f != today {
			filesToProcess = append(filesToProcess, f)
		}
	}

	if len(filesToProcess) == 0 {
		return map[string]interface{}{
			"status":       "nothing to prune",
			"files_before": len(files),
			"tasks_before": 0,
			"tasks_after":  0,
			"pruned":       0,
		}, nil
	}

	// Load events from files to process
	var events []Event
	for _, f := range filesToProcess {
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

	// Calculate cutoff for save-days
	cutoff := time.Time{}
	if saveDays > 0 && !keepAll {
		cutoff = time.Now().UTC().AddDate(0, 0, -saveDays)
	}

	// Generate snapshot events, filtering as needed
	var snapshotEvents []Event
	var prunedCount int
	for _, task := range tasks {
		if task.Deleted {
			continue
		}

		// Decide whether to keep this task
		shouldPrune := false
		if !keepAll && task.Status == StatusDone {
			if saveDays > 0 {
				// Prune if older than cutoff
				shouldPrune = task.Updated.Before(cutoff)
			} else {
				// Prune all done tasks
				shouldPrune = true
			}
		}

		if shouldPrune {
			prunedCount++
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

	tasksBefore := len(tasks)
	tasksAfter := len(snapshotEvents)

	if dryRun {
		status := "dry run"
		if keepAll {
			status = "dry run (keep-all)"
		}
		return map[string]interface{}{
			"status":          status,
			"files_to_remove": filesToProcess,
			"tasks_before":    tasksBefore,
			"tasks_after":     tasksAfter,
			"pruned":          prunedCount,
		}, nil
	}

	// Write compacted file (only if there are tasks to write)
	compactedFilename := "compacted.jsonl"
	if len(snapshotEvents) > 0 {
		if err := WriteEventsToFile(root, compactedFilename, snapshotEvents); err != nil {
			return nil, fmt.Errorf("writing compacted file: %w", err)
		}
	} else {
		// Remove compacted file if no tasks remain
		_ = DeleteEventFile(root, compactedFilename)
	}

	// Delete old files
	for _, f := range filesToProcess {
		if err := DeleteEventFile(root, f); err != nil {
			return nil, fmt.Errorf("deleting %s: %w", f, err)
		}
	}

	status := "pruned"
	if keepAll {
		status = "compacted"
	}

	return map[string]interface{}{
		"status":        status,
		"files_removed": len(filesToProcess),
		"tasks_before":  tasksBefore,
		"tasks_after":   tasksAfter,
		"pruned":        prunedCount,
	}, nil
}
