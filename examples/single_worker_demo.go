package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	charmlog "github.com/charmbracelet/log"
	"github.com/localadmin/cyclog"
)

func main() {
	var logstreamer = flag.Bool("logstreamer", false, "Start real-time log console viewer")
	flag.Parse()
	
	if *logstreamer {
		startLogStreamer()
		return
	}
	
	// Mode normal : démarrer l'app avec worker
	startSingleWorkerApp()
}

func startSingleWorkerApp() {
	fmt.Println("📦 Single Worker Demo")
	fmt.Println("Starting collector and worker process...")
	fmt.Println("Use --logstreamer in another terminal to view logs")
	fmt.Println("───────────────────────────────────────")
	
	// 1. Créer le collector qui va recevoir les logs
	config := cyclog.Config{
		PipeName: "single-worker-demo",
		Console:  false, // Pas de console directe pour le collector
		File:     "logs/single-worker-demo.log",
	}
	
	collector := cyclog.NewCollector(config)
	defer collector.Close()
	
	pipePath := cyclog.GetPipePath(config)
	err := collector.StartListening(pipePath)
	if err != nil {
		fmt.Printf("Error starting collector: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Collector started, pipe: %s\n", pipePath)
	
	// 2. Démarrer le worker dans un goroutine
	go startWorker(pipePath)
	
	// 3. Attendre l'arrêt avec Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	fmt.Println("🔄 Worker running... Press Ctrl+C to stop")
	<-c
	fmt.Println("\n🛑 Stopping demo...")
}

func startWorker(pipePath string) {
	// Créer le producer qui envoie vers le collector
	producer := cyclog.NewProducer(pipePath)
	defer producer.Close()
	
	producer.Info("Worker started", "worker_id", "worker-1", "pid", os.Getpid())
	
	// Générer des logs en continu
	counter := 0
	for {
		counter++
		
		switch rand.Intn(4) {
		case 0:
			producer.Info("Processing task", "task_id", counter, "status", "running", "worker", "worker-1")
		case 1:
			producer.Debug("Cache operation", "key", fmt.Sprintf("item_%d", counter), "hit", rand.Intn(2) == 1, "worker", "worker-1")
		case 2:
			if rand.Intn(10) == 0 {
				producer.Warn("Performance warning", "cpu", rand.Intn(40)+60, "memory_mb", rand.Intn(500)+200, "worker", "worker-1")
			} else {
				producer.Info("Metrics", "cpu", rand.Intn(50)+20, "memory", rand.Intn(300)+100, "worker", "worker-1")
			}
		case 3:
			if rand.Intn(20) == 0 {
				producer.Error("Task failed", "task_id", counter, "error", "timeout", "retries", rand.Intn(3), "worker", "worker-1")
			} else {
				producer.Info("Task completed", "task_id", counter, "duration_ms", rand.Intn(500)+50, "worker", "worker-1")
			}
		}
		
		time.Sleep(time.Duration(rand.Intn(2000)+500) * time.Millisecond)
	}
}

func startLogStreamer() {
	fmt.Println("📺 Log Streamer - Real-time Console")
	fmt.Println("Reading single-worker-demo buffer...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("───────────────────────────────────────")
	
	// Créer le lecteur charmlog pour affichage
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Level:           charmlog.DebugLevel,
	})
	
	// Obtenir le chemin du buffer pour cette application
	bufferPath := cyclog.GetBufferPath("single-worker-demo")
	pipePath := cyclog.GetPipePath(cyclog.Config{PipeName: "single-worker-demo"})
	
	fmt.Printf("✅ Buffer: %s\n", bufferPath)
	fmt.Printf("✅ Pipe: %s\n", pipePath)
	fmt.Println("📚 Showing buffer history + real-time logs:")
	fmt.Println("───────────────────────────────────────")
	
	// 1. D'abord afficher l'historique depuis le fichier buffer
	displayBufferHistory(logger, bufferPath)
	
	// 2. Ensuite écouter le pipe pour les nouveaux logs
	go listenToPipeForStreamer(logger, pipePath)
	
	// Gérer l'arrêt avec Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	<-c
	fmt.Println("\n🛑 Stopping streamer...")
}

func displayBufferHistory(logger *charmlog.Logger, bufferPath string) {
	data, err := os.ReadFile(bufferPath)
	if err != nil {
		fmt.Printf("📭 No buffer history found (file: %s)\n", bufferPath)
		return
	}
	
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("📭 Buffer is empty")
		return
	}
	
	fmt.Printf("📚 Found %d entries in buffer history:\n", len(lines))
	
	for _, line := range lines {
		if line != "" {
			var entry cyclog.LogEntry
			if json.Unmarshal([]byte(line), &entry) == nil {
				displayLogEntry(logger, entry)
			}
		}
	}
	
	fmt.Println("--- End of history, waiting for real-time logs ---")
}

func listenToPipeForStreamer(logger *charmlog.Logger, pipePath string) {
	for {
		// Patient connection - wait for pipe to become available
		pipe, err := os.OpenFile(pipePath, os.O_RDONLY, 0)
		if err != nil {
			// Pipe not available yet, wait patiently
			time.Sleep(1 * time.Second)
			continue
		}
		
		// Create JSON decoder for pipe
		decoder := json.NewDecoder(pipe)
		
		// Listen for log entries (blocking read)
		for {
			var entry cyclog.LogEntry
			if err := decoder.Decode(&entry); err != nil {
				// Pipe closed - writer disconnected, wait for reconnection
				pipe.Close()
				break // Try to reconnect
			}
			
			// Display new entry immediately
			displayLogEntry(logger, entry)
		}
		
		// Brief pause before reconnection attempt
		time.Sleep(100 * time.Millisecond)
	}
}

func displayLogEntry(logger *charmlog.Logger, entry cyclog.LogEntry) {
	// Convert fields back to slice
	var fields []interface{}
	for key, value := range entry.Fields {
		fields = append(fields, key, value)
	}
	
	// Render with charmbracelet based on level
	switch strings.ToUpper(entry.Level) {
	case "DEBUG":
		logger.Debug(entry.Message, fields...)
	case "INFO":
		logger.Info(entry.Message, fields...)
	case "WARN", "WARNING":
		logger.Warn(entry.Message, fields...)
	case "ERROR":
		logger.Error(entry.Message, fields...)
	default:
		logger.Info(entry.Message, fields...)
	}
}