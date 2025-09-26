package main

import (
	"time"

	"github.com/bathtubsailor82/cyclog"
	"go.uber.org/zap"
)

func main() {
	// Create production logger
	logger := cyclog.New("example-app", "local")
	defer logger.Sync()

	// Basic logging
	logger.Info("Application started")

	// Structured logging with fields
	logger.Info("User action",
		zap.String("user_id", "12345"),
		zap.String("action", "login"),
		zap.Duration("response_time", 150*time.Millisecond))

	// Error logging
	logger.Error("Database connection failed",
		zap.String("database", "postgres"),
		zap.String("host", "localhost:5432"),
		zap.Error(nil)) // Would be real error in practice

	// Development logger for comparison
	devLogger := cyclog.NewDevelopment("example-dev")
	defer devLogger.Sync()

	devLogger.Info("This is development mode",
		zap.String("note", "Pretty console output"))

	logger.Info("Application finished")
}