package cyclog

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	charmlog "github.com/charmbracelet/log"
	"gopkg.in/yaml.v3"
)

// LogEntry represents a structured log entry for streaming
type LogEntry struct {
	Timestamp time.Time                 `json:"timestamp"`
	Level     string                    `json:"level"`  
	Message   string                    `json:"message"`
	Fields    map[string]interface{}    `json:"fields,omitempty"`
	Status    string                    `json:"status,omitempty"`  // Pour communication parent/enfant
	Source    string                    `json:"source,omitempty"`  // Identifie l'expéditeur (app_pid)
}

// LogWriter interface for different output destinations
type LogWriter interface {
	Write(entry LogEntry) error
	Close() error
}

// CycLogger wraps charmbracelet/log with multiple writers
type CycLogger struct {
	charmLogger *charmlog.Logger
	writers     []LogWriter
	buffer      *CircularBuffer  // Buffer centralisé pour historique
	listeners   []chan LogEntry  // Lecteurs temps réel connectés
	mutex       sync.RWMutex
}

// ConsoleWriter outputs to console using charmbracelet/log
type ConsoleWriter struct {
	logger *charmlog.Logger
}

// FileWriter outputs clean text to log files
type FileWriter struct {
	file *os.File
}

// CircularBuffer represents a memory-based circular buffer
type CircularBuffer struct {
	entries    []LogEntry
	maxEntries int
	index      int
	count      int
	mutex      sync.RWMutex
}

// StreamWriter writes to both circular buffer and named pipe for hybrid persistence
type StreamWriter struct {
	bufferPath   string
	pipePath     string
	memBuffer    *CircularBuffer
	pipe         *os.File
	pipeEncoder  *json.Encoder
	maxEntries   int
	mutex        sync.Mutex
	cleanupOnce  sync.Once
	initialized  bool
}

// LoggingConfig represents the logging configuration structure
type LoggingConfig struct {
	Logging struct {
		Level   string `yaml:"level"`
		Console struct {
			Enabled   bool `yaml:"enabled"`
			Colors    bool `yaml:"colors"`
			Timestamp bool `yaml:"timestamp"`
			Caller    bool `yaml:"caller"`
		} `yaml:"console"`
		File struct {
			Enabled   bool   `yaml:"enabled"`
			Path      string `yaml:"path"`
			Colors    bool   `yaml:"colors"`
			Timestamp bool   `yaml:"timestamp"`
			Caller    bool   `yaml:"caller"`
			MaxSizeMB int    `yaml:"max_size_mb"`
			MaxFiles  int    `yaml:"max_files"`
			Compress  bool   `yaml:"compress"`
		} `yaml:"file"`
		Format struct {
			TimeFormat  string `yaml:"time_format"`
			LevelFormat string `yaml:"level_format"`
		} `yaml:"format"`
		Modules map[string]string `yaml:"modules"`
	} `yaml:"logging"`
}


// === CONSOLE WRITER ===

func NewConsoleWriter(enabled bool, colors bool, caller bool) *ConsoleWriter {
	if !enabled {
		return nil
	}
	
	// Use NewWithOptions for proper configuration
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		ReportTimestamp: true,
		ReportCaller:    caller,
		TimeFormat:      "2006/01/02 15:04:05", // Consistent with file format
		Level:           charmlog.InfoLevel,
	})
	
	// Configure caller information after creation
	if caller {
		logger.SetReportCaller(true)
	}
	
	// Let charmbracelet handle colors automatically - don't force them!
	// charmbracelet auto-detects TTY and applies colors appropriately
	
	return &ConsoleWriter{logger: logger}
}

