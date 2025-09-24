# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CycLog is a Go library that provides structured logging with multiple output destinations and cross-language support:
- Console output with charmbracelet/log for beautiful terminal display
- File-based logging for persistence
- Named pipe streaming for real-time log consumption between processes
- Producer/Collector pattern for parent/child process communication

## Development Commands

### Build
```bash
go build .
```

### Run Tests
```bash
go test ./...
```

### Format Code
```bash
go fmt ./...
```

### Vet Code
```bash
go vet ./...
```

### Run Examples
```bash
cd examples
go run demo.go                    # Basic Producer/Collector demo
go run example_parent.go go-worker      # Go worker example
go run example_parent.go swift-worker   # Swift worker example  
go run example_parent.go both           # Both workers
```

### Module Management
```bash
go mod tidy    # Clean up dependencies
go mod verify  # Verify dependencies
```

## New API (v2) - Producer/Collector Pattern

### Core Components

1. **CycLogger** - Main logger that orchestrates multiple writers
2. **LogEntry** - Structured log entry with timestamp, level, message, fields, and status
3. **LogWriter Interface** - Common interface for all output destinations

### Usage Patterns

#### 1. Standalone Application (Simple)
```go
import "github.com/localadmin/cyclog"

logger := cyclog.NewStandalone(cyclog.Config{
    Console: true,
    File: "/var/log/myapp.log",
})
defer logger.Close()

logger.Info("Application started", "version", "1.0")
logger.Warn("Config missing", "path", "/etc/myapp.conf")
```

#### 2. Producer (Child Process)
```go
// Worker process that sends logs to parent
pipePath := os.Getenv("CYCLOG_PIPE") // Provided by parent
logger := cyclog.NewProducer(pipePath)
defer logger.Close()

logger.Info("Worker started", "pid", os.Getpid())
logger.Debug("Processing item", "id", 123)
logger.Info("Worker completed")
```

#### 3. Collector (Parent Process)
```go
// Parent process that collects logs from children
config := cyclog.Config{
    PipeName: "myapp",     // Creates /tmp/myapp.fifo
    Console:  true,
    File:     "/var/log/myapp.log",
}

collector := cyclog.NewCollector(config)
defer collector.Close()

pipePath := cyclog.GetPipePath(config)
collector.StartListening(pipePath)

// Launch child processes
cmd := exec.Command("./worker")
cmd.Env = append(os.Environ(), "CYCLOG_PIPE="+pipePath)
cmd.Start()

collector.Info("Parent started", "workers", 3)
```

### Cross-Language Support

Workers in any language can write to the pipe using JSON format:

#### Swift Example
```swift
let entry = [
    "timestamp": ISO8601DateFormatter().string(from: Date()),
    "level": "INFO",
    "message": "Task completed",
    "fields": ["duration": 1.2, "status": "success"],
    "status": ""
]
let jsonData = try JSONSerialization.data(withJSONObject: entry)
pipe.write(jsonData + "\n".data(using: .utf8)!)
```

#### Python Example  
```python
import json
entry = {
    "timestamp": datetime.utcnow().isoformat(),
    "level": "INFO", 
    "message": "Processing complete",
    "fields": {"items": 42, "errors": 0},
    "status": ""
}
with open(os.environ["CYCLOG_PIPE"], "w") as f:
    f.write(json.dumps(entry) + "\n")
```

## Architecture

### Producer/Collector Pattern
```
Parent (Collector) ← named pipe ← Worker 1 (Go)
       ↓                        ← Worker 2 (Swift)  
   [Console + File]             ← Worker N (any language)
```

### Key Features

- **Explicit initialization**: No more automatic init() 
- **Configurable paths**: No hardcoded `/tmp/` paths
- **Non-blocking pipes**: Graceful degradation if pipe unavailable
- **Cross-language**: JSON protocol works with any language
- **Thread-safe**: All writers use proper mutex protection
- **API preservation**: `logger.Level(message, key, value)` unchanged

## Configuration

```go
type Config struct {
    PipeName string  // Name for pipe (creates /tmp/{name}.fifo)
    PipePath string  // Explicit pipe path (overrides PipeName)  
    Console  bool    // Enable console output
    File     string  // File path for logging
}
```

## Migration from v1

**Old (deprecated):**
```go
// Auto-initialized global logger
cyclog.Info("message")  // Global function
```

**New (recommended):**
```go
// Explicit initialization
logger := cyclog.NewStandalone(config)
logger.Info("message")  // Instance method
```

## Examples

See `examples/` directory for working demonstrations:
- `demo.go` - Basic Producer/Collector usage
- `example_parent.go` - Multi-worker orchestration
- `example_worker.swift` - Cross-language worker
- `README.md` - Detailed examples documentation

## Dependencies

- `github.com/charmbracelet/log` - Beautiful terminal logging
- `gopkg.in/yaml.v3` - YAML configuration parsing  
- Standard library: `encoding/json`, `os`, `sync`, `syscall`, `time`

## Important Notes

- Use explicit constructors (`NewProducer`, `NewCollector`, `NewStandalone`) instead of global functions
- Pipe paths are configurable - no hardcoded `/tmp/` assumptions
- The library is now fully independent and can be imported in any Go application
- Cross-language workers communicate via JSON LogEntry format over named pipes
- All examples in `examples/` directory are tested and working