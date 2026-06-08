package update

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

// Logger is a secret-redacting logger for the update engine.
type Logger struct {
	mu      sync.Mutex
	output  io.Writer
	secrets []string
}

// secretPattern matches common secret patterns.
var secretPattern = regexp.MustCompile(`(?i)(password|secret|key|token|auth|credential)[:=]\s*['"]?([^'"\s]+)['"]?`)

// NewLogger creates a new secret-redacting logger.
func NewLogger(verbose bool) *Logger {
	l := &Logger{
		output: os.Stderr,
	}

	// Collect secrets from environment
	for _, name := range []string{
		"ORVIX_SERVER_SECRET_KEY",
		"ORVIX_MASTER_KEY",
		"ORVIX_DB_PATH",
	} {
		if val := os.Getenv(name); val != "" {
			l.secrets = append(l.secrets, val)
		}
	}

	// Set log level
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	return l
}

// redact replaces known secrets with [REDACTED].
func (l *Logger) redact(s string) string {
	result := s

	// Replace known secrets
	for _, secret := range l.secrets {
		if len(secret) > 4 {
			result = strings.ReplaceAll(result, secret, "[REDACTED]")
		}
	}

	// Replace patterns
	result = secretPattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := secretPattern.FindStringSubmatch(match)
		if len(parts) >= 3 {
			return parts[1] + "=[REDACTED]"
		}
		return "[REDACTED]"
	})

	return result
}

// Log writes a log entry with secret redaction.
func (l *Logger) Log(level, msg string, fields map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	redactedMsg := l.redact(msg)

	var fieldPairs []interface{}
	for k, v := range fields {
		redactedV := l.redact(fmt.Sprintf("%v", v))
		fieldPairs = append(fieldPairs, k, redactedV)
	}

	fmt.Fprintf(l.output, "[%s] %s", level, redactedMsg)
	if len(fieldPairs) > 0 {
		fmt.Fprint(l.output, " ")
		for i := 0; i < len(fieldPairs); i += 2 {
			fmt.Fprintf(l.output, "%v=%v ", fieldPairs[i], fieldPairs[i+1])
		}
	}
	fmt.Fprintln(l.output)
}

// Info logs an info message.
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.Log("INFO", msg, f)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.Log("WARN", msg, f)
}

// Error logs an error message.
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.Log("ERROR", msg, f)
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.Log("DEBUG", msg, f)
}

// PrintStep prints a step indicator.
func (l *Logger) PrintStep(step, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.output, "\n==> %s: %s\n", step, msg)
}

// PrintSuccess prints a success message.
func (l *Logger) PrintSuccess(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.output, "✓ %s\n", msg)
}

// PrintFailure prints a failure message.
func (l *Logger) PrintFailure(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.output, "✗ %s\n", msg)
}

// PrintWarning prints a warning message.
func (l *Logger) PrintWarning(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.output, "⚠ %s\n", msg)
}

// WriteLogToFile writes the log output to a file.
func (l *Logger) WriteLogToFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	l.mu.Lock()
	l.output = io.MultiWriter(os.Stderr, file)
	l.mu.Unlock()

	return nil
}

// Global logger instance
var globalLogger *Logger

// InitLogger initializes the global logger.
func InitLogger(verbose bool) {
	globalLogger = NewLogger(verbose)
}

// GetLogger returns the global logger.
func GetLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewLogger(false)
	}
	return globalLogger
}