func (w *ConsoleWriter) Write(entry LogEntry) error {
	// Convert fields to slice format for charmbracelet
	var fields []interface{}
	for key, value := range entry.Fields {
		fields = append(fields, key, value)
	}
	
	// Call appropriate charmbracelet method
	switch strings.ToUpper(entry.Level) {
	case "DEBUG":
		w.logger.Debug(entry.Message, fields...)
	case "INFO":
		w.logger.Info(entry.Message, fields...)
	case "WARN", "WARNING":
		w.logger.Warn(entry.Message, fields...)
	case "ERROR":
		w.logger.Error(entry.Message, fields...)
	case "FATAL":
		w.logger.Fatal(entry.Message, fields...)
	default:
		w.logger.Info(entry.Message, fields...)
	}
	
	return nil
}

func (w *ConsoleWriter) Close() error {
	return nil // Nothing to close for console
}

// === FILE WRITER ===

func NewFileWriter(enabled bool, filePath string) *FileWriter {
	if !enabled {
		return nil
	}
	
	// Ensure logs directory exists
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil
	}
	
	// Use daily log files if no specific path
	if filePath == "" || filePath == "app.log" {
		filePath = getCurrentLogFile()
	}
	
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil
	}
	
	return &FileWriter{file: file}
}

func (w *FileWriter) Write(entry LogEntry) error {
	// Format as clean text (no ANSI codes)
	timestamp := entry.Timestamp.Format("2006/01/02 15:04:05")
	level := strings.ToUpper(entry.Level)
	
	// Build fields string
	var fieldParts []string
	for key, value := range entry.Fields {
		fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", key, value))
	}
	fieldsStr := ""
	if len(fieldParts) > 0 {
		fieldsStr = " " + strings.Join(fieldParts, " ")
	}
	
	// Write clean text line
	line := fmt.Sprintf("%s %s %s%s\n", timestamp, level, entry.Message, fieldsStr)
	_, err := w.file.WriteString(line)
	return err
}

func (w *FileWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// === BUFFER PERSISTENCE WRITER ===

// BufferWriter persists logs to buffer file for specific application
type BufferWriter struct {
	bufferPath string
	mutex      sync.Mutex
}

func NewCollectorBufferWriter(config Config) *BufferWriter {
	bufferPath := GetBufferPathFromConfig(config)
	return &BufferWriter{
		bufferPath: bufferPath,
		mutex:      sync.Mutex{},
	}
}

func (w *BufferWriter) Write(entry LogEntry) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	// Append entry to buffer file as JSON line
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	
	// Open file in append mode
	file, err := os.OpenFile(w.bufferPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write JSON line
	_, err = file.WriteString(string(data) + "\n")
	return err
}

func (w *BufferWriter) Close() error {
	return nil // Nothing to close
}

// === CIRCULAR BUFFER ===

func NewCircularBuffer(maxEntries int) *CircularBuffer {
	return &CircularBuffer{
		entries:    make([]LogEntry, maxEntries),
		maxEntries: maxEntries,
		index:      0,
		count:      0,
		mutex:      sync.RWMutex{},
	}
}

func (cb *CircularBuffer) Add(entry LogEntry) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.entries[cb.index] = entry
	cb.index = (cb.index + 1) % cb.maxEntries
	
	if cb.count < cb.maxEntries {
		cb.count++
	}
}

func (cb *CircularBuffer) GetAll() []LogEntry {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	if cb.count == 0 {
		return nil
	}
	
	result := make([]LogEntry, cb.count)
	
	if cb.count < cb.maxEntries {
		// Buffer not full yet - entries are from index 0 to count-1
		copy(result, cb.entries[:cb.count])
	} else {
		// Buffer is full - entries are circular from current index
		oldestIndex := cb.index
		copy(result, cb.entries[oldestIndex:])
		copy(result[cb.maxEntries-oldestIndex:], cb.entries[:oldestIndex])
	}
	
	return result
}

func (cb *CircularBuffer) Clear() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.index = 0
	cb.count = 0
	// Don't reallocate slice, just reset counters
}

// === STREAM WRITER ===

