package agentforge

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
)

var (
	envCache map[string]string
	envOnce  sync.Once
)

func initEnvCache() {
	envCache = make(map[string]string)
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}

	for i := 0; i < 5; i++ {
		envPath := filepath.Join(dir, ".env")
		if env, err := godotenv.Read(envPath); err == nil {
			for k, v := range env {
				envCache[k] = v
			}
			break // Found and loaded
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// GetEnvVar retrieves an environment variable with fallback priority:
// 1. os.Getenv() (environment variables take precedence)
// 2. .env file (in current directory or parent directories up to project root)
// 3. Returns error if not found
//
// Parameters:
//   - key: The environment variable name to retrieve
//
// Returns:
//   - string: The environment variable value
//   - error: Error if the variable is not found in .env file or os environment
func GetEnvVar(key string) (string, error) {
	// Check os.Getenv() first (environment variables take precedence)
	if value := os.Getenv(key); value != "" {
		return value, nil
	}

	// Read from cached .env
	envOnce.Do(initEnvCache)
	if value, ok := envCache[key]; ok && value != "" {
		return value, nil
	}

	// Not found anywhere
	return "", fmt.Errorf("environment variable %s not found in .env file or os environment", key)
}
