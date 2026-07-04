package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (cm *ConfigManager) AddTool(name string, params map[string]any) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, t := range cm.config.Agent.Tools {
		if t.Name == name {
			return fmt.Errorf("tool already exists: %s", name)
		}
	}

	err := cm.patchYAMLNode(func(root *yaml.Node) error {
		toolsNode, err := ensureToolsSequence(root)
		if err != nil {
			return err
		}
		toolsNode.Content = append(toolsNode.Content, buildToolYAMLNode(name, params))
		return nil
	})
	if err != nil {
		return err
	}
	return cm.save()
}

func (cm *ConfigManager) RemoveTool(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	found := false
	for _, t := range cm.config.Agent.Tools {
		if t.Name == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("tool not found: %s", name)
	}

	err := cm.patchYAMLNode(func(root *yaml.Node) error {
		toolsNode, err := yamlMappingNode(root, "agent", "tools")
		if err != nil {
			return err
		}
		if toolsNode.Kind != yaml.SequenceNode {
			return fmt.Errorf("agent.tools is not a sequence")
		}
		var kept []*yaml.Node
		for _, toolNode := range toolsNode.Content {
			if toolNode.Kind != yaml.MappingNode {
				kept = append(kept, toolNode)
				continue
			}
			toolName := mappingScalar(toolNode, "name")
			if toolName == name {
				continue
			}
			kept = append(kept, toolNode)
		}
		if len(kept) == len(toolsNode.Content) {
			return fmt.Errorf("tool not found in YAML: %s", name)
		}
		toolsNode.Content = kept
		return nil
	})
	if err != nil {
		return err
	}
	return cm.save()
}

func (cm *ConfigManager) AddPlugin(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	name = strings.TrimSpace(name)
	for _, p := range cm.config.Agent.Plugins {
		if p == name {
			return fmt.Errorf("plugin already listed: %s", name)
		}
	}
	plugins := append(append([]string{}, cm.config.Agent.Plugins...), name)
	return cm.updatePluginsLocked(plugins)
}

func (cm *ConfigManager) RemovePlugin(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	name = strings.TrimSpace(name)
	found := false
	var plugins []string
	for _, p := range cm.config.Agent.Plugins {
		if p == name {
			found = true
			continue
		}
		plugins = append(plugins, p)
	}
	if !found {
		return fmt.Errorf("plugin not found: %s", name)
	}
	return cm.updatePluginsLocked(plugins)
}

func (cm *ConfigManager) SetHeartbeat(every string) error {
	every = strings.TrimSpace(every)
	if every == "" {
		return fmt.Errorf("every is required")
	}
	if _, err := parseHeartbeatInterval(every); err != nil {
		return err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	err := cm.patchYAMLNode(func(root *yaml.Node) error {
		hbNode, err := yamlMappingNode(root, "agent", "heartbeat")
		if err != nil {
			return err
		}
		if hbNode.Kind != yaml.MappingNode {
			hbNode.Kind = yaml.MappingNode
			hbNode.Tag = "!!map"
			hbNode.Content = nil
		}
		setMappingScalar(hbNode, "every", every)
		return nil
	})
	if err != nil {
		return err
	}
	return cm.save()
}

func (cm *ConfigManager) SetDream(status, dreamTime string) error {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "on", "off":
	default:
		return fmt.Errorf("status must be on or off")
	}
	dreamTime = strings.TrimSpace(dreamTime)
	if dreamTime != "" && !isDreamTimeValid(dreamTime) {
		return fmt.Errorf("invalid dream time %q; use HH:MM or HH:MM:SS", dreamTime)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	err := cm.patchYAMLNode(func(root *yaml.Node) error {
		bpNode, err := yamlMappingNode(root, "agent", "brain_plugin")
		if err != nil {
			return err
		}
		if bpNode.Kind != yaml.MappingNode {
			bpNode.Kind = yaml.MappingNode
			bpNode.Tag = "!!map"
			bpNode.Content = nil
		}
		setMappingScalar(bpNode, "dream", status)
		if dreamTime != "" {
			setMappingScalar(bpNode, "dreamTime", dreamTime)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return cm.save()
}

func (cm *ConfigManager) updatePluginsLocked(plugins []string) error {
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

func ensureToolsSequence(root *yaml.Node) (*yaml.Node, error) {
	n, err := yamlMappingNode(root, "agent", "tools")
	if err != nil {
		return nil, err
	}
	if n.Kind != yaml.SequenceNode {
		n.Kind = yaml.SequenceNode
		n.Tag = "!!seq"
		n.Value = ""
		n.Content = nil
	}
	return n, nil
}

func buildToolYAMLNode(name string, params map[string]any) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	setMappingScalar(node, "name", name)
	for key, val := range params {
		appendMappingValue(node, key, val)
	}
	return node
}

func setMappingScalar(mapping *yaml.Node, key, val string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1].Kind = yaml.ScalarNode
			mapping.Content[i+1].Value = val
			mapping.Content[i+1].Tag = "!!str"
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"},
	)
}

func appendMappingValue(mapping *yaml.Node, key string, val any) {
	switch v := val.(type) {
	case []any:
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range v {
			if s, ok := item.(string); ok {
				seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: s, Tag: "!!str"})
			}
		}
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
			seq,
		)
	case []string:
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, s := range v {
			seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: s, Tag: "!!str"})
		}
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
			seq,
		)
	case bool:
		tag := "!!bool"
		strVal := "false"
		if v {
			strVal = "true"
		}
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: strVal, Tag: tag},
		)
	default:
		setMappingScalar(mapping, key, fmt.Sprint(v))
	}
}

func mappingScalar(mapping *yaml.Node, key string) string {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1].Value
		}
	}
	return ""
}

func parseHeartbeatInterval(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("heartbeat: empty interval")
	}
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Minute, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("heartbeat: invalid interval %q: %w", s, err)
	}
	return d, nil
}

func isDreamTimeValid(s string) bool {
	for _, layout := range []string{"15:04:05", "15:04"} {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	return false
}
