# CycLog

Simple Go logging package wrapping Uber Zap for structured JSON logging.

## Features

- 📄 **JSON output** to both console and file
- 🏷️ **Auto-labeling** with app name and site
- ⚡ **High performance** using Uber Zap
- 🛠️ **Simple API** for common use cases
- 🔧 **Flexible** - use with standard Zap features

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

### Advanced Usage

```go
// Use with custom Zap configuration
config := zap.NewProductionConfig()
config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
logger := cyclog.NewWithConfig(config)
```

## Output

Logs are written to:
- **Console**: Human-readable in development, JSON in production
- **File**: `/var/log/{appname}.jsonl` (JSON Lines format)

Example log entry:
```json
{
    "level": "info",
    "timestamp": "2025-01-24T15:30:45.123Z",
    "app": "myapp",
    "site": "production",
    "message": "User logged in",
    "user_id": 123,
    "session": "abc123"
}
```

## Integration with Log Aggregation

The JSON Lines output format is perfect for:
- **Grafana Loki** with Promtail
- **ELK Stack** (Elasticsearch, Logstash, Kibana)
- **Fluentd/Fluent Bit**
- Any log aggregation system

## License

MIT License