package cyclog

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	// Clean up test logs before and after
	cleanupTestLogs()
	code := m.Run()
	cleanupTestLogs()
	os.Exit(code)
}

func cleanupTestLogs() {
	os.RemoveAll("./logs")
}

func TestNew(t *testing.T) {
	cleanupTestLogs()

	logger := New("test-app", "test-site")
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}

	// Test logging
	logger.Info("test message", zap.String("key", "value"))
	logger.Sync()

	// Check if log file was created
	expectedFile := fmt.Sprintf("./logs/test-app-%d.jsonl", os.Getpid())
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("Log file not created: %s", expectedFile)
	}

	// Verify log content
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 {
		t.Fatal("No log entries found")
	}

	if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	// Verify required fields
	if logEntry["app"] != "test-app" {
		t.Errorf("Expected app=test-app, got %v", logEntry["app"])
	}
	if logEntry["site"] != "test-site" {
		t.Errorf("Expected site=test-site, got %v", logEntry["site"])
	}
	if logEntry["msg"] != "test message" {
		t.Errorf("Expected msg=test message, got %v", logEntry["msg"])
	}
	if logEntry["key"] != "value" {
		t.Errorf("Expected key=value, got %v", logEntry["key"])
	}
}

func TestNewFileOnly(t *testing.T) {
	cleanupTestLogs()

	logger := NewFileOnly("test-fileonly", "test-site")
	if logger == nil {
		t.Fatal("NewFileOnly() returned nil logger")
	}

	logger.Info("file only message")
	logger.Sync()

	// Check if log file was created
	expectedFile := fmt.Sprintf("./logs/test-fileonly-%d.jsonl", os.Getpid())
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("Log file not created: %s", expectedFile)
	}

	// Verify log content
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("Log file is empty")
	}

	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	if logEntry["app"] != "test-fileonly" {
		t.Errorf("Expected app=test-fileonly, got %v", logEntry["app"])
	}
}

func TestNewDevelopment(t *testing.T) {
	logger := NewDevelopment("test-dev")
	if logger == nil {
		t.Fatal("NewDevelopment() returned nil logger")
	}

	// Just verify it doesn't crash
	logger.Info("development message", zap.String("env", "dev"))
	logger.Sync()
}

func TestNewWithConfig(t *testing.T) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	logger := NewWithConfig(config)
	if logger == nil {
		t.Fatal("NewWithConfig() returned nil logger")
	}

	// Just verify it doesn't crash
	logger.Debug("debug message")
	logger.Sync()
}

func TestParentPIDHandling(t *testing.T) {
	cleanupTestLogs()

	// Set parent PID environment variable
	parentPID := 12345
	os.Setenv("CYCLOG_PARENT_PID", fmt.Sprintf("%d", parentPID))
	defer os.Unsetenv("CYCLOG_PARENT_PID")

	logger := New("test-parent", "test-site")
	logger.Info("parent pid test")
	logger.Sync()

	// Should use parent PID in filename, not current PID
	expectedFile := fmt.Sprintf("./logs/test-parent-%d.jsonl", parentPID)
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("Log file with parent PID not created: %s", expectedFile)
	}

	// Verify actual PID is still in log content
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	// PID in log should be current process, not parent
	if int(logEntry["pid"].(float64)) != os.Getpid() {
		t.Errorf("Expected pid=%d, got %v", os.Getpid(), logEntry["pid"])
	}
}

func TestLogDirectoryCreation(t *testing.T) {
	cleanupTestLogs()

	// Verify logs directory doesn't exist
	if _, err := os.Stat("./logs"); !os.IsNotExist(err) {
		t.Fatal("logs directory should not exist initially")
	}

	logger := New("test-mkdir", "test-site")
	logger.Info("test directory creation")
	logger.Sync()

	// Verify logs directory was created
	if _, err := os.Stat("./logs"); os.IsNotExist(err) {
		t.Fatal("logs directory was not created")
	}
}

