package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const secretsFileName = "secrets.json"

// loadSecrets reads vault/secrets.json and returns the map of {key: encryptedValue}.
// Returns an empty map if the file does not exist yet.
func loadSecrets(dir string) (map[string]string, error) {
	path := filepath.Join(dir, secretsFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("vault: failed to read secrets file: %w", err)
	}

	var secrets map[string]string
	if err := json.Unmarshal(data, &secrets); err != nil {
		return nil, fmt.Errorf("vault: failed to parse secrets file: %w", err)
	}

	return secrets, nil
}

// saveSecrets writes the secrets map to vault/secrets.json, creating the directory
// if it does not exist.
func saveSecrets(dir string, secrets map[string]string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("vault: failed to create vault directory: %w", err)
	}

	data, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("vault: failed to serialize secrets: %w", err)
	}

	path := filepath.Join(dir, secretsFileName)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("vault: failed to write secrets file: %w", err)
	}

	return nil
}
