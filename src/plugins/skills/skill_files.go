package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const skillFileName = "SKILL.md"

var validSkillName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

func parseFrontmatterAndBody(raw []byte) (skillFrontmatter, string, error) {
	s := string(raw)
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.TrimLeft(s, " \t")
	if !strings.HasPrefix(s, "---") {
		return skillFrontmatter{}, strings.TrimSpace(s), nil
	}

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

func loadSkill(dir string) (*Skill, error) {
	skillPath := filepath.Join(dir, skillFileName)
	raw, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", skillFileName, err)
	}

	return loadSkillFromBytes(dir, raw)
}

func loadSkillFromBytes(dir string, raw []byte) (*Skill, error) {
	fm, body, err := parseFrontmatterAndBody(raw)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", skillFileName, err)
	}
	if err := validateSkillFrontmatter(fm); err != nil {
		return nil, err
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("%s must contain a markdown body", skillFileName)
	}

	return &Skill{
		Name:          fm.Name,
		Description:   fm.Description,
		Version:       fm.Version,
		Usage:         extractSkillUsage(body),
		Body:          body,
		Dir:           dir,
		SkillFile:     filepath.Join(dir, skillFileName),
		ReferencesDir: filepath.Join(dir, "references"),
	}, nil
}

func validateSkillFrontmatter(fm skillFrontmatter) error {
	name := strings.TrimSpace(fm.Name)
	if name == "" {
		return fmt.Errorf("%s frontmatter must include 'name'", skillFileName)
	}
	if len(name) > 64 {
		return fmt.Errorf("%s frontmatter 'name' must be 64 characters or fewer", skillFileName)
	}
	if !validSkillName.MatchString(name) {
		return fmt.Errorf("%s frontmatter 'name' must use only lowercase letters, numbers, and hyphens", skillFileName)
	}

	description := strings.TrimSpace(fm.Description)
	if description == "" {
		return fmt.Errorf("%s frontmatter must include 'description'", skillFileName)
	}
	if len(description) > 1024 {
		return fmt.Errorf("%s frontmatter 'description' must be 1024 characters or fewer", skillFileName)
	}
	return nil
}

func extractSkillUsage(body string) string {
	if section := findMarkdownSection(body, "when to use", "usage"); section != "" {
		return firstParagraph(section)
	}
	return firstParagraph(body)
}

func findMarkdownSection(body string, names ...string) string {
	lines := strings.Split(body, "\n")
	normalized := make(map[string]struct{}, len(names))
	for _, name := range names {
		normalized[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}

	var active bool
	var sb strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "##") {
			title := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			_, active = normalized[strings.ToLower(title)]
			if active {
				sb.Reset()
			} else if sb.Len() > 0 {
				break
			}
			continue
		}
		if !active {
			continue
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

func listReferenceFiles(skill *Skill) ([]string, error) {
	if skill == nil {
		return nil, fmt.Errorf("skill is nil")
	}
	if _, err := os.Stat(skill.ReferencesDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	_, refsReal, err := resolveReferencesRoot(skill)
	if err != nil {
		return nil, err
	}

	files := []string{}
	err = filepath.WalkDir(skill.ReferencesDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		if err := ensurePathWithinRoot(realPath, refsReal); err != nil {
			return nil
		}
		rel, err := filepath.Rel(skill.ReferencesDir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func readReferenceFile(skill *Skill, relPath string) (string, error) {
	if skill == nil {
		return "", fmt.Errorf("skill is nil")
	}
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("referencePath is required")
	}

	clean := filepath.Clean(relPath)
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("referencePath must be relative to references/")
	}
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("referencePath must stay within references/")
	}

	fullPath := filepath.Join(skill.ReferencesDir, clean)
	refsAbs, refsReal, err := resolveReferencesRoot(skill)
	if err != nil {
		return "", err
	}
	fullAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	if err := ensurePathWithinRoot(fullAbs, refsAbs); err != nil {
		return "", err
	}

	info, err := os.Stat(fullAbs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("referencePath must point to a file")
	}
	realPath, err := filepath.EvalSymlinks(fullAbs)
	if err != nil {
		return "", err
	}
	if err := ensurePathWithinRoot(realPath, refsReal); err != nil {
		return "", err
	}

	content, err := os.ReadFile(realPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func resolveReferencesRoot(skill *Skill) (string, string, error) {
	refsAbs, err := filepath.Abs(skill.ReferencesDir)
	if err != nil {
		return "", "", err
	}
	refsReal, err := filepath.EvalSymlinks(refsAbs)
	if err != nil {
		return "", "", err
	}
	skillRootAbs, err := filepath.Abs(skill.Dir)
	if err != nil {
		return "", "", err
	}
	if err := ensurePathWithinRoot(refsReal, skillRootAbs); err != nil {
		return "", "", fmt.Errorf("references/ must stay within the skill directory")
	}
	return refsAbs, refsReal, nil
}

func ensurePathWithinRoot(path, root string) error {
	if path != root && !strings.HasPrefix(path, root+string(filepath.Separator)) {
		return fmt.Errorf("referencePath must stay within references/")
	}
	return nil
}

func ensureDirWithinRoot(path, root string, label string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if pathAbs != rootAbs && !strings.HasPrefix(pathAbs, rootAbs+string(filepath.Separator)) {
		return fmt.Errorf("%s must stay within %s", label, rootAbs)
	}
	return nil
}

func copyDirNoSymlinks(srcDir, destDir string) error {
	srcAbs, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}
	return filepath.WalkDir(srcAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcAbs, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, rel)

		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("install source cannot contain symlinks: %s", filepath.ToSlash(rel))
		}
		if d.IsDir() {
			return os.MkdirAll(destPath, info.Mode().Perm())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(destPath, data, info.Mode().Perm())
	})
}
