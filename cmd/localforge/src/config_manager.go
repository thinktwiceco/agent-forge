// ─── Config Authority ─────────────────────────────────────────────────────────
//
// ConfigManager is the single owner of the on-disk configuration state.
// It sits between the YAML file and the AgentManager:
//
//	config.yaml  ──Load()──►  ConfigManager  ──GetConfig()──►  AgentManager
//	                 ▲                │
//	        Save()───┘        Update*() methods (called by HTTP handlers)
//
// Design decisions:
//
//  1. rawYAML storage: Load() keeps the original YAML bytes (with any
//     ${VAR_NAME} placeholders intact). Update methods patch specific nodes
//     using yaml.Node so that unreferenced fields — including ${...} env-var
//     references in tool URLs — are never overwritten. Save() writes rawYAML
//     back to disk, not a re-marshaled struct (which would silently destroy
//     all ${...} placeholders).
//
//  2. Two-level read: after patching rawYAML, the manager re-interpolates env
//     vars to produce the in-memory builder.Config used to build the agent.
//     The UI always receives interpolated values; it writes back literal values
//     only for the fields it actually changed.
//
//  3. Thread safety: all reads/writes are guarded by a sync.RWMutex.
//     HTTP handlers may call Update* concurrently; each grabs the lock,
//     patches rawYAML, writes the file, then updates config.

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
	rawYAML    string         // original file content, ${VAR} placeholders intact
	config     builder.Config // interpolated, used for building the agent
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

	raw := string(data)

	interpolatedData, err := interpolateEnvVars(raw)
	if err != nil {
		return fmt.Errorf("interpolate env vars: %w", err)
	}

	var cfg builder.Config
	if err := yaml.Unmarshal([]byte(interpolatedData), &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	cm.mu.Lock()
	cm.rawYAML = raw
	cm.config = cfg
	cm.mu.Unlock()
	return nil
}

// interpolateEnvVars replaces ${VAR_NAME} with the corresponding environment variable value.
// Returns an error if a referenced environment variable is not set.
func interpolateEnvVars(content string) (string, error) {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	var interpolationError error
	result := re.ReplaceAllStringFunc(content, func(match string) string {
		varName := re.FindStringSubmatch(match)[1]
		value := os.Getenv(varName)
		if value == "" {
			interpolationError = fmt.Errorf("environment variable not set: %s", varName)
			return match
		}
		return value
	})

	if interpolationError != nil {
		return "", interpolationError
	}

	return result, nil
}

// save writes rawYAML back to disk and re-parses the interpolated config.
// Must be called with cm.mu held (write lock).
func (cm *ConfigManager) save() error {
	if err := os.WriteFile(cm.configPath, []byte(cm.rawYAML), 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	interpolated, err := interpolateEnvVars(cm.rawYAML)
	if err != nil {
		return fmt.Errorf("interpolate env vars after save: %w", err)
	}

	var cfg builder.Config
	if err := yaml.Unmarshal([]byte(interpolated), &cfg); err != nil {
		return fmt.Errorf("re-parse config after save: %w", err)
	}
	cm.config = cfg
	return nil
}

// patchYAMLNode decodes rawYAML into a yaml.Node tree, applies the patcher
// function, then re-encodes back to rawYAML. This preserves all unchanged
// nodes verbatim — including ${VAR} references and YAML comments.
func (cm *ConfigManager) patchYAMLNode(patcher func(root *yaml.Node) error) error {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(cm.rawYAML), &root); err != nil {
		return fmt.Errorf("decode yaml for patch: %w", err)
	}

	if err := patcher(&root); err != nil {
		return err
	}

	out, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("re-encode yaml after patch: %w", err)
	}
	cm.rawYAML = string(out)
	return nil
}

// yamlMappingNode navigates a yaml.Node mapping by key sequence.
// Each element in keys is a mapping key at successive nesting levels.
// Returns the value node for the last key, creating intermediate mapping nodes
// and the final key entry if they don't exist.
func yamlMappingNode(root *yaml.Node, keys ...string) (*yaml.Node, error) {
	// Unwrap document node
	node := root
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	for i, key := range keys {
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("expected mapping at key %q, got kind %d", key, node.Kind)
		}
		found := false
		for j := 0; j+1 < len(node.Content); j += 2 {
			if node.Content[j].Value == key {
				if i == len(keys)-1 {
					return node.Content[j+1], nil
				}
				node = node.Content[j+1]
				found = true
				break
			}
		}
		if !found {
			// Create missing intermediate mapping or leaf node
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
			var valNode *yaml.Node
			if i == len(keys)-1 {
				valNode = &yaml.Node{Kind: yaml.ScalarNode, Value: ""}
			} else {
				valNode = &yaml.Node{Kind: yaml.MappingNode}
			}
			node.Content = append(node.Content, keyNode, valNode)
			node = valNode
			if i < len(keys)-1 {
				// continue navigating the newly created node
			} else {
				return valNode, nil
			}
		}
	}
	return node, nil
}

func (cm *ConfigManager) GetConfig() builder.Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

func (cm *ConfigManager) ConfigPath() string {
	return cm.configPath
}

