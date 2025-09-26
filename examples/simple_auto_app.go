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
	// ✨ Magic happens here! CycLog auto-integration
	logger := cyclog.NewAutoApp("simple-demo")
	defer logger.Close()
	
	// If user ran: ./simple_auto_app --logstreamer
	// CycLog took control and this line won't be reached
	
	fmt.Println("◉ Simple Auto App Demo")
	fmt.Println("CycLog auto-integration active!")
	fmt.Println("───────────────────────────────────────")
	fmt.Println("Available CycLog flags:")
	fmt.Println("  --logstreamer    View real-time logs")
	fmt.Println("  --log-status     Show session info")
	fmt.Println("  --log-cleanup    Clean up log files")
	fmt.Println("  --log-help       Show CycLog help")
	fmt.Println("───────────────────────────────────────")
	
	// Log startup
	logger.Info("Simple auto app started", 
		"version", "1.0.0",
		"pid", os.Getpid())
	
	// Simulate app activity
	go simulateWork(logger)
	
	// Wait for shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	fmt.Println("◉ App running... Try in another terminal:")
	fmt.Println("  go run examples/simple_auto_app.go --logstreamer")
	fmt.Println("⚠ Press Ctrl+C to stop")
	
	<-c
	fmt.Println("\n⬛ Stopping app...")
	logger.Info("Simple auto app stopping")
}

func simulateWork(logger *cyclog.CycLogger) {
	counter := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	activities := []string{
		"processing_data", "validating_input", "generating_report", 
		"updating_cache", "sending_notification", "cleanup_task",
	}
	
	for {
		select {
		case <-ticker.C:
			counter++
			activity := activities[rand.Intn(len(activities))]
			
			// Vary log types
			switch rand.Intn(6) {
			case 0:
				logger.Info("Task started", 
					"task", activity,
					"task_id", counter,
					"status", "running")
			case 1:
				logger.Info("Task completed", 
					"task", activity,
					"task_id", counter,
					"duration_ms", rand.Intn(500)+100)
			case 2:
				logger.Debug("Performance metrics", 
					"cpu_usage", rand.Intn(40)+20,
					"memory_mb", rand.Intn(100)+50,
					"active_tasks", rand.Intn(10)+1)
			case 3:
				if rand.Intn(8) == 0 {
					logger.Warn("Task running slow", 
						"task", activity,
						"task_id", counter,
						"expected_ms", 200,
						"actual_ms", rand.Intn(1000)+500)
				} else {
					logger.Info("System status", 
						"uptime_seconds", counter*2,
						"tasks_completed", counter,
						"health", "ok")
				}
			case 4:
				if rand.Intn(15) == 0 {
					logger.Error("Task failed", 
						"task", activity,
						"task_id", counter,
						"error", "timeout",
						"retry_count", rand.Intn(3))
				} else {
					logger.Info("Queue status", 
						"pending", rand.Intn(20),
						"processing", rand.Intn(5),
						"completed", counter)
				}
			case 5:
				logger.Info("Heartbeat", 
					"timestamp", time.Now().Format("15:04:05"),
					"active", true,
					"tasks_total", counter)
			}
		}
	}
}