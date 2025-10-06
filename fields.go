package cyclog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is an alias for zap.Logger to avoid importing zap in user code
type Logger = zap.Logger

// Field is an alias for zap.Field to avoid importing zap in user code
type Field = zap.Field

// Config types re-exported from zap
type Config = zap.Config
type AtomicLevel = zap.AtomicLevel
type Level = zapcore.Level

// Config constructors re-exported from zap
var (
	NewProductionConfig = zap.NewProductionConfig
	NewDevelopmentConfig = zap.NewDevelopmentConfig
	NewAtomicLevel = zap.NewAtomicLevel
	NewAtomicLevelAt = zap.NewAtomicLevelAt
	NewExample = zap.NewExample  // Used in fallback cases
	NewNop = zap.NewNop  // No-op logger
)

// Log levels re-exported from zapcore
const (
	DebugLevel = zapcore.DebugLevel
	InfoLevel = zapcore.InfoLevel
	WarnLevel = zapcore.WarnLevel
	ErrorLevel = zapcore.ErrorLevel
	DPanicLevel = zapcore.DPanicLevel
	PanicLevel = zapcore.PanicLevel
	FatalLevel = zapcore.FatalLevel
)

// Commonly used field constructors re-exported from zap
var (
	// String constructs a field with the given key and value
	String = zap.String
	
	// Int constructs a field with the given key and value
	Int = zap.Int
	
	// Int64 constructs a field with the given key and value
	Int64 = zap.Int64
	
	// Float64 constructs a field with the given key and value
	Float64 = zap.Float64
	
	// Bool constructs a field with the given key and value
	Bool = zap.Bool
	
	// Error constructs a field that carries an error
	Error = zap.Error
	
	// Time constructs a field with the given key and value
	Time = zap.Time
	
	// Duration constructs a field with the given key and value
	Duration = zap.Duration
	
	// Any constructs a field with the given key and value using reflection
	Any = zap.Any
	
	// Binary constructs a field that carries an opaque binary blob
	Binary = zap.Binary
	
	// ByteString constructs a field that carries a []byte
	ByteString = zap.ByteString
	
	// Complex64 constructs a field that carries a complex number
	Complex64 = zap.Complex64
	
	// Complex128 constructs a field that carries a complex number
	Complex128 = zap.Complex128
	
	// Float32 constructs a field with the given key and value
	Float32 = zap.Float32
	
	// Int8 constructs a field with the given key and value
	Int8 = zap.Int8
	
	// Int16 constructs a field with the given key and value
	Int16 = zap.Int16
	
	// Int32 constructs a field with the given key and value
	Int32 = zap.Int32
	
	// Uint constructs a field with the given key and value
	Uint = zap.Uint
	
	// Uint8 constructs a field with the given key and value
	Uint8 = zap.Uint8
	
	// Uint16 constructs a field with the given key and value
	Uint16 = zap.Uint16
	
	// Uint32 constructs a field with the given key and value
	Uint32 = zap.Uint32
	
	// Uint64 constructs a field with the given key and value
	Uint64 = zap.Uint64
	
	// Uintptr constructs a field with the given key and value
	Uintptr = zap.Uintptr
	
	// Strings constructs a field that carries a slice of strings
	Strings = zap.Strings
	
	// Ints constructs a field that carries a slice of integers
	Ints = zap.Ints
	
	// Stack constructs a field that stores a stacktrace of the current goroutine
	Stack = zap.Stack
	
	// StackSkip constructs a field similarly to Stack, but also skips the given number of frames
	StackSkip = zap.StackSkip
)