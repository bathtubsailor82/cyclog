package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/localadmin/cyclog"
)

func main() {
	// ✨ Background app - should NOT show logs in terminal
	logger := cyclog.NewAutoDaemon("background-app")
	defer logger.Close()
	
	// If --logstreamer was used, this won't be reached
	
	// App continues running - fmt status only (no logs in terminal)
	fmt.Println("▶ Starting background processing...")
	
	// Shutdown channel for goroutines
	shutdown := make(chan bool, 2)
	
	// Background work simulation
	go backgroundWork(logger, shutdown)
	
	// Status updates via fmt (not logs)
	go statusUpdates(shutdown)
	
	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	<-c
	fmt.Println("⬛ Shutting down background app...")
	
	// Stop all goroutines
	shutdown <- true
	shutdown <- true
	
	logger.Info("Background app shutting down")
	time.Sleep(100 * time.Millisecond) // Let log flush
}

func backgroundWork(logger *cyclog.CycLogger, shutdown chan bool) {
	counter := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-shutdown:
			return
		case <-ticker.C:
			counter++
			
			// These logs should NOT appear in terminal
			// Only via --logstreamer
			logger.Info("Background task completed", 
				"task_id", counter,
				"duration_ms", 150,
				"items_processed", counter*5)
				
			logger.Debug("Task details", 
				"worker_id", "bg-worker-1",
				"memory_usage", "45MB",
				"cpu_percent", 12)
				
			if counter%10 == 0 {
				logger.Warn("Periodic maintenance needed", 
					"tasks_completed", counter,
					"next_maintenance_in", "20 tasks")
			}
		}
	}
}

func statusUpdates(shutdown chan bool) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	uptime := 0
	for {
		select {
		case <-shutdown:
			return
		case <-ticker.C:
			uptime += 10
			// Basic status via fmt (not logs)
			fmt.Printf("◐ Uptime: %ds | Status: Running\n", uptime)
		}
	}
}