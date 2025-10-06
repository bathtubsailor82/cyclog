package main

import (
	"time"

	"github.com/bathtubsailor82/cyclog"
)

func main() {
	// Create production logger
	logger := cyclog.New("example-app", "local")
	defer logger.Sync()

	// Basic logging
	logger.Info("Application started")

	// Structured logging with fields
	logger.Info("User action",
		cyclog.String("user_id", "12345"),
		cyclog.String("action", "login"),
		cyclog.Duration("response_time", 150*time.Millisecond))

	// Error logging
	logger.Error("Database connection failed",
		cyclog.String("database", "postgres"),
		cyclog.String("host", "localhost:5432"),
		cyclog.Error(nil)) // Would be real error in practice

	// Development logger for comparison
	devLogger := cyclog.NewDevelopment("example-dev")
	defer devLogger.Sync()

	devLogger.Info("This is development mode",
		cyclog.String("note", "Pretty console output"))

	logger.Info("Application finished")
}
