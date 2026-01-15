package tlog

import (
	"time"
)

// EventType represents the type of event
type EventType string

const (
	EventCreate EventType = "create"
	EventStatus EventType = "status"
	EventDep    EventType = "dep"
	EventBlock  EventType = "block"
	EventUpdate EventType = "update"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusOpen       TaskStatus = "open"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

// Resolution represents why a task was closed
type Resolution string

const (
	ResolutionCompleted Resolution = "completed"
	ResolutionWontfix   Resolution = "wontfix"
	ResolutionDuplicate Resolution = "duplicate"
)

// Priority represents task priority (lower number = higher priority)
type Priority int

const (
	PriorityCritical Priority = 0
	PriorityHigh     Priority = 1
	PriorityMedium   Priority = 2 // default
	PriorityLow      Priority = 3
	PriorityBacklog  Priority = 4
)

// String returns the string representation of a priority
func (p Priority) String() string {
	switch p {
	case PriorityCritical:
		return "critical"
	case PriorityHigh:
		return "high"
	case PriorityMedium:
		return "medium"
	case PriorityLow:
		return "low"
	case PriorityBacklog:
		return "backlog"
	default:
		return "medium"
	}
}

// ParsePriority converts a string to a Priority
func ParsePriority(s string) Priority {
	switch s {
	case "critical":
		return PriorityCritical
	case "high":
		return PriorityHigh
	case "medium":
		return PriorityMedium
	case "low":
		return PriorityLow
	case "backlog":
		return PriorityBacklog
	default:
		return PriorityMedium
	}
}

// Event represents a single event in the event log
type Event struct {
	ID          string     `json:"id"`
	Timestamp   time.Time  `json:"ts"`
	Type        EventType  `json:"type"`
	Title       string     `json:"title,omitempty"`
	Status      TaskStatus `json:"status,omitempty"`
	Resolution  Resolution `json:"resolution,omitempty"`
	Priority    *Priority  `json:"priority,omitempty"` // Pointer to distinguish unset from zero
	Deps        []string   `json:"deps,omitempty"`
	Blocks      []string   `json:"blocks,omitempty"`
	Labels      []string   `json:"labels,omitempty"`
	Description string     `json:"description,omitempty"` // Mutable: what is this task
	Notes       string     `json:"notes,omitempty"`       // Append-only: what happened
	// For dep/block events
	Dep    string `json:"dep,omitempty"`
	Block  string `json:"block,omitempty"`
	Action string `json:"action,omitempty"` // "add" or "remove"
}

// Task represents the computed state of a task
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Status      TaskStatus `json:"status"`
	Resolution  Resolution `json:"resolution,omitempty"`
	Priority    Priority   `json:"priority"`
	Deps        []string   `json:"deps"`
	Blocks      []string   `json:"blocks"`
	Created     time.Time  `json:"created"`
	Updated     time.Time  `json:"updated"`
	Labels      []string   `json:"labels"`
	Description string     `json:"description,omitempty"` // Mutable: what is this task
	Notes       string     `json:"notes,omitempty"`       // Append-only: what happened
}

// GraphNode represents a node in the dependency graph
type GraphNode struct {
	ID     string     `json:"id"`
	Title  string     `json:"title"`
	Status TaskStatus `json:"status"`
}

// GraphEdge represents an edge in the dependency graph
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // "depends_on" or "blocks"
}

// Graph represents the full dependency graph
type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// PrimeOutput represents the output of the prime command
type PrimeOutput struct {
	Instructions    string `json:"instructions"`
	Summary         string `json:"summary"`
	ReadyTasks      []Task `json:"ready_tasks"`
	RecentCompleted []Task `json:"recent_completed"`
	BlockedTasks    []Task `json:"blocked_tasks"`
}
