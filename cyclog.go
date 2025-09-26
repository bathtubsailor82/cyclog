package cyclog

import (
	"fmt"
	"os"
	"strconv"
	"go.uber.org/zap"
	"path/filepath"
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