func NewStreamWriter() *StreamWriter {
	bufferPath := "/tmp/cyclog-buffer.jsonl"
	pipePath := "/tmp/cyclog-pipe.fifo"
	maxEntries := 200
	
	sw := &StreamWriter{
		bufferPath:  bufferPath,
		pipePath:    pipePath,
		memBuffer:   NewCircularBuffer(maxEntries),
		maxEntries:  maxEntries,
		mutex:       sync.Mutex{},
		initialized: false,
	}
	
	// Cleanup any existing resources from previous sessions
	sw.cleanupOrphanedResources()
	
	// Initialize pipe infrastructure
	sw.initializePipe()
	
	// Load existing buffer from disk if available
	sw.loadBufferFromDisk()
	
	sw.initialized = true
	
	// Log successful initialization
	initEntry := LogEntry{
		Timestamp: time.Now(),
		Level:     "DEBUG",
		Message:   "CycLog initialized successfully",
		Fields: map[string]interface{}{
			"buffer_path": bufferPath,
			"pipe_path":   pipePath,
			"max_entries": maxEntries,
		},
	}
	sw.Write(initEntry)
	
	return sw
}

// NewStreamWriterForProducer creates a stream writer for producers (no buffer, just pipe)
func NewStreamWriterForProducer(pipePath string) *StreamWriter {
	sw := &StreamWriter{
		pipePath:    pipePath,
		memBuffer:   nil, // Producers don't need buffer
		maxEntries:  0,
		mutex:       sync.Mutex{},
		initialized: true, // Simple initialization for producers
	}
	
	return sw
}

func (w *StreamWriter) cleanupOrphanedResources() {
	// Keep pipe and buffer files for inter-session persistence
	// Only clean truly corrupted state if needed
}

func (w *StreamWriter) initializePipe() {
	// Create named pipe if it doesn't exist
	if _, err := os.Stat(w.pipePath); os.IsNotExist(err) {
		if err := syscall.Mkfifo(w.pipePath, 0666); err != nil {
			// Pipe creation failed, but continue without pipe
			return
		}
	}
}

func (w *StreamWriter) loadBufferFromDisk() {
	// Load existing buffer entries into memory for persistence
	if data, err := os.ReadFile(w.bufferPath); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, line := range lines {
			if line != "" {
				var entry LogEntry
				if json.Unmarshal([]byte(line), &entry) == nil {
					w.memBuffer.Add(entry)
				}
			}
		}
	}
}

func (w *StreamWriter) Write(entry LogEntry) error {
	if !w.initialized {
		return nil // Skip if not properly initialized
	}
	
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	// For producers: just write to pipe
	if w.memBuffer == nil {
		w.writeToPipe(entry)
		return nil
	}
	
	// For collectors: full functionality
	// 1. Add to memory buffer (fast, always works)
	w.memBuffer.Add(entry)
	
	// 2. Write to named pipe (real-time streaming)
	w.writeToPipe(entry)
	
	// 3. Persist to disk buffer (durability across sessions)
	w.persistToDisk()
	
	return nil
}

func (w *StreamWriter) writeToPipe(entry LogEntry) {
	// Open pipe for writing if not already open
	if w.pipe == nil {
		pipe, err := os.OpenFile(w.pipePath, os.O_WRONLY, 0)
		if err != nil {
			// No reader connected, skip silently (normal behavior)
			return
		}
		w.pipe = pipe
		w.pipeEncoder = json.NewEncoder(w.pipe)
	}
	
	// Write JSON entry to pipe
	if err := w.pipeEncoder.Encode(entry); err != nil {
		// Pipe broken (reader disconnected), close and retry next time
		w.pipe.Close()
		w.pipe = nil
		w.pipeEncoder = nil
	}
}

func (w *StreamWriter) persistToDisk() {
	// Skip for producers (no buffer)
	if w.memBuffer == nil {
		return
	}
	
	// Get all entries from memory buffer
	entries := w.memBuffer.GetAll()
	if len(entries) == 0 {
		return
	}
	
	// Convert to JSON lines
	var lines []string
	for _, entry := range entries {
		if data, err := json.Marshal(entry); err == nil {
			lines = append(lines, string(data))
		}
	}
	
	// Write to disk
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	
	os.WriteFile(w.bufferPath, []byte(content), 0644)
}

