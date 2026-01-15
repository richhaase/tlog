package tlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()

	if !strings.HasPrefix(id1, "tl-") {
		t.Errorf("ID should start with 'tl-', got %s", id1)
	}
	if len(id1) != 11 { // "tl-" + 8 hex chars
		t.Errorf("ID should be 11 chars, got %d", len(id1))
	}
	if id1 == id2 {
		t.Error("IDs should be unique")
	}
}

func TestComputeState(t *testing.T) {
	now := time.Now().UTC()

	events := []Event{
		{
			ID:        "tl-001",
			Timestamp: now,
			Type:      EventCreate,
			Title:     "Task 1",
			Status:    StatusOpen,
			Deps:      []string{},
			Blocks:    []string{},
		},
		{
			ID:        "tl-002",
			Timestamp: now.Add(time.Second),
			Type:      EventCreate,
			Title:     "Task 2",
			Status:    StatusOpen,
			Deps:      []string{"tl-001"},
			Blocks:    []string{},
		},
		{
			ID:        "tl-001",
			Timestamp: now.Add(2 * time.Second),
			Type:      EventStatus,
			Status:    StatusDone,
		},
	}

	tasks := ComputeState(events)

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	task1 := tasks["tl-001"]
	if task1.Status != StatusDone {
		t.Errorf("Task 1 should be done, got %s", task1.Status)
	}

	task2 := tasks["tl-002"]
	if len(task2.Deps) != 1 || task2.Deps[0] != "tl-001" {
		t.Errorf("Task 2 should depend on tl-001")
	}
}

func TestGetReadyTasks(t *testing.T) {
	now := time.Now().UTC()

	events := []Event{
		{
			ID:        "tl-001",
			Timestamp: now,
			Type:      EventCreate,
			Title:     "Task 1",
			Status:    StatusOpen,
			Deps:      []string{},
			Blocks:    []string{},
		},
		{
			ID:        "tl-002",
			Timestamp: now.Add(time.Second),
			Type:      EventCreate,
			Title:     "Task 2",
			Status:    StatusOpen,
			Deps:      []string{"tl-001"},
			Blocks:    []string{},
		},
	}

	tasks := ComputeState(events)
	ready := GetReadyTasks(tasks)

	if len(ready) != 1 {
		t.Errorf("Expected 1 ready task, got %d", len(ready))
	}
	if ready[0].ID != "tl-001" {
		t.Errorf("Ready task should be tl-001, got %s", ready[0].ID)
	}

	// Mark task 1 as done
	events = append(events, Event{
		ID:        "tl-001",
		Timestamp: now.Add(2 * time.Second),
		Type:      EventStatus,
		Status:    StatusDone,
	})

	tasks = ComputeState(events)
	ready = GetReadyTasks(tasks)

	if len(ready) != 1 {
		t.Errorf("Expected 1 ready task after completing dep, got %d", len(ready))
	}
	if ready[0].ID != "tl-002" {
		t.Errorf("Ready task should be tl-002, got %s", ready[0].ID)
	}
}

func TestInitializeAndStorage(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tlog-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test initialize
	err = Initialize(tmpDir)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Check .tlog directory exists
	tlogPath := filepath.Join(tmpDir, TlogDir)
	if _, err := os.Stat(tlogPath); os.IsNotExist(err) {
		t.Error(".tlog directory should exist")
	}

	// Test append and load events
	event := Event{
		ID:        "tl-test",
		Timestamp: NowISO(),
		Type:      EventCreate,
		Title:     "Test task",
		Status:    StatusOpen,
		Deps:      []string{},
		Blocks:    []string{},
	}

	err = AppendEvent(tlogPath, event)
	if err != nil {
		t.Fatalf("AppendEvent failed: %v", err)
	}

	events, err := LoadAllEvents(tlogPath)
	if err != nil {
		t.Fatalf("LoadAllEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
	if events[0].ID != "tl-test" {
		t.Errorf("Event ID should be tl-test, got %s", events[0].ID)
	}
}

func TestDepEvents(t *testing.T) {
	now := time.Now().UTC()

	events := []Event{
		{
			ID:        "tl-001",
			Timestamp: now,
			Type:      EventCreate,
			Title:     "Task 1",
			Status:    StatusOpen,
			Deps:      []string{},
			Blocks:    []string{},
		},
		{
			ID:        "tl-002",
			Timestamp: now.Add(time.Second),
			Type:      EventCreate,
			Title:     "Task 2",
			Status:    StatusOpen,
			Deps:      []string{},
			Blocks:    []string{},
		},
		{
			ID:        "tl-002",
			Timestamp: now.Add(2 * time.Second),
			Type:      EventDep,
			Dep:       "tl-001",
			Action:    "add",
		},
	}

	tasks := ComputeState(events)
	task2 := tasks["tl-002"]

	if len(task2.Deps) != 1 || task2.Deps[0] != "tl-001" {
		t.Errorf("Task 2 should have tl-001 as dependency")
	}

	// Remove dependency
	events = append(events, Event{
		ID:        "tl-002",
		Timestamp: now.Add(3 * time.Second),
		Type:      EventDep,
		Dep:       "tl-001",
		Action:    "remove",
	})

	tasks = ComputeState(events)
	task2 = tasks["tl-002"]

	if len(task2.Deps) != 0 {
		t.Errorf("Task 2 should have no dependencies after removal")
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	now := time.Now().UTC()

	events := []Event{
		{
			ID:        "tl-001",
			Timestamp: now,
			Type:      EventCreate,
			Title:     "Task 1",
			Status:    StatusOpen,
			Deps:      []string{},
			Blocks:    []string{"tl-002"},
		},
		{
			ID:        "tl-002",
			Timestamp: now.Add(time.Second),
			Type:      EventCreate,
			Title:     "Task 2",
			Status:    StatusOpen,
			Deps:      []string{"tl-001"},
			Blocks:    []string{},
		},
	}

	tasks := ComputeState(events)
	graph := BuildDependencyGraph(tasks)

	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
	}

	// Should have dep edge and block edge
	if len(graph.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(graph.Edges))
	}
}
