package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	charmlog "github.com/charmbracelet/log"
	"github.com/localadmin/cyclog"
)

func main() {
	var appName = flag.String("app", "demoapp", "Application name to stream")
	flag.Parse()
	
	fmt.Println("⚡ CycLog Real-time Streamer")
	fmt.Printf("Connecting to app: %s\n", *appName)
	fmt.Println("───────────────────────────────────────")
	
	// 1. Read session to discover MainPID
	session, err := cyclog.ReadSession(*appName)
	if err != nil {
		fmt.Printf("✗ Failed to read session: %v\n", err)
		fmt.Println("⚠ Make sure the main app is running first")
		os.Exit(1)
	}
	
	fmt.Printf("✓ Found session: %d_%s\n", session.MainPID, session.AppName)
	
	// 2. Create charmbracelet logger for beautiful output
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Level:           charmlog.DebugLevel,
	})
	
	// 3. Display buffer history first
	bufferPath := cyclog.GetBufferPathFromConfig(cyclog.Config{PipeName: session.AppName})
	displayBufferHistory(logger, bufferPath)
	
	// 4. Start real-time streaming
	fmt.Println("--- End of history, starting real-time stream ---")
	
	// Create a collector to listen to existing multi-pipe session
	collector := cyclog.NewCollector(cyclog.Config{Console: false})
	defer collector.Close()
	
	// Start multi-pipe watcher for this session
	err = collector.StartMultiPipeWatcher(session.MainPID, session.AppName)
	if err != nil {
		fmt.Printf("✗ Failed to start multi-pipe watcher: %v\n", err)
		os.Exit(1)
	}
	defer collector.StopMultiPipeWatcher()
	
	// Attach real-time reader
	readerChan := collector.AttachReader()
	
	fmt.Printf("◉ Real-time streaming active for %d_%s\n", session.MainPID, session.AppName)
	fmt.Println("⚠ Press Ctrl+C to stop")
	fmt.Println("───────────────────────────────────────")
	
	// 5. Handle shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	// 6. Display real-time logs
	for {
		select {
		case entry := <-readerChan:
			displayLogEntry(logger, entry)
			
		case <-c:
			fmt.Println("\n⬛ Stopping streamer...")
			return
		}
	}
}

func displayBufferHistory(logger *charmlog.Logger, bufferPath string) {
	data, err := os.ReadFile(bufferPath)
	if err != nil {
		fmt.Printf("◯ No buffer history found (%s)\n", bufferPath)
		return
	}
	
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("◯ Buffer is empty")
		return
	}
	
	fmt.Printf("◐ Found %d entries in buffer history:\n", len(lines))
	
	for _, line := range lines {
		if line != "" {
			var entry cyclog.LogEntry
			if json.Unmarshal([]byte(line), &entry) == nil {
				displayLogEntry(logger, entry)
			}
		}
	}
}

func displayLogEntry(logger *charmlog.Logger, entry cyclog.LogEntry) {
	// Convert fields back to slice for charmbracelet
	var fields []interface{}
	for key, value := range entry.Fields {
		fields = append(fields, key, value)
	}
	
	// Add source information if available
	if entry.Source != "" {
		fields = append(fields, "source", entry.Source)
	}
	
	// Display with appropriate level
	switch strings.ToUpper(entry.Level) {
	case "DEBUG":
		logger.Debug(entry.Message, fields...)
	case "INFO":
		logger.Info(entry.Message, fields...)
	case "WARN", "WARNING":
		logger.Warn(entry.Message, fields...)
	case "ERROR":
		logger.Error(entry.Message, fields...)
	case "FATAL":
		logger.Fatal(entry.Message, fields...)
	default:
		logger.Info(entry.Message, fields...)
	}
}