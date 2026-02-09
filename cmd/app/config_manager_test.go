package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInterpolateEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
		wantErr  bool
	}{
		{
			name:  "single env var",
			input: "url: ${DATABASE_URL}",
			envVars: map[string]string{
				"DATABASE_URL": "postgres://localhost:5432/db",
			},
			expected: "url: postgres://localhost:5432/db",
			wantErr:  false,
		},
		{
			name:  "multiple env vars",
			input: "host: ${DB_HOST}, port: ${DB_PORT}",
			envVars: map[string]string{
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
			},
			expected: "host: localhost, port: 5432",
			wantErr:  false,
		},
		{
			name:  "missing env var",
			input: "url: ${MISSING_VAR}",
			envVars: map[string]string{
				"OTHER_VAR": "value",
			},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "no env vars",
			input:    "url: hardcoded_value",
			envVars:  map[string]string{},
			expected: "url: hardcoded_value",
			wantErr:  false,
		},
		{
			name:  "env var with special characters",
			input: "url: ${DATABASE_URL}",
			envVars: map[string]string{
				"DATABASE_URL": "postgres://user:pass@host:5432/db?sslmode=require&schema=private",
			},
			expected: "url: postgres://user:pass@host:5432/db?sslmode=require&schema=private",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Run interpolation
			result, err := interpolateEnvVars(tt.input)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("interpolateEnvVars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check result
			if !tt.wantErr && result != tt.expected {
				t.Errorf("interpolateEnvVars() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigManagerLoadWithInterpolation(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	configContent := `agent:
  name: "Test Agent"
  system_prompt: "Test prompt"
  model: "test-model"
  working_dir: "/tmp"
  persistence: "json"
  tools:
    - name: postgres
      postgresURL: "${TEST_DATABASE_URL}"
      mode: "read"
      allowedSchemas:
        - "public"
      allowedTables:
        - "users"
`

	// Write config file
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set environment variable
	testDBURL := "postgres://test:test@localhost:5432/testdb"
	_ = os.Setenv("TEST_DATABASE_URL", testDBURL)
	defer os.Unsetenv("TEST_DATABASE_URL")

	// Create config manager and load config
	cm, err := NewConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create config manager: %v", err)
	}

	// Verify the environment variable was interpolated
	cfg := cm.GetConfig()
	if len(cfg.Agent.Tools) == 0 {
		t.Fatal("No tools found in config")
	}

	postgresTool := cfg.Agent.Tools[0]
	if postgresTool.PostgresURL != testDBURL {
		t.Errorf("PostgresURL = %v, want %v", postgresTool.PostgresURL, testDBURL)
	}
}