func TestMultipleLogEntries(t *testing.T) {
	cleanupTestLogs()

	logger := New("test-multi", "test-site")

	// Write multiple log entries
	logger.Info("first message")
	logger.Warn("warning message")
	logger.Error("error message")
	logger.Sync()

	expectedFile := fmt.Sprintf("./logs/test-multi-%d.jsonl", os.Getpid())
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 log entries, got %d", len(lines))
	}

	// Verify each entry
	var entry1, entry2, entry3 map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &entry1)
	json.Unmarshal([]byte(lines[1]), &entry2)
	json.Unmarshal([]byte(lines[2]), &entry3)

	if entry1["level"] != "info" || entry1["msg"] != "first message" {
		t.Error("First entry incorrect")
	}
	if entry2["level"] != "warn" || entry2["msg"] != "warning message" {
		t.Error("Second entry incorrect")
	}
	if entry3["level"] != "error" || entry3["msg"] != "error message" {
		t.Error("Third entry incorrect")
	}
}

func TestInvalidParentPID(t *testing.T) {
	cleanupTestLogs()

	// Set invalid parent PID
	os.Setenv("CYCLOG_PARENT_PID", "invalid")
	defer os.Unsetenv("CYCLOG_PARENT_PID")

	logger := New("test-invalid-pid", "test-site")
	logger.Info("invalid parent pid test")
	logger.Sync()

	// Should fall back to current PID
	expectedFile := fmt.Sprintf("./logs/test-invalid-pid-%d.jsonl", os.Getpid())
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("Log file with current PID not created: %s", expectedFile)
	}
}

func TestReadEntries(t *testing.T) {
	cleanupTestLogs()

	// Create test logs
	logger := New("test-read", "test-site")
	logger.Info("first entry")
	logger.Warn("second entry")
	logger.Error("third entry")
	logger.Sync()

	// Test reading all entries
	entries, err := ReadEntries("test-read", 0)
	if err != nil {
		t.Fatalf("ReadEntries failed: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}

	// Verify entry contents
	if entries[0].Message != "first entry" || entries[0].Level != "info" {
		t.Errorf("First entry incorrect: %+v", entries[0])
	}
	if entries[1].Message != "second entry" || entries[1].Level != "warn" {
		t.Errorf("Second entry incorrect: %+v", entries[1])
	}
	if entries[2].Message != "third entry" || entries[2].Level != "error" {
		t.Errorf("Third entry incorrect: %+v", entries[2])
	}

	// Test with limit
	limitedEntries, err := ReadEntries("test-read", 2)
	if err != nil {
		t.Fatalf("ReadEntries with limit failed: %v", err)
	}

	if len(limitedEntries) != 2 {
		t.Fatalf("Expected 2 entries with limit, got %d", len(limitedEntries))
	}

	// Should get the last 2 entries
	if limitedEntries[0].Message != "second entry" {
		t.Errorf("Limited entries should start with second entry, got: %s", limitedEntries[0].Message)
	}
}

func TestReadEntriesByPID(t *testing.T) {
	cleanupTestLogs()

	currentPID := os.Getpid()

	// Create test logs
	logger := New("test-read-pid", "test-site")
	logger.Info("pid specific entry")
	logger.Sync()

	// Test reading by PID
	entries, err := ReadEntriesByPID("test-read-pid", currentPID, 0)
	if err != nil {
		t.Fatalf("ReadEntriesByPID failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Message != "pid specific entry" {
		t.Errorf("Entry message incorrect: %s", entries[0].Message)
	}
	if entries[0].PID != currentPID {
		t.Errorf("Entry PID incorrect: %d, expected %d", entries[0].PID, currentPID)
	}

	// Test with non-existent PID
	_, err = ReadEntriesByPID("test-read-pid", 99999, 0)
	if err == nil {
		t.Error("Expected error for non-existent PID, got nil")
	}
}

func TestReadEntriesNonExistentApp(t *testing.T) {
	cleanupTestLogs()

	// Test reading from non-existent app
	_, err := ReadEntries("non-existent-app", 0)
	if err == nil {
		t.Error("Expected error for non-existent app, got nil")
	}
}

func TestReadEntriesEmptyFile(t *testing.T) {
	cleanupTestLogs()

	// Create empty log file
	os.MkdirAll("./logs", 0755)
	emptyFile := fmt.Sprintf("./logs/test-empty-%d.jsonl", os.Getpid())
	os.WriteFile(emptyFile, []byte(""), 0644)

	entries, err := ReadEntries("test-empty", 0)
	if err != nil {
		t.Fatalf("ReadEntries failed on empty file: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries from empty file, got %d", len(entries))
	}
}
