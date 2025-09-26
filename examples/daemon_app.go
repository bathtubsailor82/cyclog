package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/localadmin/cyclog"
)

func main() {
	// ✨ Daemon mode: returns terminal immediately
	logger := cyclog.NewAutoDaemon("daemon-app")
	defer logger.Close()
	
	// If user ran: ./daemon_app --logstreamer
	// CycLog took control and this line won't be reached
	
	// This runs in background - no fmt output after startup
	logger.Info("Daemon app started", 
		"version", "1.0.0",
		"pid", os.Getpid(),
		"mode", "daemon")
	
	// Start background work
	go backgroundProcessor(logger)
	go backgroundMonitor(logger)
	
	// Wait for shutdown signal (daemon keeps running)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	<-c
	logger.Info("Daemon app shutting down")
	// Clean shutdown - no fmt output
}

func backgroundProcessor(logger *cyclog.CycLogger) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	
	counter := 0
	for {
		select {
		case <-ticker.C:
			counter++
			
			// Simulate various processing tasks
			tasks := []string{"data_sync", "file_cleanup", "cache_refresh", "backup_job"}
			task := tasks[rand.Intn(len(tasks))]
			
			logger.Info("Background task started", 
				"task", task,
				"task_id", counter)
			
			// Simulate work duration
			duration := rand.Intn(2000) + 500
			time.Sleep(time.Duration(duration) * time.Millisecond)
			
			if rand.Intn(10) == 0 {
				logger.Warn("Task completed with warnings", 
					"task", task,
					"task_id", counter,
					"duration_ms", duration,
					"warnings", rand.Intn(3)+1)
			} else {
				logger.Info("Background task completed", 
					"task", task,
					"task_id", counter,
					"duration_ms", duration)
			}
		}
	}
}

func backgroundMonitor(logger *cyclog.CycLogger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// System monitoring
			logger.Debug("System health check", 
				"cpu_usage", rand.Intn(30)+20,
				"memory_usage_mb", rand.Intn(200)+100,
				"disk_free_gb", rand.Intn(50)+20,
				"active_connections", rand.Intn(10)+1)
			
			if rand.Intn(8) == 0 {
				logger.Warn("Resource warning", 
					"metric", "memory_usage",
					"value", rand.Intn(400)+600,
					"threshold", 800)
			}
			
			if rand.Intn(20) == 0 {
				logger.Error("Background service error", 
					"service", "file_watcher",
					"error", "permission_denied",
					"retry_in_seconds", 30)
			}
		}
	}
}