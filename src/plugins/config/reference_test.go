package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func staticReferencePath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "cmd", "localforge", "src", "static", "docs", "config-reference.md"))
}

func TestReferenceContentNotEmpty(t *testing.T) {
	content, err := ReferenceContent()
	if err != nil {
		t.Fatalf("ReferenceContent: %v", err)
	}
	if !strings.Contains(content, toolsSectionHeading) {
		t.Fatalf("reference missing %q section", toolsSectionHeading)
	}
}

func TestExtractToolsReference(t *testing.T) {
	content, err := ReferenceContent()
	if err != nil {
		t.Fatalf("ReferenceContent: %v", err)
	}
	section := ExtractToolsReference(content)
	if section == "" {
		t.Fatal("ExtractToolsReference returned empty")
	}
	if strings.Contains(section, "## Plugins Configuration") {
		t.Fatal("tools section should not include plugins section")
	}
	if !strings.Contains(section, toolsSectionHeading) {
		t.Fatalf("section should start with %q", toolsSectionHeading)
	}
}

func TestConfigReferenceSyncedWithStatic(t *testing.T) {
	embedded, err := ReferenceContent()
	if err != nil {
		t.Fatalf("ReferenceContent: %v", err)
	}
	staticPath := staticReferencePath()
	if staticPath == "" {
		t.Fatal("could not resolve static reference path")
	}
	staticBytes, err := os.ReadFile(staticPath)
	if err != nil {
		t.Fatalf("read static reference %q: %v", staticPath, err)
	}
	if string(staticBytes) != embedded {
		t.Fatalf("static copy out of sync with %s; run: go generate ./src/plugins/config/", referenceFileName)
	}
}
