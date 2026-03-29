package util

import (
	"fmt"
	"io"
	"log"
	"os"
)

var (
	// Logger is the global logger
	Logger *log.Logger
	// Verbose enables verbose logging
	Verbose bool
)

func init() {
	SetOutput(os.Stderr)
}

// SetOutput sets the output destination for the logger
func SetOutput(w io.Writer) {
	Logger = log.New(w, "", log.LstdFlags)
}

// SetVerbose enables or disables verbose logging
func SetVerbose(v bool) {
	Verbose = v
	if v {
		Logger.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}

// Debug logs a debug message (only in verbose mode)
func Debug(format string, args ...interface{}) {
	if Verbose {
		Logger.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	Logger.Printf("[INFO] "+format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	Logger.Printf("[WARN] "+format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	Logger.Printf("[ERROR] "+format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(format string, args ...interface{}) {
	Logger.Printf("[FATAL] "+format, args...)
	os.Exit(1)
}

// Print prints a message to stdout
func Print(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// PrintError prints an error message to stderr
func PrintError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
