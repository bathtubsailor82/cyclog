package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/localadmin/cyclog"
)

func main() {
	fmt.Println("◉ Multi-Module App Demo")
	fmt.Println("Starting main app with multi-pipe collector...")
	fmt.Println("───────────────────────────────────────")
	
	appName := "demoapp"
	mainPID := os.Getpid()
	
	// 1. Create session file for child processes
	err := cyclog.CreateSession(appName, mainPID)
	if err != nil {
		fmt.Printf("✗ Failed to create session: %v\n", err)
		os.Exit(1)
	}
	defer cyclog.CleanupSession(mainPID, appName)
	
	fmt.Printf("✓ Session created: %s (MainPID: %d)\n", appName, mainPID)
	
	// 2. Create multi-pipe collector
	config := cyclog.Config{
		Console: false, // Collector doesn't need console (modules will log via pipes)
		File:    "logs/multi-app.log",
	}
	
	collector := cyclog.NewCollector(config)
	defer collector.Close()
	
	// 3. Start multi-pipe watcher
	err = collector.StartMultiPipeWatcher(mainPID, appName)
	if err != nil {
		fmt.Printf("✗ Failed to start multi-pipe watcher: %v\n", err)
		os.Exit(1)
	}
	defer collector.StopMultiPipeWatcher()
	
	fmt.Printf("⚡ Multi-pipe collector started, watching: %d_%s_*.fifo\n", mainPID, appName)
	
	// 4. Start the main application's own logging
	startMainAppLogging(appName, mainPID)
	
	// 5. Launch worker modules
	workers := []string{"processor", "validator", "indexer"}
	var processes []*exec.Cmd
	
	for _, workerName := range workers {
		cmd := exec.Command("go", "run", "examples/worker_module.go", "--module="+workerName, "--app="+appName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		err := cmd.Start()
		if err != nil {
			fmt.Printf("⚠ Failed to start worker %s: %v\n", workerName, err)
			continue
		}
		
		processes = append(processes, cmd)
		fmt.Printf("▶ Started worker: %s (PID: %d)\n", workerName, cmd.Process.Pid)
	}
	
	// 6. Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	fmt.Println("◉ Multi-module app running...")
	fmt.Println("◐ Use: go run examples/streamer.go --app=demoapp (to view logs)")
	fmt.Println("⚠ Press Ctrl+C to stop")
	fmt.Println("───────────────────────────────────────")
	
	<-c
	fmt.Println("\n⬛ Stopping multi-module app...")
	
	// 7. Cleanup: terminate all worker processes
	for i, cmd := range processes {
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
			// Give process time to cleanup gracefully
			go func(c *exec.Cmd, name string) {
				c.Wait()
				fmt.Printf("◯ Worker %s stopped\n", name)
			}(cmd, workers[i])
		}
	}
	
	// Wait a bit for graceful shutdown
	time.Sleep(1 * time.Second)
	fmt.Println("✓ Multi-module app stopped")
}

func startMainAppLogging(appName string, mainPID int) {
	// Main app creates its own producer to log via pipe
	pipePath := cyclog.GetPipePathNew(mainPID, appName, "main", mainPID)
	producer := cyclog.NewProducer(pipePath)
	
	// Log startup
	producer.Info("Main application started", 
		"app", appName,
		"main_pid", mainPID,
		"version", "1.0.0")
	
	// Simulate main app activity in background
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		counter := 0
		
		for {
			select {
			case <-ticker.C:
				counter++
				producer.Info("Main app heartbeat", 
					"counter", counter,
					"uptime", fmt.Sprintf("%ds", counter*5),
					"status", "running")
			}
		}
	}()
}