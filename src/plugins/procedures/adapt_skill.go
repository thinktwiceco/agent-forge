package procedures

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const skillFileName = "SKILL.md"
const adaptedDescriptionFallback = "Procedure adapted from SKILL.md"

var (
	reH2Line = regexp.MustCompile(`(?m)^##\s+`)
	reH1Line = regexp.MustCompile(`(?m)^#\s+(.+)$`)
)

// maybeAdaptSkill converts procedures/<slug>/SKILL.md into manifest.yaml and
// numbered phase folders when manifest.yaml is absent. If manifest.yaml exists,
// or SKILL.md is absent, it is a no-op.
func maybeAdaptSkill(dir string) error {
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if _, err := os.Stat(manifestPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	skillPath := filepath.Join(dir, skillFileName)
	raw, err := os.ReadFile(skillPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	fm, body, err := parseFrontmatterAndBody(raw)
	if err != nil {
		return fmt.Errorf("adapt SKILL.md: %w", err)
	}

	base := filepath.Base(dir)
	name := strings.TrimSpace(fm.Name)
	if name == "" {
		name = strings.TrimSpace(firstH1Title(body))
	}
	if name == "" {
		name = base
	}

	desc := strings.TrimSpace(fm.Description)
	if desc == "" {
		desc = inferDescription(body)
	}
	if desc == "" {
		desc = adaptedDescriptionFallback
	}

	phases := phaseContentsFromBody(body)
	if err := removeNumberedPhaseDirs(dir); err != nil {
		return fmt.Errorf("adapt SKILL.md: %w", err)
	}
	for i, content := range phases {
		phaseDir := filepath.Join(dir, strconv.Itoa(i))
		if err := os.MkdirAll(phaseDir, 0755); err != nil {
			return fmt.Errorf("adapt SKILL.md: mkdir %s: %w", phaseDir, err)
		}
		instPath := filepath.Join(phaseDir, "instructions.md")
		if err := os.WriteFile(instPath, []byte(strings.TrimSpace(content)+"\n"), 0644); err != nil {
			return fmt.Errorf("adapt SKILL.md: write %s: %w", instPath, err)
		}
	}

	m := manifest{Name: name, Description: desc}
	if err := writeManifestAtomic(manifestPath, m); err != nil {
		return fmt.Errorf("adapt SKILL.md: %w", err)
	}
	return nil
}

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func parseFrontmatterAndBody(raw []byte) (skillFrontmatter, string, error) {
	s := string(raw)
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.TrimLeft(s, " \t")
	if !strings.HasPrefix(s, "---") {
		return skillFrontmatter{}, strings.TrimSpace(s), nil
	}

	// First line is "---"; YAML ends at "\n---" before the body.
	openEnd := 3
	closeRel := strings.Index(s[openEnd:], "\n---")
	if closeRel == -1 {
		return skillFrontmatter{}, strings.TrimSpace(s), nil
	}
	closeAbs := openEnd + closeRel
	fmBlock := strings.TrimSpace(s[openEnd:closeAbs])
	body := strings.TrimSpace(s[closeAbs+len("\n---"):])

	var fm skillFrontmatter
	if fmBlock != "" {
		if err := yaml.Unmarshal([]byte(fmBlock), &fm); err != nil {
			return skillFrontmatter{}, "", fmt.Errorf("frontmatter: %w", err)
		}
	}
	return fm, body, nil
}

func firstH1Title(body string) string {
	m := reH1Line.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	title := strings.TrimSpace(m[1])
	if strings.HasPrefix(title, "#") {
		return ""
	}
	return title
}

func inferDescription(body string) string {
	if p := firstParagraphAfterH1(body); p != "" {
		return p
	}
	return firstParagraph(body)
}

func firstParagraphAfterH1(body string) string {
	lines := strings.Split(body, "\n")
	var afterH1 bool
	var sb strings.Builder
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if !afterH1 {
			if strings.HasPrefix(trim, "#") && !strings.HasPrefix(trim, "##") {
				afterH1 = true
			}
			continue
		}
		if trim == "" && sb.Len() > 0 {
			break
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(line)
	}
	return strings.TrimSpace(sb.String())
}

func firstParagraph(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	parts := strings.SplitN(s, "\n\n", 2)
	return strings.TrimSpace(parts[0])
}

func phaseContentsFromBody(body string) []string {
	body = strings.TrimSpace(body)
	if body == "" {
		return []string{""}
	}

	locs := reH2Line.FindAllStringIndex(body, -1)
	if len(locs) == 0 {
		return []string{body}
	}

	preamble := strings.TrimSpace(body[:locs[0][0]])
	var sections []string
	for i, loc := range locs {
		end := len(body)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		sections = append(sections, strings.TrimSpace(body[loc[0]:end]))
	}
	if preamble != "" && len(sections) > 0 {
		sections[0] = preamble + "\n\n" + sections[0]
	}
	return sections
}

func removeNumberedPhaseDirs(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, err := strconv.Atoi(name); err != nil {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

func writeManifestAtomic(path string, m manifest) error {
	data, err := yaml.Marshal(&m)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
