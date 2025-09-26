package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/localadmin/cyclog"
)

func main() {
	// ✨ Interactive mode: keeps terminal, shows progress
	logger := cyclog.NewAutoApp("interactive-app") // Default is InteractiveMode
	defer logger.Close()
	
	// If user ran: ./interactive_app --logstreamer
	// CycLog took control and this line won't be reached
	
	fmt.Println("◉ Interactive App Demo")
	fmt.Println("This app shows basic progress in terminal")
	fmt.Println("Detailed logs available via: go run . --logstreamer")
	fmt.Println("───────────────────────────────────────")
	
	logger.Info("Interactive app started", 
		"version", "1.0.0",
		"pid", os.Getpid(),
		"mode", "interactive")
	
	// Show basic progress in terminal (fmt)
	// Detailed logs go to CycLog
	fmt.Println("▶ Starting data processing...")
	
	// Start background logging
	go detailedLogging(logger)
	
	// Show progress in terminal
	go showProgress()
	
	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	fmt.Println("⚠ Press Ctrl+C to stop")
	<-c
	
	fmt.Println("\n⬛ Stopping interactive app...")
	logger.Info("Interactive app shutting down")
	fmt.Println("✓ App stopped")
}

func showProgress() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	tasks := []string{
		"Validating input data",
		"Processing batch 1/5",
		"Processing batch 2/5", 
		"Processing batch 3/5",
		"Processing batch 4/5",
		"Processing batch 5/5",
		"Generating reports",
		"Cleanup and finalization",
	}
	
	for i, task := range tasks {
		select {
		case <-ticker.C:
			progress := int(float64(i+1) / float64(len(tasks)) * 100)
			fmt.Printf("◐ %s... (%d%%)\n", task, progress)
			
			if i == len(tasks)-1 {
				fmt.Println("✓ All tasks completed")
				return
			}
		}
	}
}

func detailedLogging(logger *cyclog.CycLogger) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	counter := 0
	
	for {
		select {
		case <-ticker.C:
			counter++
			
			// Detailed logging that doesn't clutter terminal
			switch rand.Intn(5) {
			case 0:
				logger.Debug("Processing record", 
					"record_id", counter,
					"fields", rand.Intn(20)+5,
					"validation_time_ms", rand.Intn(50)+10)
					
			case 1:
				logger.Info("Batch processing status", 
					"records_processed", counter,
					"records_per_second", rand.Intn(100)+50,
					"memory_usage_mb", rand.Intn(100)+50)
					
			case 2:
				logger.Info("Database operation", 
					"operation", "insert",
					"table", "processed_data",
					"rows_affected", rand.Intn(10)+1,
					"query_time_ms", rand.Intn(100)+20)
					
			case 3:
				if rand.Intn(8) == 0 {
					logger.Warn("Performance degradation", 
						"component", "data_processor",
						"current_speed", rand.Intn(50)+20,
						"expected_speed", 100,
						"cpu_usage", rand.Intn(40)+60)
				} else {
					logger.Info("System metrics", 
						"active_threads", rand.Intn(10)+5,
						"queue_size", rand.Intn(20),
						"processed_total", counter)
				}
				
			case 4:
				if rand.Intn(15) == 0 {
					logger.Error("Processing error", 
						"record_id", counter,
						"error", "data_format_invalid",
						"field", "timestamp",
						"retry_count", rand.Intn(3))
				} else {
					logger.Debug("Cache operations", 
						"cache_hits", rand.Intn(100)+50,
						"cache_misses", rand.Intn(20)+5,
						"hit_rate_percent", rand.Intn(30)+70)
				}
			}
		}
	}
}