func (w *StreamWriter) Close() error {
	w.cleanupOnce.Do(func() {
		w.mutex.Lock()
		defer w.mutex.Unlock()
		
		// 1. Final persist to disk before closing (only for collectors)
		if w.initialized && w.memBuffer != nil {
			w.persistToDisk()
		}
		
		// 2. Close pipe connection (but keep pipe file for next session)
		if w.pipe != nil {
			w.pipe.Close()
			w.pipe = nil
			w.pipeEncoder = nil
		}
		
		// 3. Clear memory buffer (only for collectors)
		if w.memBuffer != nil {
			w.memBuffer.Clear()
			w.memBuffer = nil
		}
		
		// 4. Keep pipe and buffer files for inter-session persistence
		// Files will be auto-cleaned by OS on reboot (/tmp cleanup)
		
		// 5. Mark as uninitialized
		w.initialized = false
	})
	
	return nil
}


// === CYCLOGGER MAIN ===

func NewCycLogger(config *LoggingConfig) *CycLogger {
	cyclog := &CycLogger{
		writers: make([]LogWriter, 0),
		mutex:   sync.RWMutex{},
	}
	
	// Create console writer
	if consoleWriter := NewConsoleWriter(config.Logging.Console.Enabled, config.Logging.Console.Colors, config.Logging.Console.Caller); consoleWriter != nil {
		cyclog.writers = append(cyclog.writers, consoleWriter)
	}
	
	// Create file writer
	if fileWriter := NewFileWriter(config.Logging.File.Enabled, config.Logging.File.Path); fileWriter != nil {
		cyclog.writers = append(cyclog.writers, fileWriter)
	}
	
	// Create stream writer
	if streamWriter := NewStreamWriter(); streamWriter != nil {
		cyclog.writers = append(cyclog.writers, streamWriter)
	}
	
	return cyclog
}

// === NEW API ===

// Config represents the configuration for cyclog
type Config struct {
	PipeName   string  // Name for pipe (will create /tmp/{name}.fifo)
	PipePath   string  // Explicit pipe path (overrides PipeName)
	Console    bool    // Enable console output
	File       string  // File path for logging (optional)
}

// NewProducer creates a logger that writes to a named pipe
func NewProducer(pipePath string) *CycLogger {
	cyclog := &CycLogger{
		writers: make([]LogWriter, 0),
		mutex:   sync.RWMutex{},
	}
	
	// Only stream writer for producers
	if streamWriter := NewStreamWriterForProducer(pipePath); streamWriter != nil {
		cyclog.writers = append(cyclog.writers, streamWriter)
	}
	
	return cyclog
}

// NewCollector creates a logger that collects from a named pipe and outputs locally
func NewCollector(config Config) *CycLogger {
	cyclog := &CycLogger{
		writers: make([]LogWriter, 0),
		buffer:  NewCircularBuffer(200), // Buffer circulaire de 200 entrées
		mutex:   sync.RWMutex{},
	}
	
	// Console writer if enabled
	if config.Console {
		if consoleWriter := NewConsoleWriter(true, true, false); consoleWriter != nil {
			cyclog.writers = append(cyclog.writers, consoleWriter)
		}
	}
	
	// File writer if specified
	if config.File != "" {
		if fileWriter := NewFileWriter(true, config.File); fileWriter != nil {
			cyclog.writers = append(cyclog.writers, fileWriter)
		}
	}
	
	// Buffer persistence writer - save buffer to disk for this specific app
	if bufferWriter := NewCollectorBufferWriter(config); bufferWriter != nil {
		cyclog.writers = append(cyclog.writers, bufferWriter)
	}
	
	return cyclog
}

