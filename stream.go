package cyclog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// LogEntry represents a parsed log entry from JSON
type LogEntry struct {
	Timestamp time.Time              `json:"ts"`
	Level     string                 `json:"level"`
	Message   string                 `json:"msg"`
	App       string                 `json:"app"`
	Site      string                 `json:"site,omitempty"`
	PID       int                    `json:"pid"`
	Caller    string                 `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:"-"`
}

// ReadEntries reads log entries from the latest log file for the given app
// Returns the most recent entries up to the specified limit
func ReadEntries(appName string, limit int) ([]LogEntry, error) {
	logFile, err := findLatestLogFile(appName)
	if err != nil {
		return nil, fmt.Errorf("could not find log file for app '%s': %w", appName, err)
	}

	return readLogFile(logFile, limit)
}

// ReadEntriesByPID reads log entries from a specific PID log file
// Returns the most recent entries up to the specified limit
func ReadEntriesByPID(appName string, pid int, limit int) ([]LogEntry, error) {
	logFile := filepath.Join("logs", fmt.Sprintf("%s-%d.jsonl", appName, pid))

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("log file not found: %s", logFile)
	}

	return readLogFile(logFile, limit)
}

// Stream finds and displays logs for the given app with pretty formatting
func Stream(appName string) error {
	logFile, err := findLatestLogFile(appName)
	if err != nil {
		return fmt.Errorf("could not find log file for app '%s': %w", appName, err)
	}

	fmt.Printf("Streaming logs for %s: %s\n", appName, logFile)
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println("─────────────────────────────────────────")

	return streamLogFile(logFile)
}

// StreamByPID displays logs for a specific app and PID
func StreamByPID(appName string, pid int) error {
	logFile := filepath.Join("logs", fmt.Sprintf("%s-%d.jsonl", appName, pid))

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return fmt.Errorf("log file not found: %s", logFile)
	}

	fmt.Printf("Streaming logs for %s (PID %d): %s\n", appName, pid, logFile)
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println("─────────────────────────────────────────")

	return streamLogFile(logFile)
}

// findLatestLogFile finds the most recently modified log file for the given app
func findLatestLogFile(appName string) (string, error) {
	logsDir := "logs"

	// Check if logs directory exists
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		return "", fmt.Errorf("logs directory not found: %s", logsDir)
	}

	files, err := os.ReadDir(logsDir)
	if err != nil {
		return "", fmt.Errorf("could not read logs directory: %w", err)
	}

	var latestFile string
	var latestTime int64

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Check if file matches pattern: appname-*.jsonl
		name := file.Name()
		if len(name) < 6 || name[len(name)-6:] != ".jsonl" {
			continue
		}

		// Check if filename starts with appname-
		prefix := appName + "-"
		if len(name) <= len(prefix) || name[:len(prefix)] != prefix {
			continue
		}

		// Get file modification time
		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Unix() > latestTime {
			latestTime = info.ModTime().Unix()
			latestFile = filepath.Join(logsDir, name)
		}
	}

	if latestFile == "" {
		return "", fmt.Errorf("no log files found matching pattern '%s-*.jsonl'", appName)
	}

	return latestFile, nil
}

// streamLogFile streams the content of a log file with pretty formatting
func streamLogFile(logFile string) error {
	// Create charmbracelet logger for pretty output
	prettyLogger := log.New(os.Stdout)
	prettyLogger.SetReportTimestamp(false) // We'll format timestamps ourselves

	// Use tail -f to follow the file
	cmd := exec.Command("tail", "-f", logFile)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not create pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		// Fallback to reading existing content if tail fails
		return readExistingFile(logFile, prettyLogger)
	}

	// Process lines as they come
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			prettyPrintLogLine(line, prettyLogger)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return cmd.Wait()
}

// readExistingFile reads and displays existing log content (fallback)
func readExistingFile(logFile string, prettyLogger *log.Logger) error {
	content, err := os.ReadFile(logFile)
	if err != nil {
		return fmt.Errorf("could not read log file: %w", err)
	}

	lines := bufio.NewScanner(bufio.NewReader(bytes.NewReader(content)))
	for lines.Scan() {
		line := lines.Text()
		if line != "" {
			prettyPrintLogLine(line, prettyLogger)
		}
	}

	return nil
}

// prettyPrintLogLine parses a JSON log line and displays it with charmbracelet
func prettyPrintLogLine(jsonLine string, prettyLogger *log.Logger) {
	var entry map[string]interface{}

	if err := json.Unmarshal([]byte(jsonLine), &entry); err != nil {
		// If not valid JSON, just print the raw line
		fmt.Println(jsonLine)
		return
	}

	// Extract fields
	level := getStringField(entry, "level", "info")
	message := getStringField(entry, "msg", "")
	app := getStringField(entry, "app", "unknown")

	// Format timestamp
	timestampStr := ""
	if ts, ok := entry["ts"].(float64); ok {
		timestamp := time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e9))
		timestampStr = timestamp.Format("15:04:05")
	}

	// Build context fields (exclude standard fields)
	var contextFields []interface{}
	for key, value := range entry {
		if key != "level" && key != "msg" && key != "app" && key != "site" && key != "pid" && key != "ts" && key != "caller" {
			contextFields = append(contextFields, key, value)
		}
	}

	// Add app info to context
	contextFields = append(contextFields, "app", app)
	if pid, ok := entry["pid"].(float64); ok {
		contextFields = append(contextFields, "pid", int(pid))
	}

	// Custom timestamp prefix
	if timestampStr != "" {
		message = fmt.Sprintf("[%s] %s", timestampStr, message)
	}

	// Display with appropriate level
	switch level {
	case "debug":
		prettyLogger.Debug(message, contextFields...)
	case "info":
		prettyLogger.Info(message, contextFields...)
	case "warn", "warning":
		prettyLogger.Warn(message, contextFields...)
	case "error":
		prettyLogger.Error(message, contextFields...)
	default:
		prettyLogger.Info(message, contextFields...)
	}
}

// readLogFile reads and parses log entries from a file, returning the most recent entries
func readLogFile(logFile string, limit int) ([]LogEntry, error) {
	content, err := os.ReadFile(logFile)
	if err != nil {
		return nil, fmt.Errorf("could not read log file: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return []LogEntry{}, nil
	}

	var entries []LogEntry
	start := 0
	if limit > 0 && len(lines) > limit {
		start = len(lines) - limit
	}

	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		var rawEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &rawEntry); err != nil {
			continue // Skip invalid JSON lines
		}

		entry := LogEntry{
			Level:   getStringField(rawEntry, "level", "info"),
			Message: getStringField(rawEntry, "msg", ""),
			App:     getStringField(rawEntry, "app", "unknown"),
			Site:    getStringField(rawEntry, "site", ""),
			Caller:  getStringField(rawEntry, "caller", ""),
			Fields:  make(map[string]interface{}),
		}

		// Parse timestamp
		if ts, ok := rawEntry["ts"].(float64); ok {
			entry.Timestamp = time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e9))
		}

		// Parse PID
		if pid, ok := rawEntry["pid"].(float64); ok {
			entry.PID = int(pid)
		}

		// Extract additional fields (excluding standard ones)
		for key, value := range rawEntry {
			if key != "level" && key != "msg" && key != "app" && key != "site" && key != "pid" && key != "ts" && key != "caller" {
				entry.Fields[key] = value
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// getStringField safely extracts a string field from the parsed JSON
func getStringField(entry map[string]interface{}, field, defaultValue string) string {
	if value, ok := entry[field].(string); ok {
		return value
	}
	return defaultValue
}
