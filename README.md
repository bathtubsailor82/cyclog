# CycLog

Simple Go logging package wrapping Uber Zap for structured JSON logging.

## Features

▪ ✓ **JSON output** to console+file, file-only, or development mode
▪ ◌ **Auto-labeling** with app name, site, and PID
▪ ⟐ **PID-based files** for process isolation (`app-12345.jsonl`)
▪ ⚊ **Multi-process support** for parent/child logging
▪ ⚡ **High performance** using Uber Zap
▪ ⚒ **Simple API** for common use cases
▪ ⊙ **Flexible** - use with standard Zap features
▪ 📖 **Log reading** - API to read and parse log entries for integration

## Installation

```bash
go get github.com/bathtubsailor82/cyclog
```

## Usage

### Basic Usage

```go
package main

import (
    "github.com/bathtubsailor82/cyclog"
    "go.uber.org/zap"
)

func main() {
    // Create logger for production
    logger := cyclog.New("myapp", "production")
    defer logger.Sync()

    // Use standard Zap logging
    logger.Info("Application started",
        zap.String("version", "1.0.0"),
        zap.Int("port", 8080))

    logger.Error("Something went wrong",
        zap.Error(err),
        zap.String("operation", "database_connect"))
}
```

### Development Mode

```go
// Pretty console output for development
logger := cyclog.NewDevelopment("myapp")
logger.Info("Debug information", zap.String("user", "john"))
```

### File-Only Mode (Daemons)

```go
// JSON output ONLY to file (no console output)
// Perfect for background daemons and services
logger := cyclog.NewFileOnly("myapp", "production")
logger.Info("Daemon started") // Only in logs/myapp-12345.jsonl
```

### Reading Log Entries

```go
// Read recent log entries from latest log file
entries, err := cyclog.ReadEntries("myapp", 10) // Get last 10 entries
if err != nil {
    log.Fatal(err)
}

for _, entry := range entries {
    fmt.Printf("[%s] %s: %s\n", 
        entry.Level, 
        entry.Timestamp.Format("15:04:05"), 
        entry.Message)
}

// Read from specific PID log file
entries, err := cyclog.ReadEntriesByPID("myapp", 12345, 0) // All entries
if err != nil {
    log.Fatal(err)
}
```

### Advanced Usage

```go
// Use with custom Zap configuration
config := zap.NewProductionConfig()
config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
logger := cyclog.NewWithConfig(config)
```

## Output

Logs are written to:
▪ **Console**: Human-readable in development, JSON in production
▪ **File**: `./logs/{appname}-{pid}.jsonl` (JSON Lines format)

Example log entry:
```json
{
    "level": "info",
    "ts": 1758897151.664017,
    "caller": "main.go:16",
    "msg": "User logged in",
    "app": "myapp",
    "site": "production",
    "pid": 12345,
    "user_id": 123,
    "session": "abc123"
}
```

## Multi-Process Usage

### Parent Process
```go
logger := cyclog.New("myapp", "production")
// Creates: ./logs/myapp-12345.jsonl

// Launch child process with parent PID
cmd := exec.Command("./worker", "task1")
cmd.Env = append(os.Environ(), fmt.Sprintf("CYCLOG_PARENT_PID=%d", os.Getpid()))
cmd.Run()
```

### Child Process (Go)
```go
// Reads CYCLOG_PARENT_PID automatically
logger := cyclog.New("myapp", "production")
// Writes to same file: ./logs/myapp-12345.jsonl
logger.Info("Child process started")
```

### Child Process (Other Languages)
```swift
// Swift example - manual JSON writing
let parentPID = Int(ProcessInfo.processInfo.environment["CYCLOG_PARENT_PID"] ?? "0")!
let logFile = "./logs/myapp-\(parentPID).jsonl"

let entry = [
    "level": "info",
    "ts": Date().timeIntervalSince1970,
    "msg": "Swift worker started",
    "app": "myapp",
    "pid": ProcessInfo.processInfo.processIdentifier
]
// Append JSON to logFile
```

### API Integration Example

```go
// HTTP API endpoint to serve recent logs
func logsHandler(w http.ResponseWriter, r *http.Request) {
    appName := r.URL.Query().Get("app")
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 {
        limit = 100
    }
    
    entries, err := cyclog.ReadEntries(appName, limit)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(entries)
}
```

## Integration with Log Aggregation

The JSON Lines output format is perfect for:
▪ ⚈ **Grafana Loki** with Promtail
▪ ⚈ **ELK Stack** (Elasticsearch, Logstash, Kibana)
▪ ⚈ **Fluentd/Fluent Bit**
▪ ⚈ Any log aggregation system
▪ ⚈ **Custom APIs** with ReadEntries functions

### Promtail Configuration
```yaml
scrape_configs:
  - job_name: myapp
    static_configs:
      - targets: [localhost]
        labels:
          job: myapp
          __path__: ./logs/myapp-*.jsonl
    pipeline_stages:
      - json:
          expressions:
            level: level
            app: app
            pid: pid
      - labels:
          level:
          app:
```

### Viewing Logs in Development
```bash
# Real-time log streaming with formatting
tail -f ./logs/myapp-12345.jsonl | jq -r '"\(.ts | strftime("%H:%M:%S")) [\(.level | ascii_upcase)] \(.msg)"'

# Filter by level
tail -f ./logs/myapp-12345.jsonl | jq 'select(.level=="error")'

# Filter by PID (useful when multiple instances)
tail -f ./logs/myapp-*.jsonl | jq 'select(.pid==12345)'
```

## License

MIT License