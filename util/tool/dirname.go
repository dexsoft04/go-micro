// Package tool provides common utility functions
package tool

import (
	"os"
	"path/filepath"
	"strings"
)

// GetDirectoryNameLower returns the current working directory name in lowercase
// If there's an error getting the directory, it returns "go-micro" as fallback
func GetDirectoryNameLower() string {
	wd, err := os.Getwd()
	if err != nil {
		return "go-micro" // fallback value
	}
	
	dirName := filepath.Base(wd)
	return strings.ToLower(dirName)
}

// GetServiceName returns the service name based on the execution context:
// 1. If running as compiled binary, returns the executable name (without extension)
// 2. If running via "go run" with module name (e.g., go run github.com/z/abc), returns the last part of module name (abc)
// 3. If running via "go run" with file path, falls back to current directory name
// 4. If all else fails, returns "go-micro" as fallback
func GetServiceName() string {
	// Try to get the executable path
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to directory name if executable path fails
		return GetDirectoryNameLower()
	}
	
	// Extract the base name from the executable path
	execName := filepath.Base(execPath)
	
	// Check if this looks like a "go run" temporary executable
	// go run creates temporary executables in /tmp/go-build* directories
	if strings.Contains(execPath, "/tmp/go-build") || 
	   strings.Contains(execPath, "\\Temp\\go-build") ||
	   strings.HasPrefix(execName, "go_build_") {
		// This is likely "go run", check for module name in command line args
		return getGoRunServiceName()
	}
	
	// Remove common executable extensions
	execName = strings.TrimSuffix(execName, ".exe")
	execName = strings.TrimSuffix(execName, ".bin")
	
	// Convert to lowercase for consistency
	return strings.ToLower(execName)
}

// getGoRunServiceName extracts service name when running via "go run"
// Checks command line arguments for module name pattern (e.g., github.com/z/abc -> abc)
func getGoRunServiceName() string {
	// Check command line arguments for module name
	for _, arg := range os.Args {
		// Look for module-like patterns (contains /)
		if strings.Contains(arg, "/") && !strings.HasPrefix(arg, "-") {
			// This could be a module name like github.com/z/abc
			parts := strings.Split(arg, "/")
			if len(parts) > 1 {
				// Get the last part of the path/module name
				lastPart := parts[len(parts)-1]
				// Remove any file extensions
				lastPart = strings.TrimSuffix(lastPart, ".go")
				if lastPart != "" {
					return strings.ToLower(lastPart)
				}
			}
		}
	}
	
	// If no module name found, fallback to directory name
	return GetDirectoryNameLower()
}