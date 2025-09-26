package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/localadmin/cyclog"
)

func main() {
	var moduleName = flag.String("module", "worker", "Module name")
	var appName = flag.String("app", "demoapp", "Application name")
	flag.Parse()
	
	fmt.Printf("▶ Starting %s module for app %s\n", *moduleName, *appName)
	
	// 1. Read session to discover MainPID
	session, err := cyclog.ReadSession(*appName)
	if err != nil {
		fmt.Printf("✗ Failed to read session: %v\n", err)
		fmt.Println("⚠ Make sure main app is running first")
		os.Exit(1)
	}
	
	modulePID := os.Getpid()
	
	// 2. Create pipe using new naming convention
	pipePath := cyclog.GetPipePathNew(session.MainPID, session.AppName, *moduleName, modulePID)
	
	// 3. Create producer to send logs
	producer := cyclog.NewProducer(pipePath)
	defer producer.Close()
	
	fmt.Printf("✓ %s connected to session %d_%s (pipe: %s)\n", 
		*moduleName, session.MainPID, session.AppName, pipePath)
	
	// 4. Log module startup
	producer.Info("Module started", 
		"module", *moduleName,
		"module_pid", modulePID,
		"main_pid", session.MainPID,
		"session", session.AppName)
	
	// 5. Start module-specific work simulation
	go simulateModuleWork(producer, *moduleName, modulePID)
	
	// 6. Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	<-c
	fmt.Printf("⬛ Stopping %s module...\n", *moduleName)
	producer.Info("Module stopping", "module", *moduleName)
}

func simulateModuleWork(producer *cyclog.CycLogger, moduleName string, modulePID int) {
	counter := 0
	
	// Each module has different behavior
	interval := getModuleInterval(moduleName)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			counter++
			generateModuleActivity(producer, moduleName, modulePID, counter)
		}
	}
}

func getModuleInterval(moduleName string) time.Duration {
	intervals := map[string]time.Duration{
		"processor": 800 * time.Millisecond,  // Fast
		"validator": 1200 * time.Millisecond, // Medium  
		"indexer":   2000 * time.Millisecond, // Slow
	}
	
	if interval, exists := intervals[moduleName]; exists {
		return interval
	}
	
	return 1500 * time.Millisecond // Default
}

func generateModuleActivity(producer *cyclog.CycLogger, moduleName string, modulePID int, counter int) {
	switch moduleName {
	case "processor":
		generateProcessorLogs(producer, modulePID, counter)
	case "validator":
		generateValidatorLogs(producer, modulePID, counter)
	case "indexer":
		generateIndexerLogs(producer, modulePID, counter)
	default:
		generateGenericLogs(producer, moduleName, modulePID, counter)
	}
}

func generateProcessorLogs(producer *cyclog.CycLogger, pid int, counter int) {
	switch rand.Intn(4) {
	case 0:
		producer.Info("Processing batch", 
			"batch_id", counter,
			"items", rand.Intn(50)+10,
			"module_pid", pid)
	case 1:
		producer.Debug("Cache lookup", 
			"key", fmt.Sprintf("batch_%d", counter),
			"hit", rand.Intn(2) == 1,
			"module_pid", pid)
	case 2:
		if rand.Intn(10) == 0 {
			producer.Warn("Processing slow", 
				"batch_id", counter,
				"duration_ms", rand.Intn(2000)+1000,
				"module_pid", pid)
		} else {
			producer.Info("Batch completed", 
				"batch_id", counter,
				"duration_ms", rand.Intn(500)+100,
				"module_pid", pid)
		}
	case 3:
		if rand.Intn(20) == 0 {
			producer.Error("Processing failed", 
				"batch_id", counter,
				"error", "timeout",
				"retry_count", rand.Intn(3),
				"module_pid", pid)
		} else {
			producer.Info("Queue status", 
				"pending", rand.Intn(20),
				"processed", counter,
				"module_pid", pid)
		}
	}
}

func generateValidatorLogs(producer *cyclog.CycLogger, pid int, counter int) {
	switch rand.Intn(3) {
	case 0:
		producer.Info("Validation started", 
			"record_id", counter,
			"schema_version", "v2.1",
			"module_pid", pid)
	case 1:
		if rand.Intn(8) == 0 {
			producer.Warn("Validation warning", 
				"record_id", counter,
				"field", "email",
				"issue", "invalid_format",
				"module_pid", pid)
		} else {
			producer.Info("Record validated", 
				"record_id", counter,
				"fields_checked", rand.Intn(15)+5,
				"module_pid", pid)
		}
	case 2:
		producer.Debug("Schema cache", 
			"schemas_loaded", rand.Intn(10)+5,
			"cache_hit_rate", fmt.Sprintf("%.1f%%", rand.Float64()*100),
			"module_pid", pid)
	}
}

func generateIndexerLogs(producer *cyclog.CycLogger, pid int, counter int) {
	switch rand.Intn(3) {
	case 0:
		producer.Info("Indexing document", 
			"doc_id", counter,
			"type", []string{"pdf", "txt", "json", "xml"}[rand.Intn(4)],
			"size_kb", rand.Intn(500)+10,
			"module_pid", pid)
	case 1:
		producer.Info("Index updated", 
			"index_name", "main_index",
			"documents", counter,
			"size_mb", rand.Intn(100)+10,
			"module_pid", pid)
	case 2:
		if rand.Intn(15) == 0 {
			producer.Error("Indexing failed", 
				"doc_id", counter,
				"error", "corrupted_file",
				"module_pid", pid)
		} else {
			producer.Debug("Search performance", 
				"avg_query_time_ms", rand.Intn(50)+5,
				"total_docs", counter,
				"module_pid", pid)
		}
	}
}

func generateGenericLogs(producer *cyclog.CycLogger, moduleName string, pid int, counter int) {
	producer.Info("Module activity", 
		"module", moduleName,
		"operation_id", counter,
		"status", "running",
		"module_pid", pid)
}