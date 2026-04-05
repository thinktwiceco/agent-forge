package procedures

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFrontmatterAndBody(t *testing.T) {
	t.Parallel()
	raw := `---
name: my-skill
description: One line desc
---

# Body title

Intro paragraph.

## Step A
Do A.
`
	fm, body, err := parseFrontmatterAndBody([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if fm.Name != "my-skill" || fm.Description != "One line desc" {
		t.Fatalf("frontmatter: got %+v", fm)
	}
	if !strings.Contains(body, "# Body title") || !strings.Contains(body, "## Step A") {
		t.Fatalf("body: %q", body)
	}
}

func TestParseFrontmatterAndBody_noFrontmatter(t *testing.T) {
	t.Parallel()
	raw := `# Only title

No yaml.
`
	fm, body, err := parseFrontmatterAndBody([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if fm.Name != "" || fm.Description != "" {
		t.Fatalf("expected empty fm, got %+v", fm)
	}
	if !strings.Contains(body, "Only title") {
		t.Fatalf("body: %q", body)
	}
}

func TestPhaseContentsFromBody_noH2(t *testing.T) {
	t.Parallel()
	body := "# T\n\nHello world.\n\nMore."
	phases := phaseContentsFromBody(body)
	if len(phases) != 1 || phases[0] != strings.TrimSpace(body) {
		t.Fatalf("got %#v", phases)
	}
}

func TestPhaseContentsFromBody_h2Sections(t *testing.T) {
	t.Parallel()
	body := `Preamble line.

## First

Content one.

## Second

Content two.
`
	phases := phaseContentsFromBody(body)
	if len(phases) != 2 {
		t.Fatalf("want 2 phases, got %d: %#v", len(phases), phases)
	}
	if !strings.Contains(phases[0], "Preamble line") || !strings.Contains(phases[0], "## First") {
		t.Fatalf("phase 0: %q", phases[0])
	}
	if !strings.Contains(phases[1], "## Second") {
		t.Fatalf("phase 1: %q", phases[1])
	}
}

func TestPhaseContentsFromBody_preambleMergedIntoFirst(t *testing.T) {
	t.Parallel()
	body := `Intro only.

## Step

Step body.
`
	phases := phaseContentsFromBody(body)
	if len(phases) != 1 {
		t.Fatalf("got %d phases", len(phases))
	}
	if !strings.HasPrefix(phases[0], "Intro only.") {
		t.Fatalf("phase 0: %q", phases[0])
	}
}

func TestMaybeAdaptSkill_writesManifestAndPhases(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skill := `---
name: test-proc
description: Test description
---

# Ignored for name

Preamble.

## Alpha

Do alpha.

## Beta

Do beta.
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skill), 0644); err != nil {
		t.Fatal(err)
	}
	if err := maybeAdaptSkill(dir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "name: test-proc") || !strings.Contains(s, "Test description") {
		t.Fatalf("manifest:\n%s", s)
	}

	a, err := os.ReadFile(filepath.Join(dir, "0", "instructions.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(a), "Preamble") || !strings.Contains(string(a), "## Alpha") {
		t.Fatalf("0/instructions.md: %s", a)
	}
	b, err := os.ReadFile(filepath.Join(dir, "1", "instructions.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "## Beta") {
		t.Fatalf("1/instructions.md: %s", b)
	}
}

func TestLoadProcedure_skillOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skill := `---
name: load-test
description: From skill
---

## Only

Step.
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skill), 0644); err != nil {
		t.Fatal(err)
	}
	proc, err := loadProcedure(dir)
	if err != nil {
		t.Fatal(err)
	}
	if proc.Name != "load-test" || proc.PhaseCount != 1 {
		t.Fatalf("procedure: %+v", proc)
	}
}

func TestMaybeAdaptSkill_skipsWhenManifestExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte("name: keep\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: other\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := maybeAdaptSkill(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "keep") {
		t.Fatalf("manifest was overwritten: %s", data)
	}
}