// NewStandalone creates a logger for standalone applications (console + file, no pipe)
func NewStandalone(config Config) *CycLogger {
	cyclog := &CycLogger{
		writers: make([]LogWriter, 0),
		mutex:   sync.RWMutex{},
	}
	
	// Console writer if enabled
	if config.Console {
		if consoleWriter := NewConsoleWriter(true, true, false); consoleWriter != nil {
			cyclog.writers = append(cyclog.writers, consoleWriter)
		}
	}
	
	// File writer if specified
	if config.File != "" {
		if fileWriter := NewFileWriter(true, config.File); fileWriter != nil {
			cyclog.writers = append(cyclog.writers, fileWriter)
		}
	}
	
	return cyclog
}

// NewSmartLogger creates a logger with automatic detection of execution context
func NewSmartLogger(appName string) *CycLogger {
	// Check if parent provided a pipe path
	if pipePath := os.Getenv("CYCLOG_PIPE"); pipePath != "" {
		// Producer mode: send logs to parent via pipe
		return NewProducer(pipePath)
	}
	
	// Standalone mode: console + auto log file
	logFile := getAutoLogFile(appName)
	return NewStandalone(Config{
		Console: true,
		File:    logFile,
	})
}

// getAutoLogFile generates automatic log file path
func getAutoLogFile(appName string) string {
	// Ensure logs directory exists
	if err := os.MkdirAll("logs", 0755); err != nil {
		// Fallback to /tmp if can't create logs/
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		pid := os.Getpid()
		return fmt.Sprintf("/tmp/%s_%s_%d.log", timestamp, appName, pid)
	}
	
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	pid := os.Getpid()
	return fmt.Sprintf("logs/%s_%s_%d.log", timestamp, appName, pid)
}

// GetPipePath returns the pipe path for a given config
func GetPipePath(config Config) string {
	if config.PipePath != "" {
		return config.PipePath
	}
	if config.PipeName != "" {
		return fmt.Sprintf("/tmp/%s.fifo", config.PipeName)
	}
	return "/tmp/cyclog.fifo" // Default
}

// StartListening starts collecting logs from the pipe
func (c *CycLogger) StartListening(pipePath string) error {
	// Create the listening pipe infrastructure 
	return c.startPipeListener(pipePath)
}

// startPipeListener creates and listens to the named pipe
func (c *CycLogger) startPipeListener(pipePath string) error {
	// Create named pipe if it doesn't exist
	if _, err := os.Stat(pipePath); os.IsNotExist(err) {
		if err := syscall.Mkfifo(pipePath, 0666); err != nil {
			return fmt.Errorf("failed to create pipe %s: %v", pipePath, err)
		}
	}
	
	// Start listening in a goroutine
	go c.listenToPipe(pipePath)
	
	return nil
}

// ShowBufferHistory displays all entries currently in the buffer
func (c *CycLogger) ShowBufferHistory() {
	if c.buffer == nil {
		return
	}
	
	entries := c.buffer.GetAll()
	if len(entries) == 0 {
		return
	}
	
	// Display buffer header
	c.mutex.RLock()
	if len(c.writers) > 0 {
		if consoleWriter, ok := c.writers[0].(*ConsoleWriter); ok {
			consoleWriter.logger.Info("📚 Buffer history", "entries", len(entries))
		}
	}
	c.mutex.RUnlock()
	
	// Display all buffer entries
	for _, entry := range entries {
		c.mutex.RLock()
		for _, writer := range c.writers {
			writer.Write(entry)
		}
		c.mutex.RUnlock()
	}
}

// AttachReader connects a real-time log reader to this logger
func (c *CycLogger) AttachReader() chan LogEntry {
	readerChan := make(chan LogEntry, 50)
	
	c.mutex.Lock()
	if c.listeners == nil {
		c.listeners = make([]chan LogEntry, 0)
	}
	c.listeners = append(c.listeners, readerChan)
	c.mutex.Unlock()
	
	// Send buffer history first
	go func() {
		if c.buffer != nil {
			entries := c.buffer.GetAll()
			for _, entry := range entries {
				select {
				case readerChan <- entry:
				default: // Don't block if reader is slow
				}
			}
		}
	}()
	
	return readerChan
}

