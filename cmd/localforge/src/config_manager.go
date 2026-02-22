package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/thinktwiceco/agent-forge/src/builder"
	"gopkg.in/yaml.v3"
)

type ConfigManager struct {
	mu         sync.RWMutex
	configPath string
	config     builder.Config
}

func NewConfigManager(configPath string) (*ConfigManager, error) {
	manager := &ConfigManager{
		configPath: configPath,
	}
	if err := manager.Load(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	// Interpolate environment variables in the YAML content
	interpolatedData, err := interpolateEnvVars(string(data))
	if err != nil {
		return fmt.Errorf("interpolate env vars: %w", err)
	}

	var cfg builder.Config
	if err := yaml.Unmarshal([]byte(interpolatedData), &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	cm.mu.Lock()
	cm.config = cfg
	cm.mu.Unlock()
	return nil
}

// interpolateEnvVars replaces ${VAR_NAME} with the corresponding environment variable value.
// Returns an error if a referenced environment variable is not set.
func interpolateEnvVars(content string) (string, error) {
	// Match ${VAR_NAME} pattern
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	var interpolationError error
	result := re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		varName := re.FindStringSubmatch(match)[1]

		// Get environment variable value
		value := os.Getenv(varName)
		if value == "" {
			interpolationError = fmt.Errorf("environment variable not set: %s", varName)
			return match // Return original if not found
		}

		return value
	})

	if interpolationError != nil {
		return "", interpolationError
	}

	return result, nil
}

func (cm *ConfigManager) Save() error {
	cm.mu.RLock()
	cfg := cm.config
	cm.mu.RUnlock()

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (cm *ConfigManager) GetConfig() builder.Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

func (cm *ConfigManager) ConfigPath() string {
	return cm.configPath
}

func (cm *ConfigManager) UpdateToolConfig(toolName string, update UpdateToolConfigRequest) error {
	if update.Root == nil &&
		update.PostgresURL == nil &&
		update.Mode == nil &&
		update.AllowedTables == nil &&
		update.AllowedSchemas == nil {
		return errors.New("no updates provided")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i := range cm.config.Agent.Tools {
		if cm.config.Agent.Tools[i].Name != toolName {
			continue
		}

		if update.Root != nil {
			cm.config.Agent.Tools[i].Root = *update.Root
		}
		if update.PostgresURL != nil {
			cm.config.Agent.Tools[i].PostgresURL = *update.PostgresURL
		}
		if update.Mode != nil {
			cm.config.Agent.Tools[i].Mode = *update.Mode
		}
		if update.AllowedTables != nil {
			cm.config.Agent.Tools[i].AllowedTables = *update.AllowedTables
		}
		if update.AllowedSchemas != nil {
			cm.config.Agent.Tools[i].AllowedSchemas = *update.AllowedSchemas
		}

		return cm.Save()
	}

	return fmt.Errorf("tool not found: %s", toolName)
}
