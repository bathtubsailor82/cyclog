package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	charmlog "github.com/charmbracelet/log"
	"github.com/localadmin/cyclog"
)

func main() {
	var logstreamer = flag.Bool("logstreamer", false, "Start real-time log console viewer")
	var workers = flag.Int("workers", 3, "Number of worker processes to start")
	flag.Parse()
	
	if *logstreamer {
		startLogStreamer()
		return
	}
	
	// Mode normal : démarrer l'app avec plusieurs workers
	startMultiWorkerApp(*workers)
}

func startMultiWorkerApp(numWorkers int) {
	fmt.Printf("📦 Multi-Worker Demo (%d workers)\n", numWorkers)
	fmt.Println("Starting collector and multiple worker processes...")
	fmt.Println("Use --logstreamer in another terminal to view logs")
	fmt.Println("───────────────────────────────────────")
	
	// 1. Créer le collector qui va recevoir les logs de tous les workers
	config := cyclog.Config{
		PipeName: "multi-worker-demo",
		Console:  false, // Pas de console directe pour le collector
		File:     "logs/multi-worker-demo.log",
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
	
	// 2. Démarrer plusieurs workers dans des goroutines
	var wg sync.WaitGroup
	stopChan := make(chan bool, numWorkers)
	
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			startWorker(pipePath, workerID, stopChan)
		}(i)
	}
	
	// 3. Attendre l'arrêt avec Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	fmt.Printf("🔄 %d workers running... Press Ctrl+C to stop\n", numWorkers)
	<-c
	fmt.Println("\n🛑 Stopping all workers...")
	
	// Arrêter tous les workers
	for i := 0; i < numWorkers; i++ {
		stopChan <- true
	}
	
	// Attendre que tous les workers se terminent proprement
	wg.Wait()
	fmt.Println("✅ All workers stopped")
}

func startWorker(pipePath string, workerID int, stopChan chan bool) {
	// Créer le producer qui envoie vers le collector
	producer := cyclog.NewProducer(pipePath)
	defer producer.Close()
	
	workerName := fmt.Sprintf("worker-%d", workerID)
	producer.Info("Worker started", 
		"worker_id", workerName, 
		"pid", os.Getpid(),
		"worker_type", getWorkerType(workerID))
	
	// Chaque worker a un comportement légèrement différent
	counter := 0
	ticker := time.NewTicker(getWorkerInterval(workerID))
	defer ticker.Stop()
	
	for {
		select {
		case <-stopChan:
			producer.Info("Worker stopping", "worker_id", workerName)
			return
		case <-ticker.C:
			counter++
			generateWorkerLog(producer, workerID, workerName, counter)
		}
	}
}

func getWorkerType(workerID int) string {
	types := []string{"processor", "converter", "validator", "indexer", "analyzer"}
	return types[(workerID-1)%len(types)]
}

func getWorkerInterval(workerID int) time.Duration {
	// Chaque worker a une fréquence différente
	intervals := []time.Duration{
		800 * time.Millisecond,  // worker-1: rapide
		1200 * time.Millisecond, // worker-2: moyen
		2000 * time.Millisecond, // worker-3: lent
		600 * time.Millisecond,  // worker-4: très rapide
		1500 * time.Millisecond, // worker-5: moyen-lent
	}
	return intervals[(workerID-1)%len(intervals)]
}

func generateWorkerLog(producer *cyclog.CycLogger, workerID int, workerName string, counter int) {
	switch rand.Intn(5) {
	case 0:
		producer.Info("Processing batch", 
			"worker_id", workerName,
			"batch_id", counter, 
			"items", rand.Intn(50)+10,
			"status", "running")
			
	case 1:
		producer.Debug("Cache operation", 
			"worker_id", workerName,
			"operation", "lookup",
			"key", fmt.Sprintf("item_%d_%d", workerID, counter), 
			"hit", rand.Intn(2) == 1)
			
	case 2:
		if rand.Intn(15) == 0 {
			producer.Warn("Performance warning", 
				"worker_id", workerName,
				"cpu_percent", rand.Intn(40)+60, 
				"memory_mb", rand.Intn(500)+200,
				"queue_size", rand.Intn(100)+50)
		} else {
			producer.Info("Performance metrics", 
				"worker_id", workerName,
				"cpu_percent", rand.Intn(50)+20, 
				"memory_mb", rand.Intn(300)+100,
				"processed", counter*rand.Intn(10)+1)
		}
		
	case 3:
		if rand.Intn(25) == 0 {
			producer.Error("Processing failed", 
				"worker_id", workerName,
				"batch_id", counter, 
				"error", "timeout",
				"retry_count", rand.Intn(3))
		} else {
			producer.Info("Batch completed", 
				"worker_id", workerName,
				"batch_id", counter, 
				"duration_ms", rand.Intn(500)+50,
				"items_processed", rand.Intn(50)+10)
		}
		
	case 4:
		actions := []string{"validate", "transform", "index", "compress", "analyze"}
		action := actions[rand.Intn(len(actions))]
		producer.Info("Worker activity", 
			"worker_id", workerName,
			"action", action,
			"target", fmt.Sprintf("dataset_%d", rand.Intn(10)+1),
			"progress", fmt.Sprintf("%d%%", rand.Intn(100)+1))
	}
}

func startLogStreamer() {
	fmt.Println("📺 Multi-Worker Log Streamer")
	fmt.Println("Connecting to multi-worker-demo collector...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("───────────────────────────────────────")
	
	// Se connecter au collector existant
	config := cyclog.Config{
		PipeName: "multi-worker-demo",
		Console:  false,
	}
	
	collector := cyclog.NewCollector(config)
	defer collector.Close()
	
	pipePath := cyclog.GetPipePath(config)
	err := collector.StartListening(pipePath)
	if err != nil {
		fmt.Printf("Error connecting to collector: %v\n", err)
		os.Exit(1)
	}
	
	// Créer le lecteur charmlog pour affichage
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Level:           charmlog.DebugLevel,
	})
	
	// Attacher le lecteur temps réel
	readerChan := collector.AttachReader()
	
	fmt.Printf("✅ Connected to pipe: %s\n", pipePath)
	fmt.Println("📚 Showing buffer history + real-time logs from all workers:")
	fmt.Println("───────────────────────────────────────")
	
	// Gérer l'arrêt avec Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	// Boucle d'affichage
	for {
		select {
		case entry := <-readerChan:
			// Afficher avec charmlog
			var fields []interface{}
			for key, value := range entry.Fields {
				fields = append(fields, key, value)
			}
			
			switch entry.Level {
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
			
		case <-c:
			fmt.Println("\n🛑 Stopping streamer...")
			return
		}
	}
}