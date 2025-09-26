# CycLog V2 - Multi-Module Architecture

This is the **new architecture** with automatic multi-pipe discovery and session management.

## 🎯 Key Features

- **✓ Auto-discovery** of modules via fsnotify
- **✓ Session files** for MainPID sharing  
- **✓ Multi-pipe collector** with parallel listening
- **✓ Source identification** for each module
- **✓ Real-time streaming** with buffer history

## 📁 Architecture

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

## 🚀 Usage

### 1. Start the main multi-module app:
```bash
go run examples/multi_app.go
```

**What it does:**
- Creates session file with MainPID + app name
- Starts multi-pipe collector watching `8234_demoapp_*.fifo`
- Launches 3 worker modules (processor, validator, indexer)
- Each worker creates its own pipe using the session info

### 2. View real-time logs (separate terminal):
```bash
go run examples/streamer.go --app=demoapp
```

**What you'll see:**
- Buffer history from previous runs
- Real-time logs from all modules with source identification
- Beautiful charmbracelet formatting with colors

### 3. Individual worker (for testing):
```bash
go run examples/worker_module.go --module=processor --app=demoapp
```

## 📋 File Naming Convention

### Session Files
```
/tmp/.cyclog_session_{MainPID}_{appName}
/tmp/.cyclog_session_8234_demoapp
```

### Pipe Files  
```
/tmp/{MainPID}_{appName}_{moduleName}_{modulePID}.fifo
/tmp/8234_demoapp_processor_9012.fifo
/tmp/8234_demoapp_validator_9134.fifo
```

### Buffer Files
```
/tmp/{appName}-buffer.jsonl  
/tmp/demoapp-buffer.jsonl
```

## 🔄 Process Flow

1. **Main app starts:**
   - `CreateSession(appName, mainPID)`
   - `StartMultiPipeWatcher(mainPID, appName)`

2. **Worker modules start:**
   - `ReadSession(appName)` → discovers MainPID
   - `GetPipePathNew(mainPID, appName, moduleName, modulePID)`
   - `NewProducer(pipePath)` → sends logs

3. **Collector receives:**
   - fsnotify detects new pipes automatically
   - Starts goroutine per pipe
   - Enriches logs with source info
   - Feeds central buffer + real-time listeners

4. **Streamer connects:**
   - Reads buffer history first
   - Attaches to real-time listener
   - Displays everything with beautiful formatting

## ⚙ Advanced Usage

### Custom Module Types
```bash
go run examples/worker_module.go --module=authentication --app=myapp
go run examples/worker_module.go --module=database --app=myapp
go run examples/worker_module.go --module=api --app=myapp
```

### Multiple Apps Simultaneously  
```bash
# Terminal 1: App A
go run examples/multi_app.go  # Creates session with PID 8234

# Terminal 2: App B (different PID)
go run examples/multi_app.go  # Creates session with PID 8567

# Terminal 3: Stream App A
go run examples/streamer.go --app=demoapp

# Terminal 4: Stream App B  
go run examples/streamer.go --app=demoapp
```

Each app gets isolated pipes and buffers!

## 🧹 Cleanup

Session files and pipes are automatically cleaned up when:
- Main app exits gracefully (defer cleanup)
- System reboots (/tmp cleanup)
- Process dies (orphaned pipes detected)

## 🎨 Log Format

Each log entry includes source identification:

```json
{
  "timestamp": "2025-01-24T20:30:45Z",
  "level": "INFO", 
  "message": "Processing batch",
  "fields": {
    "batch_id": 42,
    "items": 25,
    "module_pid": 9012
  },
  "source": "processor_9012"
}
```

The **source** field shows exactly which module/PID generated the log!