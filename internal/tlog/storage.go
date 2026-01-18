package tlog

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

const (
	TlogDir   = ".tlog"
	EventsDir = "events"
)

// GetTlogRoot searches up from cwd to find .tlog directory
func GetTlogRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		tlogPath := filepath.Join(dir, TlogDir)
		if info, err := os.Stat(tlogPath); err == nil && info.IsDir() {
			return tlogPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a tlog repository (or any parent)")
		}
		dir = parent
	}
}

// RequireTlog returns tlog root or exits with error
func RequireTlog() (string, error) {
	root, err := GetTlogRoot()
	if err != nil {
		return "", err
	}
	return root, nil
}

// GenerateID creates a unique task ID
func GenerateID() string {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 16)
	_, _ = rand.Read(randomBytes)

	data := fmt.Sprintf("%d%x", timestamp, randomBytes)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:8]
}

// NowISO returns current time in ISO format
func NowISO() time.Time {
	return time.Now().UTC()
}

// TodayStr returns today's date as YYYY-MM-DD
func TodayStr() string {
	return time.Now().UTC().Format("2006-01-02")
}

// AppendEvent appends an event to today's JSONL file
func AppendEvent(root string, event Event) error {
	eventsPath := filepath.Join(root, EventsDir)
	if err := os.MkdirAll(eventsPath, 0755); err != nil {
		return err
	}

	// Acquire lock to prevent concurrent write corruption
	lockPath := filepath.Join(root, "tlog.lock")
	fileLock := flock.New(lockPath)
	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer func() { _ = fileLock.Unlock() }()

	filename := filepath.Join(eventsPath, TodayStr()+".jsonl")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = f.WriteString(string(data) + "\n")
	return err
}

// LoadAllEvents loads and sorts all events chronologically
func LoadAllEvents(root string) ([]Event, error) {
	eventsPath := filepath.Join(root, EventsDir)

	entries, err := os.ReadDir(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Event{}, nil
		}
		return nil, err
	}

	var events []Event

	// Sort files by name (date order)
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jsonl" {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, filename := range files {
		filePath := filepath.Join(eventsPath, filename)
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var event Event
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				_ = f.Close()
				return nil, err
			}
			events = append(events, event)
		}
		_ = f.Close()

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	// Sort by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	return events, nil
}

// Initialize creates a new tlog repository
func Initialize(path string) error {
	tlogPath := filepath.Join(path, TlogDir)

	if _, err := os.Stat(tlogPath); err == nil {
		return fmt.Errorf("tlog already initialized")
	}

	if err := os.MkdirAll(filepath.Join(tlogPath, EventsDir), 0755); err != nil {
		return err
	}

	// Best effort: add tlog.lock to .git/info/exclude if this is a git repo
	_ = addToGitExclude(path, ".tlog/tlog.lock")

	return nil
}

// addToGitExclude adds an entry to .git/info/exclude if the git repo exists.
// Returns nil if successful or if .git doesn't exist (not an error).
func addToGitExclude(path, entry string) error {
	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		return nil // Not a git repo, nothing to do
	}

	infoPath := filepath.Join(gitPath, "info")
	if err := os.MkdirAll(infoPath, 0755); err != nil {
		return err
	}

	excludePath := filepath.Join(infoPath, "exclude")

	// Read existing content to check if entry already exists
	content, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if entry already exists
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return nil // Already excluded
		}
	}

	// Append entry
	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	// Add newline before entry if file doesn't end with one
	prefix := ""
	if len(content) > 0 && content[len(content)-1] != '\n' {
		prefix = "\n"
	}

	_, err = f.WriteString(prefix + entry + "\n")
	return err
}

// ListEventFiles returns sorted list of event file names (without path)
func ListEventFiles(root string) ([]string, error) {
	eventsPath := filepath.Join(root, EventsDir)

	entries, err := os.ReadDir(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jsonl" {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

// LoadEventsFromFile loads events from a specific file
func LoadEventsFromFile(root, filename string) ([]Event, error) {
	filePath := filepath.Join(root, EventsDir, filename)
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// WriteEventsToFile writes events to a specific file (overwrites if exists)
func WriteEventsToFile(root, filename string, events []Event) error {
	eventsPath := filepath.Join(root, EventsDir)
	if err := os.MkdirAll(eventsPath, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(eventsPath, filename)
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := f.WriteString(string(data) + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// DeleteEventFile removes an event file
func DeleteEventFile(root, filename string) error {
	filePath := filepath.Join(root, EventsDir, filename)
	return os.Remove(filePath)
}
