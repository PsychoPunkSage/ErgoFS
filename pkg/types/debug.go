package types

import (
	"fmt"
	"os"
)

// Debug prints a message if the debug level is high enough
func Debug(level int, format string, args ...interface{}) {
	if level <= GlobalConfig.DebugLevel {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// Error prints an error message
func Error(format string, args ...interface{}) {
	Debug(EROFS_ERR, "Error: "+format, args...)
}

// Warning prints a warning message
func Warning(format string, args ...interface{}) {
	Debug(EROFS_WARN, "Warning: "+format, args...)
}

// Info prints an informational message
func Info(format string, args ...interface{}) {
	Debug(EROFS_INFO, format, args...)
}