// notifyListeners sends new entries to all connected readers
func (c *CycLogger) notifyListeners(entry LogEntry) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	for _, listener := range c.listeners {
		select {
		case listener <- entry:
		default: // Don't block if listener is slow
		}
	}
}

// listenToPipe reads from the pipe and dispatches to writers
func (c *CycLogger) listenToPipe(pipePath string) {
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
			var entry LogEntry
			if err := decoder.Decode(&entry); err != nil {
				// Pipe closed - writer disconnected, wait for reconnection
				pipe.Close()
				break // Try to reconnect
			}
			
			// 1. Add to central buffer first
			c.buffer.Add(entry)
			
			// 2. Notify all connected readers
			c.notifyListeners(entry)
			
			// 3. Dispatch to all writers of this collector
			c.mutex.RLock()
			for _, writer := range c.writers {
				writer.Write(entry)
			}
			c.mutex.RUnlock()
		}
		
		// Brief pause before reconnection attempt
		time.Sleep(100 * time.Millisecond)
	}
}

// writeToAll sends log entry to all writers
func (c *CycLogger) writeToAll(level, message string, keyvals ...interface{}) {
	c.writeToAllWithStatus(level, message, "", keyvals...)
}

// writeToAllWithStatus sends log entry with status field to all writers
func (c *CycLogger) writeToAllWithStatus(level, message, status string, keyvals ...interface{}) {
	// Parse keyvals to map
	fields := make(map[string]interface{})
	for i := 0; i < len(keyvals)-1; i += 2 {
		if key, ok := keyvals[i].(string); ok && i+1 < len(keyvals) {
			fields[key] = keyvals[i+1]
		}
	}
	
	// Extract status from fields if provided via keyvals
	if statusValue, exists := fields["status"]; exists && status == "" {
		if statusStr, ok := statusValue.(string); ok {
			status = statusStr
			delete(fields, "status") // Remove from fields to avoid duplication
		}
	}
	
	// Create log entry
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
		Status:    status,
	}
	
	// Write to all writers
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	for _, writer := range c.writers {
		writer.Write(entry) // Ignore errors for now
	}
}

// === PUBLIC API ===

func (c *CycLogger) Debug(msg string, keyvals ...interface{}) {
	c.writeToAll("DEBUG", msg, keyvals...)
}

func (c *CycLogger) Info(msg string, keyvals ...interface{}) {
	c.writeToAll("INFO", msg, keyvals...)
}

func (c *CycLogger) Warn(msg string, keyvals ...interface{}) {
	c.writeToAll("WARN", msg, keyvals...)
}

func (c *CycLogger) Error(msg string, keyvals ...interface{}) {
	c.writeToAll("ERROR", msg, keyvals...)
}

func (c *CycLogger) Fatal(msg string, keyvals ...interface{}) {
	c.writeToAll("FATAL", msg, keyvals...)
	os.Exit(1)
}

func (c *CycLogger) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	for _, writer := range c.writers {
		writer.Close()
	}
	
	return nil
}

// === STREAMING FUNCTIONS ===

// GetBufferPath returns the path to the log buffer file for a specific app
func GetBufferPath(pipeName string) string {
	if pipeName == "" {
		pipeName = "cyclog"
	}
	return fmt.Sprintf("/tmp/%s-buffer.jsonl", pipeName)
}

// GetBufferPathFromConfig returns buffer path from config
func GetBufferPathFromConfig(config Config) string {
	pipeName := config.PipeName
	if pipeName == "" {
		pipeName = "cyclog"
	}
	return GetBufferPath(pipeName)
}


