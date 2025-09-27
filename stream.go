package cyclog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// getStringField safely extracts a string field from the parsed JSON
func getStringField(entry map[string]interface{}, field, defaultValue string) string {
	if value, ok := entry[field].(string); ok {
		return value
	}
	return defaultValue
}