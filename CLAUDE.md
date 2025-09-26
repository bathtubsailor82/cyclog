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
go run multi_app.go                     # Multi-module app with session management
go run streamer.go --app=demoapp       # Real-time log streaming (separate terminal)
go run worker_module.go --module=processor --app=demoapp  # Individual worker
go run interactive_app.go              # Interactive logging demo
go run background_app.go               # Background process demo
go run daemon_app.go                   # Daemon-style logging
go run simple_auto_app.go              # Simple auto-discovery demo
```

### Module Management
```bash
go mod tidy    # Clean up dependencies
go mod verify  # Verify dependencies
```

## New API (v2) - Multi-Module Architecture

### Core Components

1. **CycLogger** - Main logger that orchestrates multiple writers
2. **LogEntry** - Structured log entry with timestamp, level, message, fields, and source
3. **LogWriter Interface** - Common interface for all output destinations
4. **Session Management** - Automatic discovery and coordination between main app and modules
5. **Multi-pipe Collector** - Parallel listening to multiple named pipes with auto-discovery

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

#### 3. Multi-Module Application (Advanced)
```go
// Main app with auto-discovery of worker modules
appName := "demoapp"
mainPID := os.Getpid()

// Create session for module coordination
cyclog.CreateSession(appName, mainPID)
defer cyclog.CleanupSession(appName, mainPID)

// Start multi-pipe collector with auto-discovery
collector := cyclog.NewCollector(cyclog.Config{
    Console: true,
    File: "/var/log/app.log",
})
defer collector.Close()

// Start watching for new modules
collector.StartMultiPipeWatcher(mainPID, appName)

// Launch worker modules - they auto-discover the session
cmd := exec.Command("./worker", "--module=processor", "--app="+appName)
cmd.Start()

collector.Info("Main app started", "session", appName)
```

#### 4. Worker Module
```go
// Worker that auto-discovers main app session
appName := "demoapp"
moduleName := "processor"
modulePID := os.Getpid()

// Read session to find MainPID
mainPID, err := cyclog.ReadSession(appName)
if err != nil {
    log.Fatal("Session not found")
}

// Create pipe path with session info
pipePath := cyclog.GetPipePathNew(mainPID, appName, moduleName, modulePID)
logger := cyclog.NewProducer(pipePath)
defer logger.Close()

logger.Info("Worker started", "module", moduleName)
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

### Multi-Module Architecture with Session Management
```
Main App (PID 8234)
├── Creates session: /tmp/.cyclog_session_8234_demoapp
├── Multi-pipe watcher: 8234_demoapp_*.fifo
└── Spawns modules:
    ├── processor (PID 9012) → 8234_demoapp_processor_9012.fifo
    ├── validator (PID 9134) → 8234_demoapp_validator_9134.fifo
    └── indexer (PID 9256) → 8234_demoapp_indexer_9256.fifo

Collector receives all logs → Central buffer → Real-time streaming
```

### File Naming Convention
- **Session files**: `/tmp/.cyclog_session_{MainPID}_{appName}`
- **Pipe files**: `/tmp/{MainPID}_{appName}_{moduleName}_{modulePID}.fifo`
- **Buffer files**: `/tmp/{appName}-buffer.jsonl`

### Key Features

- **Auto-discovery**: Modules automatically discover main app session via fsnotify
- **Session management**: MainPID sharing through session files
- **Multi-pipe collector**: Parallel listening with automatic pipe detection
- **Source identification**: Each log includes module and PID information
- **Real-time streaming**: Buffer history + live log streaming
- **Cross-language**: JSON protocol works with any language
- **Thread-safe**: All writers use proper mutex protection
- **Non-blocking pipes**: Graceful degradation if pipe unavailable

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
- `multi_app.go` - Multi-module app with session management and auto-discovery
- `worker_module.go` - Configurable worker module that auto-discovers sessions
- `streamer.go` - Real-time log streaming with buffer history
- `interactive_app.go` - Interactive logging demonstration
- `background_app.go` - Background process logging
- `daemon_app.go` - Daemon-style logging
- `simple_auto_app.go` - Simple auto-discovery demo
- `test_new_arch.go` - Architecture testing utilities
- `README.md` - Detailed V2 architecture documentation

## Dependencies

- `github.com/charmbracelet/log` - Beautiful terminal logging
- `github.com/fsnotify/fsnotify` - File system event monitoring for auto-discovery
- `gopkg.in/yaml.v3` - YAML configuration parsing
- Standard library: `encoding/json`, `os`, `sync`, `syscall`, `time`, `bufio`

## Important Notes

- Use explicit constructors (`NewProducer`, `NewCollector`, `NewStandalone`) instead of global functions
- Session management enables automatic module discovery and coordination
- Multi-pipe architecture allows parallel processing with source identification
- Real-time streaming provides both buffer history and live log feeds
- Cross-language workers communicate via JSON LogEntry format over named pipes
- File naming convention ensures isolation between different application instances
- All examples in `examples/` directory demonstrate the V2 architecture