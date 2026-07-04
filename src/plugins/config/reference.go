package config

import (
	"embed"
	"strings"
)

//go:embed CONFIG_REFERENCE.md
var embeddedReference embed.FS

const referenceFileName = "CONFIG_REFERENCE.md"
const toolsSectionHeading = "## Tools Configuration"

// ReferenceContent returns the embedded configuration reference markdown.
func ReferenceContent() (string, error) {
	data, err := embeddedReference.ReadFile(referenceFileName)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ExtractToolsReference returns the Tools Configuration section from the full reference doc.
func ExtractToolsReference(full string) string {
	start := strings.Index(full, toolsSectionHeading)
	if start < 0 {
		return ""
	}
	rest := full[start+len(toolsSectionHeading):]
	next := strings.Index(rest, "\n## ")
	if next < 0 {
		return strings.TrimSpace(toolsSectionHeading + rest)
	}
	return strings.TrimSpace(toolsSectionHeading + rest[:next])
}
