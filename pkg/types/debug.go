package types

import (
	"fmt"
)

// Debug-level constants
const (
	EROFS_DEBUG   uint8 = 3
	EROFS_VERBOSE uint8 = 4
)

// Global debug level
var G_DEBUG_LEVEL uint8 = EROFS_WARN

// SetDebugLevel sets the global debug level
func SetDebugLevel(level uint8) {
	G_DEBUG_LEVEL = level
}

// Erofs_err prints an error message
func Erofs_err(format string, a ...interface{}) {
	if G_DEBUG_LEVEL >= EROFS_ERR {
		fmt.Printf("[ERROR] "+format+"\n", a...)
	}
}

// Erofs_warn prints a warning message
func Erofs_warn(format string, a ...interface{}) {
	if G_DEBUG_LEVEL >= EROFS_WARN {
		fmt.Printf("[WARN] "+format+"\n", a...)
	}
}

// Erofs_info prints an info message
func Erofs_info(format string, a ...interface{}) {
	if G_DEBUG_LEVEL >= EROFS_INFO {
		fmt.Printf("[INFO] "+format+"\n", a...)
	}
}

// Erofs_debug prints a debug message
func Erofs_debug(format string, a ...interface{}) {
	if G_DEBUG_LEVEL >= EROFS_DEBUG {
		fmt.Printf("[DEBUG] "+format+"\n", a...)
	}
}

// Erofs_verbose prints a verbose debug message
func Erofs_verbose(format string, a ...interface{}) {
	if G_DEBUG_LEVEL >= EROFS_VERBOSE {
		fmt.Printf("[VERBOSE] "+format+"\n", a...)
	}
}

// DumpHex prints a hex dump of a byte slice
func DumpHex(data []byte, prefix string) {
	if G_DEBUG_LEVEL >= EROFS_DEBUG {
		fmt.Printf("%s: ", prefix)
		for i, b := range data {
			fmt.Printf("%02x ", b)
			if (i+1)%16 == 0 && i < len(data)-1 {
				fmt.Printf("\n%s: ", prefix)
			}
		}
		fmt.Println()
	}
}