// WaitForStatus polls buffer file until specified status is received
func WaitForStatus(targetStatus string, timeoutSeconds int) bool {
	bufferPath := GetBufferPath("cyclog")
	startTime := time.Now()
	seenTimestamps := make(map[string]bool) // Track already seen entries
	
	for {
		// Check timeout
		if timeoutSeconds > 0 && time.Since(startTime) > time.Duration(timeoutSeconds)*time.Second {
			return false
		}
		
		// Read entire buffer file (circular buffer rewrites completely)
		if data, err := os.ReadFile(bufferPath); err == nil {
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			for _, line := range lines {
				if line != "" {
					var entry LogEntry
					if json.Unmarshal([]byte(line), &entry) == nil {
						// Use timestamp as unique key to avoid processing same entry twice
						entryKey := entry.Timestamp.Format(time.RFC3339Nano) + "_" + entry.Status
						
						if !seenTimestamps[entryKey] {
							seenTimestamps[entryKey] = true
							
							// Check if this is the status we're waiting for
							if entry.Status == targetStatus {
								return true
							}
						}
					}
				}
			}
		}
		
		// Wait before next poll
		time.Sleep(100 * time.Millisecond)
	}
}

// === CONFIGURATION ===

func loadLoggingConfig(path string) (*LoggingConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return getDefaultLoggingConfig(), nil
	}

	var logCfg LoggingConfig
	err = yaml.Unmarshal(data, &logCfg)
	if err != nil {
		return getDefaultLoggingConfig(), nil
	}

	return &logCfg, nil
}

func getDefaultLoggingConfig() *LoggingConfig {
	cfg := &LoggingConfig{}
	cfg.Logging.Level = "info"
	cfg.Logging.Console.Enabled = true
	cfg.Logging.Console.Colors = true
	cfg.Logging.Console.Timestamp = true
	cfg.Logging.Console.Caller = false
	cfg.Logging.File.Enabled = true
	cfg.Logging.Format.TimeFormat = "2006-01-02 15:04:05"
	cfg.Logging.Format.LevelFormat = "short"
	return cfg
}

func getCurrentLogFile() string {
	pid := os.Getpid()
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	return fmt.Sprintf("logs/%d_%s.log", pid, timestamp)
}


// === UTILITY FUNCTIONS ===

func RunLogTail() {
	fmt.Printf("→ Live log streaming activated\n")
	fmt.Printf("• Press Ctrl+C to stop\n")
	fmt.Printf("• Will show history + wait for live logs\n")
	fmt.Println("───────────────────────────────────────")
	
	bufferPath := GetBufferPath("cyclog")
	pipePath := "/tmp/cyclog-pipe.fifo" // Legacy default
	
	// Create dedicated console logger for --log display
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		ReportTimestamp: true,
		ReportCaller:    false,
		TimeFormat:      "15:04:05", // Shorter timestamp for --log
		Level:           charmlog.InfoLevel,
	})
	
	// 1. Load buffer history (recent entries)
	loadAndDisplayBuffer(logger, bufferPath)
	
	// 2. Listen to pipe patiently (blocking, patient, autonomous)
	listenToPipePatiently(logger, pipePath)
}

func loadAndDisplayBuffer(logger *charmlog.Logger, bufferPath string) {
	if data, err := os.ReadFile(bufferPath); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, line := range lines {
			if line != "" {
				var entry LogEntry
				if json.Unmarshal([]byte(line), &entry) == nil {
					renderLogEntry(logger, entry)
				}
			}
		}
	}
}

func listenToPipePatiently(logger *charmlog.Logger, pipePath string) {
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
			var entry LogEntry
			if err := decoder.Decode(&entry); err != nil {
				// Pipe closed - writer disconnected, wait for reconnection
				pipe.Close()
				break // Try to reconnect
			}
			
			// Display new entry immediately
			renderLogEntry(logger, entry)
		}
		
		// Brief pause before reconnection attempt
		time.Sleep(100 * time.Millisecond)
	}
}

// renderLogEntry displays a log entry with charmbracelet colors
func renderLogEntry(logger *charmlog.Logger, entry LogEntry) {
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
	case "FATAL":
		logger.Fatal(entry.Message, fields...)
	default:
		logger.Info(entry.Message, fields...)
	}
}

func ShowLogsList() {
	fmt.Printf("📋 Available log files:\n")
	fmt.Printf("🔨 TODO: Implement charm table showing available log files\n")
}