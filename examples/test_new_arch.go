package main

import (
	"fmt"
	"time"

	"github.com/localadmin/cyclog"
)

func main() {
	// ✨ One line magic - CycLog handles everything!
	logger := cyclog.NewAutoApp("test-app")
	defer logger.Close()
	
	// If --logstreamer was used, this won't be reached
	fmt.Println("◉ Test App - New Architecture")
	fmt.Println("CycLog auto-configured!")
	fmt.Println("Try: go run . --logstreamer (in another terminal)")
	fmt.Println("───────────────────────────────────────")
	
	// Simple logging test
	logger.Info("Application started", "version", "1.0")
	logger.Debug("Debug message test", "counter", 1)
	logger.Warn("Warning message test", "level", "test")
	
	// Wait a bit for logs to be processed
	time.Sleep(2 * time.Second)
	
	logger.Info("Application stopping")
	fmt.Println("✓ Test completed!")
}