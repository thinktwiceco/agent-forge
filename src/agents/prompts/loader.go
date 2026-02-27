package prompts

import (
	"embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed files/main/*.md files/system-agents/*.md
var promptsFS embed.FS

// LoadMainPrompt reads a main prompt file and returns its content.
// Name is the filename without .md (e.g., "default", "main-agent", "tone-keep-it-short").
func LoadMainPrompt(name string) (string, error) {
	path := "files/main/" + name + ".md"
	data, err := promptsFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("load main prompt %s: %w", name, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// SystemAgentContent holds parsed sections from a system agent markdown file.
type SystemAgentContent struct {
	Incipit             string
	Steps               []string
	Output              string
	Examples            []string
	Critical            []string
	DescriptionIncipit  string
	DescriptionExamples []string
	AdvanceDescription  string
	Troubleshooting     string
}

// sectionHeader matches "## SectionName" at start of line.
var sectionHeaderRE = regexp.MustCompile(`(?m)^##\s+(\w+)\s*$`)

// LoadSystemAgent reads and parses a system agent markdown file into structured content.
// Name is the filename without .md (e.g., "reasoning", "vector", "os").
func LoadSystemAgent(name string) (*SystemAgentContent, error) {
	path := "files/system-agents/" + name + ".md"
	data, err := promptsFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load system agent %s: %w", name, err)
	}
	return parseSystemAgentMarkdown(string(data))
}

func parseSystemAgentMarkdown(content string) (*SystemAgentContent, error) {
	result := &SystemAgentContent{}
	matches := sectionHeaderRE.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no ## sections found in markdown")
	}

	sections := make(map[string]string)
	for i, m := range matches {
		sectionName := strings.TrimSpace(content[m[2]:m[3]])
		start := m[1]
		var end int
		if i+1 < len(matches) {
			end = matches[i+1][0]
		} else {
			end = len(content)
		}
		body := strings.TrimSpace(content[start:end])
		// Remove the "## SectionName" line from body
		if idx := strings.Index(body, "\n"); idx >= 0 {
			body = strings.TrimSpace(body[idx+1:])
		} else {
			body = ""
		}
		sections[sectionName] = body
	}

	result.Incipit = sections["Incipit"]
	result.Output = sections["Output"]
	result.AdvanceDescription = sections["AdvanceDescription"]
	result.Troubleshooting = sections["Troubleshooting"]

	// Parse Description: split by [EXAMPLES] for incipit and examples
	if desc := sections["Description"]; desc != "" {
		parts := strings.SplitN(desc, "[EXAMPLES]", 2)
		result.DescriptionIncipit = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			lines := strings.Split(strings.TrimSpace(parts[1]), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					result.DescriptionExamples = append(result.DescriptionExamples, line)
				}
			}
		}
	}

	// Parse Steps: "- Step N: text" -> extract "text"
	if s := sections["Steps"]; s != "" {
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "- ") {
				rest := strings.TrimPrefix(line, "- ")
				// Match "Step N: " and extract the step text
				if idx := strings.Index(rest, ": "); idx >= 0 {
					rest = rest[idx+2:]
				}
				result.Steps = append(result.Steps, rest)
			}
		}
	}

	// Parse Critical: "- rule" -> extract "rule"
	if s := sections["Critical"]; s != "" {
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "- ") {
				result.Critical = append(result.Critical, strings.TrimPrefix(line, "- "))
			}
		}
	}

	// Parse Examples: blocks separated by "---"
	if s := sections["Examples"]; s != "" {
		blocks := strings.Split(s, "\n---\n")
		for _, b := range blocks {
			b = strings.TrimSpace(b)
			b = strings.TrimPrefix(b, "---")
			b = strings.TrimSpace(b)
			if b != "" {
				result.Examples = append(result.Examples, b+"\n")
			}
		}
		// If no --- separators, treat whole section as one example
		if len(result.Examples) == 0 && s != "" {
			result.Examples = append(result.Examples, strings.TrimSpace(s)+"\n")
		}
	}

	return result, nil
}