// UpdateAgentFields updates top-level agent identity fields in config.yaml.
func (cm *ConfigManager) UpdateAgentFields(req UpdateAgentRequest) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	err := cm.patchYAMLNode(func(root *yaml.Node) error {
		setScalar := func(keys []string, val string) error {
			n, err := yamlMappingNode(root, keys...)
			if err != nil {
				return err
			}
			n.Kind = yaml.ScalarNode
			n.Value = val
			n.Tag = "!!str"
			return nil
		}

		if req.Name != nil {
			if err := setScalar([]string{"agent", "name"}, *req.Name); err != nil {
				return err
			}
		}
		if req.Model != nil {
			if err := setScalar([]string{"agent", "model"}, *req.Model); err != nil {
				return err
			}
		}
		if req.SystemPrompt != nil {
			n, err := yamlMappingNode(root, "agent", "system_prompt")
			if err != nil {
				return err
			}
			n.Kind = yaml.ScalarNode
			n.Value = *req.SystemPrompt
			n.Tag = "!!str"
			n.Style = yaml.LiteralStyle
		}
		if req.WorkingDir != nil {
			if err := setScalar([]string{"agent", "working_dir"}, *req.WorkingDir); err != nil {
				return err
			}
		}
		if req.Persistence != nil {
			if err := setScalar([]string{"agent", "persistence"}, *req.Persistence); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return cm.save()
}

// UpdatePlugins replaces the plugins list in config.yaml.
func (cm *ConfigManager) UpdatePlugins(plugins []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	err := cm.patchYAMLNode(func(root *yaml.Node) error {
		n, err := yamlMappingNode(root, "agent", "plugins")
		if err != nil {
			return err
		}
		n.Kind = yaml.SequenceNode
		n.Tag = "!!seq"
		n.Value = ""
		n.Content = nil
		for _, p := range plugins {
			n.Content = append(n.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: p,
				Tag:   "!!str",
			})
		}
		return nil
	})
	if err != nil {
		return err
	}
	return cm.save()
}

// UpdateToolConfig updates settings for a named tool in config.yaml.
func (cm *ConfigManager) UpdateToolConfig(toolName string, update UpdateToolConfigRequest) error {
	if update.PostgresURL == nil &&
		update.Mode == nil &&
		update.AllowedTables == nil &&
		update.AllowedSchemas == nil {
		return errors.New("no updates provided")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check the tool exists in the parsed config first.
	found := false
	for _, t := range cm.config.Agent.Tools {
		if t.Name == toolName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("tool not found: %s", toolName)
	}

	err := cm.patchYAMLNode(func(root *yaml.Node) error {
		// Navigate to agent.tools sequence
		toolsNode, err := yamlMappingNode(root, "agent", "tools")
		if err != nil {
			return err
		}
		if toolsNode.Kind != yaml.SequenceNode {
			return fmt.Errorf("agent.tools is not a sequence")
		}

		for _, toolNode := range toolsNode.Content {
			if toolNode.Kind != yaml.MappingNode {
				continue
			}
			// Find the name key and check if it matches
			nameVal := ""
			for j := 0; j+1 < len(toolNode.Content); j += 2 {
				if toolNode.Content[j].Value == "name" {
					nameVal = toolNode.Content[j+1].Value
					break
				}
			}
			if nameVal != toolName {
				continue
			}

			setField := func(key, val string) {
				for j := 0; j+1 < len(toolNode.Content); j += 2 {
					if toolNode.Content[j].Value == key {
						toolNode.Content[j+1].Value = val
						return
					}
				}
				toolNode.Content = append(toolNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"},
				)
			}

			if update.PostgresURL != nil {
				setField("postgresURL", *update.PostgresURL)
			}
			if update.Mode != nil {
				setField("mode", *update.Mode)
			}
			if update.AllowedTables != nil {
				// Replace sequence
				for j := 0; j+1 < len(toolNode.Content); j += 2 {
					if toolNode.Content[j].Value == "allowedTables" {
						seq := toolNode.Content[j+1]
						seq.Kind = yaml.SequenceNode
						seq.Tag = "!!seq"
						seq.Value = ""
						seq.Content = nil
						for _, t := range *update.AllowedTables {
							seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: t, Tag: "!!str"})
						}
						goto doneAllowedTables
					}
				}
				{
					seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
					for _, t := range *update.AllowedTables {
						seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: t, Tag: "!!str"})
					}
					toolNode.Content = append(toolNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "allowedTables", Tag: "!!str"},
						seq,
					)
				}
			doneAllowedTables:
			}
			if update.AllowedSchemas != nil {
				for j := 0; j+1 < len(toolNode.Content); j += 2 {
					if toolNode.Content[j].Value == "allowedSchemas" {
						seq := toolNode.Content[j+1]
						seq.Kind = yaml.SequenceNode
						seq.Tag = "!!seq"
						seq.Value = ""
						seq.Content = nil
						for _, s := range *update.AllowedSchemas {
							seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: s, Tag: "!!str"})
						}
						goto doneAllowedSchemas
					}
				}
				{
					seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
					for _, s := range *update.AllowedSchemas {
						seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: s, Tag: "!!str"})
					}
					toolNode.Content = append(toolNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "allowedSchemas", Tag: "!!str"},
						seq,
					)
				}
			doneAllowedSchemas:
			}
			return nil
		}
		return fmt.Errorf("tool not found in YAML: %s", toolName)
	})
	if err != nil {
		return err
	}
	return cm.save()
}
