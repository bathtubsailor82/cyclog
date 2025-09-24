# CycLog Examples

This directory contains clean examples demonstrating the three main CycLog patterns:

## 1. Single Worker Demo (`single_worker_demo.go`)

Demonstrates the Producer/Collector pattern with one worker process.

**Architecture:** Main app creates a collector + starts one worker that sends logs via pipe.

**Usage:**
```bash
# Terminal 1: Start the main app (collector + worker)
go run examples/single_worker_demo.go

# Terminal 2: View logs in real-time
go run examples/single_worker_demo.go --logstreamer
```

**What you'll see:**
- Worker generates various types of logs (info, debug, warn, error)
- Logs are saved to `logs/single-worker-demo.log`
- The `--logstreamer` shows buffer history + real-time logs

## 2. Multi-Worker Demo (`multi_worker_demo.go`)

Demonstrates multiple workers sending logs to one central collector.

**Architecture:** Main app creates one collector + multiple workers, each sending logs via the same pipe.

**Usage:**
```bash
# Terminal 1: Start with 3 workers (default)
go run examples/multi_worker_demo.go

# Or specify number of workers
go run examples/multi_worker_demo.go --workers=5

# Terminal 2: View logs from all workers
go run examples/multi_worker_demo.go --logstreamer
```

**What you'll see:**
- Multiple workers with different behaviors and frequencies
- Each worker has a unique ID and type (processor, converter, etc.)
- All logs centralized in one collector and one log file
- The streamer shows mixed logs from all workers in real-time

## 3. Standalone Demo (`standalone_demo.go`)

Demonstrates simple standalone logging (console + file, no pipes).

**Architecture:** Single process with direct console + file logging.

**Usage:**
```bash
go run examples/standalone_demo.go
```

**What you'll see:**
- Direct console output with colors (charmbracelet)
- Logs saved to `logs/standalone-demo.log`
- Simulates typical application activity (users, tasks, metrics, errors)
- No pipes or inter-process communication

## Key Concepts Demonstrated

### Producer/Collector Pattern
- **Producer:** Process that generates logs and sends them via named pipe
- **Collector:** Process that receives logs from pipe(s) and handles output (console, file)
- **Pipe:** Named FIFO file in `/tmp/` for inter-process communication

### Real-time Streaming
- Collectors maintain a circular buffer of recent log entries
- The `--logstreamer` connects to a collector's buffer and gets:
  1. History: All recent entries from the buffer
  2. Real-time: New entries as they arrive

### Cross-Language Support
- Pipes use JSON format, enabling any language to send logs
- Just set `CYCLOG_PIPE` environment variable and send JSON to the pipe

## Architecture Summary

```
Single Worker:    [Worker] --pipe--> [Collector] --> console + file
Multi Worker:     [Worker1] -----+
                  [Worker2] -----+-pipe--> [Collector] --> console + file  
                  [Worker3] -----+
Standalone:       [App] --> console + file (no pipes)
```

The `--logstreamer` is simply a reader that connects to the collector's buffer for real-time viewing.