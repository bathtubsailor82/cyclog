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
	fmt.Println("📦 Standalone Demo")
	fmt.Println("Simple application with console + file logging (no pipes)")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("───────────────────────────────────────")
	
	// Configuration standalone : console + fichier, pas de pipe
	config := cyclog.Config{
		Console: true,                         // Affichage console direct
		File:    "logs/standalone-demo.log",   // Fichier de log
	}
	
	// Créer le logger standalone
	logger := cyclog.NewStandalone(config)
	defer logger.Close()
	
	logger.Info("Standalone application started", 
		"app_name", "demo-app",
		"pid", os.Getpid(),
		"mode", "standalone")
	
	// Démarrer la simulation d'activité
	go simulateAppActivity(logger)
	
	// Attendre l'arrêt avec Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	fmt.Println("🔄 Application running...")
	fmt.Println("💡 Logs are displayed in console and saved to logs/standalone-demo.log")
	fmt.Println("───────────────────────────────────────")
	
	<-c
	fmt.Println("\n🛑 Stopping application...")
	logger.Info("Standalone application stopping")
}

func simulateAppActivity(logger *cyclog.CycLogger) {
	tasks := []string{
		"user_authentication", "data_processing", "file_upload", 
		"database_sync", "cache_refresh", "report_generation",
		"email_notification", "backup_operation", "system_check",
	}
	
	modules := []string{
		"auth", "processor", "storage", "notifier", "scheduler",
	}
	
	counter := 0
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			counter++
			simulateLogEvent(logger, counter, tasks, modules)
		}
	}
}

func simulateLogEvent(logger *cyclog.CycLogger, counter int, tasks, modules []string) {
	switch rand.Intn(6) {
	case 0:
		// Activité normale
		task := tasks[rand.Intn(len(tasks))]
		module := modules[rand.Intn(len(modules))]
		logger.Info("Task started", 
			"task", task,
			"module", module,
			"request_id", fmt.Sprintf("req_%d", counter),
			"user_id", rand.Intn(1000)+1)
			
	case 1:
		// Succès d'opération
		task := tasks[rand.Intn(len(tasks))]
		logger.Info("Task completed successfully", 
			"task", task,
			"duration_ms", rand.Intn(500)+50,
			"request_id", fmt.Sprintf("req_%d", counter-rand.Intn(5)),
			"result", "success")
			
	case 2:
		// Métriques et monitoring
		logger.Debug("System metrics", 
			"cpu_usage", rand.Intn(40)+20,
			"memory_usage", rand.Intn(60)+30,
			"active_connections", rand.Intn(50)+10,
			"queue_size", rand.Intn(20))
			
	case 3:
		// Avertissements occasionnels
		if rand.Intn(8) == 0 {
			logger.Warn("Performance warning", 
				"metric", "response_time",
				"value", rand.Intn(1000)+500,
				"threshold", 1000,
				"module", modules[rand.Intn(len(modules))])
		} else {
			logger.Info("Health check", 
				"status", "healthy",
				"services_up", rand.Intn(10)+5,
				"last_check", time.Now().Format("15:04:05"))
		}
		
	case 4:
		// Erreurs rares
		if rand.Intn(15) == 0 {
			errors := []string{"connection_timeout", "invalid_request", "resource_not_found", "permission_denied"}
			errorType := errors[rand.Intn(len(errors))]
			logger.Error("Operation failed", 
				"error", errorType,
				"request_id", fmt.Sprintf("req_%d", counter),
				"retry_count", rand.Intn(3),
				"module", modules[rand.Intn(len(modules))])
		} else {
			logger.Info("Cache operation", 
				"operation", "refresh",
				"cache_key", fmt.Sprintf("cache_%d", rand.Intn(100)),
				"hit_rate", fmt.Sprintf("%.1f%%", rand.Float64()*100))
		}
		
	case 5:
		// Activité utilisateur
		actions := []string{"login", "logout", "view_page", "download", "upload", "search"}
		action := actions[rand.Intn(len(actions))]
		logger.Info("User activity", 
			"action", action,
			"user_id", rand.Intn(1000)+1,
			"session_id", fmt.Sprintf("sess_%d", rand.Intn(500)),
			"ip_address", fmt.Sprintf("192.168.1.%d", rand.Intn(254)+1))
	}
}