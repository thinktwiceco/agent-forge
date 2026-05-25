package skills

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestParseFrontmatterAndBody(t *testing.T) {
	t.Parallel()
	raw := `---
name: my-skill
description: One line desc
version: 1.2.3
---

## Overview

Intro paragraph.
`
	fm, body, err := parseFrontmatterAndBody([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if fm.Name != "my-skill" || fm.Description != "One line desc" || fm.Version != "1.2.3" {
		t.Fatalf("frontmatter: got %+v", fm)
	}
	if !strings.Contains(body, "## Overview") || !strings.Contains(body, "Intro paragraph.") {
		t.Fatalf("body: %q", body)
	}
}

func TestLoadSkill_RequiresNameAndDescription(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skill := `---
name: missing-description
---

## Overview

Hello.
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skill), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := loadSkill(dir)
	if err == nil || !strings.Contains(err.Error(), "must include 'description'") {
		t.Fatalf("expected description validation error, got %v", err)
	}
}

func TestLoadSkill_RequiresStandardName(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skill := `---
name: Not-Valid
description: Invalid name example
---

## Overview

Hello.
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skill), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := loadSkill(dir)
	if err == nil || !strings.Contains(err.Error(), "lowercase letters, numbers, and hyphens") {
		t.Fatalf("expected name validation error, got %v", err)
	}
}

func TestLoadSkill_ExtractsBodyAndUsage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillFile := `---
name: load-test
description: Load a skill body and usage summary
version: 2.0.0
---

## Overview

Use this skill to verify body parsing.

## When to Use

Use this when a task needs a focused implementation workflow.

## Process

1. Do the thing.
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillFile), 0644); err != nil {
		t.Fatal(err)
	}
	skill, err := loadSkill(dir)
	if err != nil {
		t.Fatal(err)
	}
	if skill.Name != "load-test" || skill.Version != "2.0.0" {
		t.Fatalf("skill metadata: %+v", skill)
	}
	if !strings.Contains(skill.Body, "## Process") {
		t.Fatalf("body missing content: %q", skill.Body)
	}
	if strings.Contains(skill.Body, "name: load-test") {
		t.Fatalf("body should not include frontmatter: %q", skill.Body)
	}
	if skill.Usage != "Use this when a task needs a focused implementation workflow." {
		t.Fatalf("usage: %q", skill.Usage)
	}
}

func TestListReferenceFiles_SortsRelativePaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	if err := os.MkdirAll(filepath.Join(skillDir, "references", "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestSkill(t, skillDir, "test-skill", "Test references", "")
	if err := os.WriteFile(filepath.Join(skillDir, "references", "b.md"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "references", "nested", "a.md"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	skill, err := loadSkill(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := listReferenceFiles(skill)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"b.md", "nested/a.md"}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("paths: got %v want %v", paths, want)
	}
}

func TestReadReferenceFile_RejectsTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	if err := os.MkdirAll(filepath.Join(skillDir, "references"), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestSkill(t, skillDir, "test-skill", "Test references", "")
	if err := os.WriteFile(filepath.Join(skillDir, "references", "ok.md"), []byte("safe"), 0644); err != nil {
		t.Fatal(err)
	}

	skill, err := loadSkill(skillDir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = readReferenceFile(skill, "../secret.md")
	if err == nil || !strings.Contains(err.Error(), "must stay within references/") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func TestReadReferenceFile_RejectsSymlinkEscape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	if err := os.MkdirAll(filepath.Join(skillDir, "references"), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestSkill(t, skillDir, "test-skill", "Test references", "")

	outsideFile := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(skillDir, "references", "escape.txt")
	if err := os.Symlink(outsideFile, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	skill, err := loadSkill(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := listReferenceFiles(skill)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("expected symlink escape to be hidden from listings, got %v", paths)
	}
	_, err = readReferenceFile(skill, "escape.txt")
	if err == nil || !strings.Contains(err.Error(), "must stay within references/") {
		t.Fatalf("expected symlink escape error, got %v", err)
	}
}

func TestReferenceRoot_RejectsSymlinkEscape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestSkill(t, skillDir, "test-skill", "Test references", "")

	outsideRefs := filepath.Join(dir, "outside-refs")
	if err := os.MkdirAll(outsideRefs, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsideRefs, "secret.md"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideRefs, filepath.Join(skillDir, "references")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	skill, err := loadSkill(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = listReferenceFiles(skill)
	if err == nil || !strings.Contains(err.Error(), "references/ must stay within the skill directory") {
		t.Fatalf("expected root symlink error, got %v", err)
	}
	_, err = readReferenceFile(skill, "secret.md")
	if err == nil || !strings.Contains(err.Error(), "references/ must stay within the skill directory") {
		t.Fatalf("expected root symlink error, got %v", err)
	}
}

func TestLoadSkills_DuplicateNamesFail(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	skillsDir := filepath.Join(root, "skills")
	for _, dirName := range []string{"one", "two"} {
		dir := filepath.Join(skillsDir, dirName)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		writeTestSkill(t, dir, "duplicate-skill", "Duplicate name test", "")
	}

	plugin := NewSkillsPlugin(root)
	if err := plugin.loadSkills(); err == nil || !strings.Contains(err.Error(), "duplicate skill name") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestNewSkillsPlugin_SeedsWebNavigationSkill(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	plugin := NewSkillsPlugin(root)
	if err := plugin.loadSkills(); err != nil {
		t.Fatal(err)
	}

	skill, err := plugin.getSkill("web-navigation")
	if err != nil {
		t.Fatalf("expected seeded web-navigation skill, got %v", err)
	}
	if !strings.Contains(skill.Description, "web tool") {
		t.Fatalf("unexpected description: %q", skill.Description)
	}

	refs, err := listReferenceFiles(skill)
	if err != nil {
		t.Fatal(err)
	}
	wantRefs := []string{
		"failures-and-recovery.md",
		"forms-and-auth.md",
		"navigation-basics.md",
	}
	if strings.Join(refs, ",") != strings.Join(wantRefs, ",") {
		t.Fatalf("references: got %v want %v", refs, wantRefs)
	}
}

func TestSystemPrompt_IncludesWebNavigationInstruction(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	plugin := NewSkillsPlugin(root)
	if err := plugin.loadSkills(); err != nil {
		t.Fatal(err)
	}

	prompt := plugin.SystemPrompt()
	if !strings.Contains(prompt, "web-navigation") {
		t.Fatalf("expected prompt to mention web-navigation, got %q", prompt)
	}
	if !strings.Contains(prompt, "before using the web tool") {
		t.Fatalf("expected prompt to require loading the skill before web usage, got %q", prompt)
	}
}

func TestSkillToolActions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	skillDir := filepath.Join(root, "skills", "example-skill")
	if err := os.MkdirAll(filepath.Join(skillDir, "references"), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestSkill(t, skillDir, "example-skill", "Example skill for tool testing", "Use when testing the skill tool.")
	if err := os.WriteFile(filepath.Join(skillDir, "references", "details.md"), []byte("reference body"), 0644); err != nil {
		t.Fatal(err)
	}

	plugin := NewSkillsPlugin(root)
	tool := newSkillTool(plugin)

	listResp := tool.Call(nil, map[string]any{"action": "list_skills"})
	if !listResp.Success() || !strings.Contains(listResp.Data(), "Example skill for tool testing") {
		t.Fatalf("list response: success=%v data=%q err=%q", listResp.Success(), listResp.Data(), listResp.Error())
	}

	loadResp := tool.Call(nil, map[string]any{"action": "load_skill", "name": "example-skill"})
	if !loadResp.Success() || !strings.Contains(loadResp.Data(), "This is the body.") {
		t.Fatalf("load response: success=%v data=%q err=%q", loadResp.Success(), loadResp.Data(), loadResp.Error())
	}

	refsResp := tool.Call(nil, map[string]any{"action": "list_skill_references", "name": "example-skill"})
	if !refsResp.Success() || !strings.Contains(refsResp.Data(), "details.md") {
		t.Fatalf("refs response: success=%v data=%q err=%q", refsResp.Success(), refsResp.Data(), refsResp.Error())
	}

	refResp := tool.Call(nil, map[string]any{"action": "load_skill_reference", "name": "example-skill", "referencePath": "details.md"})
	if !refResp.Success() || !strings.Contains(refResp.Data(), "reference body") {
		t.Fatalf("reference response: success=%v data=%q err=%q", refResp.Success(), refResp.Data(), refResp.Error())
	}
}

func TestSkillToolInstallAndDeleteLocalSkill(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source", "local-example")
	if err := os.MkdirAll(filepath.Join(sourceDir, "references"), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestSkill(t, sourceDir, "local-example", "Local install example", "Use for install testing.")
	if err := os.WriteFile(filepath.Join(sourceDir, "references", "notes.md"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	plugin := NewSkillsPlugin(root)
	tool := newSkillTool(plugin)

	installResp := tool.Call(nil, map[string]any{
		"action": "install_skill",
		"path":   filepath.Join(sourceDir, "SKILL.md"),
	})
	if !installResp.Success() || !strings.Contains(installResp.Data(), "Installed skill 'local-example'") {
		t.Fatalf("install response: success=%v data=%q err=%q", installResp.Success(), installResp.Data(), installResp.Error())
	}

	installedPath := filepath.Join(root, "skills", "local-example", "SKILL.md")
	if _, err := os.Stat(installedPath); err != nil {
		t.Fatalf("expected installed skill file, got %v", err)
	}

	deleteResp := tool.Call(nil, map[string]any{
		"action": "delete_skill",
		"name":   "local-example",
	})
	if !deleteResp.Success() || !strings.Contains(deleteResp.Data(), "Deleted skill 'local-example'") {
		t.Fatalf("delete response: success=%v data=%q err=%q", deleteResp.Success(), deleteResp.Data(), deleteResp.Error())
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "local-example")); !os.IsNotExist(err) {
		t.Fatalf("expected deleted skill directory, got %v", err)
	}
}

func TestSkillToolInstallSkillRejectsDuplicate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source", "dup-skill")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestSkill(t, sourceDir, "dup-skill", "Duplicate install example", "")

	plugin := NewSkillsPlugin(root)
	tool := newSkillTool(plugin)
	first := tool.Call(nil, map[string]any{"action": "install_skill", "path": sourceDir})
	if !first.Success() {
		t.Fatalf("first install failed: %s", first.Error())
	}
	second := tool.Call(nil, map[string]any{"action": "install_skill", "path": sourceDir})
	if second.Success() || !strings.Contains(second.Error(), "already installed") {
		t.Fatalf("expected duplicate install error, got success=%v err=%q", second.Success(), second.Error())
	}
}

func TestParseRemoteSkillPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    string
		ok      bool
		wantErr string
	}{
		{name: "repository folder", input: "repository/skills/example-skill", want: "example-skill", ok: true},
		{name: "repository skill file", input: "repository/skills/example-skill/SKILL.md", want: "example-skill", ok: true},
		{name: "github tree url", input: "https://github.com/thinktwiceco/agent-forge/tree/main/repository/skills/example-skill", want: "example-skill", ok: true},
		{name: "github blob url", input: "https://github.com/thinktwiceco/agent-forge/blob/main/repository/skills/example-skill/SKILL.md", want: "example-skill", ok: true},
		{name: "wrong branch", input: "https://github.com/thinktwiceco/agent-forge/tree/dev/repository/skills/example-skill", ok: false, wantErr: "main branch"},
		{name: "nested remote path", input: "repository/skills/example-skill/references/info.md", ok: true, wantErr: "must point to a skill folder or SKILL.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := parseRemoteSkillPath(tt.input)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.ok || got != tt.want {
				t.Fatalf("got slug=%q ok=%v want slug=%q ok=%v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestSkillToolListInstallable(t *testing.T) {
	t.Parallel()
	mockGithubGet(t, func(path string, out any) error {
		switch target := out.(type) {
		case *[]ghContentsEntry:
			*target = []ghContentsEntry{{Name: "remote-skill", Type: "dir"}}
			return nil
		case *ghContentsEntry:
			if !strings.Contains(path, "remote-skill/SKILL.md") {
				return fmt.Errorf("unexpected path %s", path)
			}
			*target = ghContentsEntry{
				Content:  encodeBase64(testSkillDoc("remote-skill", "Remote skill description", "Use remotely.")),
				Encoding: "base64",
			}
			return nil
		default:
			return fmt.Errorf("unexpected output type %T", out)
		}
	})
	plugin := NewSkillsPlugin(t.TempDir())
	resp := newSkillTool(plugin).Call(nil, map[string]any{"action": "list_installable"})
	if !resp.Success() || !strings.Contains(resp.Data(), "remote-skill") || !strings.Contains(resp.Data(), "Remote skill description") {
		t.Fatalf("list_installable response: success=%v data=%q err=%q", resp.Success(), resp.Data(), resp.Error())
	}
}

func TestSkillToolInstallRemoteSkill(t *testing.T) {
	t.Parallel()
	mockGithubGet(t, func(path string, out any) error {
		switch target := out.(type) {
		case *ghTreeResponse:
			*target = ghTreeResponse{
				Tree: []ghTreeEntry{
					{Path: "repository/skills/remote-skill/SKILL.md", Type: "blob", SHA: "skill-md"},
					{Path: "repository/skills/remote-skill/references/info.md", Type: "blob", SHA: "info-md"},
				},
			}
			return nil
		case *ghBlobResponse:
			switch {
			case strings.HasSuffix(path, "/skill-md"):
				*target = ghBlobResponse{
					Content:  encodeBase64(testSkillDoc("remote-skill", "Installed from remote", "Use remote install.")),
					Encoding: "base64",
				}
				return nil
			case strings.HasSuffix(path, "/info-md"):
				*target = ghBlobResponse{
					Content:  encodeBase64("remote reference"),
					Encoding: "base64",
				}
				return nil
			default:
				return fmt.Errorf("unexpected blob path %s", path)
			}
		default:
			return fmt.Errorf("unexpected output type %T", out)
		}
	})
	root := t.TempDir()
	plugin := NewSkillsPlugin(root)
	resp := newSkillTool(plugin).Call(nil, map[string]any{
		"action": "install_skill",
		"path":   "repository/skills/remote-skill",
	})
	if !resp.Success() || !strings.Contains(resp.Data(), "Installed skill 'remote-skill'") {
		t.Fatalf("remote install response: success=%v data=%q err=%q", resp.Success(), resp.Data(), resp.Error())
	}

	refResp := newSkillTool(plugin).Call(nil, map[string]any{
		"action":        "load_skill_reference",
		"name":          "remote-skill",
		"referencePath": "references/info.md",
	})
	if refResp.Success() {
		t.Fatalf("expected nested references path to be relative to references/, got %q", refResp.Data())
	}

	refsResp := newSkillTool(plugin).Call(nil, map[string]any{
		"action": "list_skill_references",
		"name":   "remote-skill",
	})
	if !refsResp.Success() || !strings.Contains(refsResp.Data(), "info.md") {
		t.Fatalf("refs response: success=%v data=%q err=%q", refsResp.Success(), refsResp.Data(), refsResp.Error())
	}
}

func writeTestSkill(t *testing.T, dir, name, description, usage string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(testSkillDoc(name, description, usage)), 0644); err != nil {
		t.Fatal(err)
	}
}

func testSkillDoc(name, description, usage string) string {
	body := `---
name: ` + name + `
description: ` + description + `
---

## Overview

This is the body.
`
	if usage != "" {
		body += `
## When to Use

` + usage + `
`
	}
	return body
}

func encodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

var githubGetMu sync.Mutex

func mockGithubGet(t *testing.T, fn githubGetFunc) {
	t.Helper()
	githubGetMu.Lock()
	previous := githubGet
	githubGet = fn
	t.Cleanup(func() {
		githubGet = previous
		githubGetMu.Unlock()
	})
}
