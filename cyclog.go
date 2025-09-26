package cyclog

import (
	"os"
	"go.uber.org/zap"
	"path/filepath"
)

// New creates a new logger with JSON output to file and console
func New(appName, site string) *zap.Logger {
	config := zap.NewProductionConfig()

	// Output to console and file
	config.OutputPaths = []string{
		"stdout",
		filepath.Join("/var/log", appName+".jsonl"),
	}

	// Add app, site and PID information to all logs
	config.InitialFields = map[string]interface{}{
		"app":  appName,
		"site": site,
		"pid":  os.Getpid(),
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