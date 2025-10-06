package cyclog

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strconv"
)

// New creates a new logger with JSON output to file and console
func New(appName, site string) *zap.Logger {
	config := zap.NewProductionConfig()

	// Determine PID for filename - use parent PID if child process
	pid := os.Getpid()
	if parentPID := os.Getenv("CYCLOG_PARENT_PID"); parentPID != "" {
		if ppid, err := strconv.Atoi(parentPID); err == nil {
			pid = ppid
		}
	}

	// Create log file with PID in filename
	// Use ./logs/ directory for development
	logDir := "./logs"
	os.MkdirAll(logDir, 0755) // Create logs directory if it doesn't exist
	logFile := filepath.Join(logDir, fmt.Sprintf("%s-%d.jsonl", appName, pid))

	// Output to console and PID-specific file
	config.OutputPaths = []string{
		"stdout",
		logFile,
	}

	// Add app, site and PID information to all logs
	config.InitialFields = map[string]interface{}{
		"app":  appName,
		"site": site,
		"pid":  os.Getpid(), // Keep actual process PID in logs
	}

	// Build the logger
	logger, err := config.Build()
	if err != nil {
		// Fallback to basic logger if config fails
		return zap.NewExample()
	}

	return logger
}

// NewFileOnly creates a logger that outputs ONLY to file (no console output)
func NewFileOnly(appName, site string) *zap.Logger {
	config := zap.NewProductionConfig()

	// Determine PID for filename - use parent PID if child process
	pid := os.Getpid()
	if parentPID := os.Getenv("CYCLOG_PARENT_PID"); parentPID != "" {
		if ppid, err := strconv.Atoi(parentPID); err == nil {
			pid = ppid
		}
	}

	// Create log file with PID in filename
	logDir := "./logs"
	os.MkdirAll(logDir, 0755) // Create logs directory if it doesn't exist
	logFile := filepath.Join(logDir, fmt.Sprintf("%s-%d.jsonl", appName, pid))

	// Output ONLY to file (no stdout)
	config.OutputPaths = []string{
		logFile,
	}

	// Add app, site and PID information to all logs
	config.InitialFields = map[string]interface{}{
		"app":  appName,
		"site": site,
		"pid":  os.Getpid(), // Keep actual process PID in logs
	}

	// Build the logger
	logger, err := config.Build()
	if err != nil {
		// Fallback to basic logger if config fails
		return zap.NewExample()
	}

	return logger
}

// NewDevelopment creates a development logger with pretty console output
func NewDevelopment(appName string) *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.InitialFields = map[string]interface{}{
		"app": appName,
		"pid": os.Getpid(),
	}

	logger, err := config.Build()
	if err != nil {
		return zap.NewExample()
	}

	return logger
}

// NewWithConfig creates a logger with custom zap configuration
func NewWithConfig(config zap.Config) *zap.Logger {
	logger, err := config.Build()
	if err != nil {
		return zap.NewExample()
	}

	return logger
}

// PrepareChildEnv prepares environment variables for child processes
// to enable shared logging with the parent process
func PrepareChildEnv(appName string) []string {
	env := os.Environ()

	// Add parent PID for shared log file
	env = append(env, fmt.Sprintf("CYCLOG_PARENT_PID=%d", os.Getpid()))

	// Add app name so child processes use the same log file name
	env = append(env, fmt.Sprintf("CYCLOG_APP_NAME=%s", appName))

	return env
}
