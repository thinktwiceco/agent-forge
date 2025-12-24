package agentforge

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loadEnvFile reads environment variables from a .env file.
//
// This function reads key=value pairs from a .env file and returns them as a map.
// Lines starting with # are treated as comments and ignored.
// Empty lines are ignored.
func loadEnvFile(filePath string) (map[string]string, error) {
	env := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}
			env[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return env, nil
}

// GetEnvVar retrieves an environment variable with fallback priority:
// 1. .env file (in current directory or parent directories up to project root)
// 2. os.Getenv()
// 3. Returns error if not found
//
// Parameters:
//   - key: The environment variable name to retrieve
//
// Returns:
//   - string: The environment variable value
//   - error: Error if the variable is not found in .env file or os environment
func GetEnvVar(key string) (string, error) {
	// Try to find .env file starting from current directory
	// Search up directories until we find .env file or reach filesystem root
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}

	// Search up to 5 levels to handle deep test directory structures
	for i := 0; i < 5; i++ {
		envPath := filepath.Join(dir, ".env")
		if env, err := loadEnvFile(envPath); err == nil {
			if value, ok := env[key]; ok && value != "" {
				return value, nil
			}
		}
		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}

	// Fallback to os.Getenv()
	if value := os.Getenv(key); value != "" {
		return value, nil
	}

	// Not found anywhere
	return "", fmt.Errorf("environment variable %s not found in .env file or os environment", key)
}
