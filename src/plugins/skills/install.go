package skills

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const remoteSkillsRoot = "repository/skills"

func (p *SkillsPlugin) installSkill(path string) (*Skill, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("path parameter is required for install_skill")
	}

	if remoteSlug, ok, err := parseRemoteSkillPath(path); err != nil {
		return nil, err
	} else if ok {
		skill, err := installRemoteSkill(remoteSlug, p.dir)
		if err != nil {
			return nil, err
		}
		if err := p.loadSkills(); err != nil {
			return nil, err
		}
		return skill, nil
	}

	sourceDir, err := resolveLocalSkillDir(path)
	if err != nil {
		return nil, err
	}
	skill, err := installLocalSkillFromDir(sourceDir, p.dir)
	if err != nil {
		return nil, err
	}
	if err := p.loadSkills(); err != nil {
		return nil, err
	}
	return skill, nil
}

func (p *SkillsPlugin) deleteSkill(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name parameter is required for delete_skill")
	}
	if !validSkillName.MatchString(name) {
		return fmt.Errorf("skill name must use only lowercase letters, numbers, and hyphens")
	}

	destDir := filepath.Join(p.dir, name)
	if err := ensureDirWithinRoot(destDir, p.dir, "skill directory"); err != nil {
		return err
	}
	if _, err := os.Stat(destDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("skill '%s' is not installed", name)
		}
		return err
	}

	if err := os.RemoveAll(destDir); err != nil {
		return err
	}
	return p.loadSkills()
}

func resolveLocalSkillDir(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return absPath, nil
	}
	if info.Name() != skillFileName {
		return "", fmt.Errorf("local install path must point to a skill directory or %s", skillFileName)
	}
	return filepath.Dir(absPath), nil
}

func installLocalSkillFromDir(sourceDir, destRoot string) (*Skill, error) {
	skill, err := loadSkill(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("invalid skill at '%s': %w", sourceDir, err)
	}

	destDir := filepath.Join(destRoot, skill.Name)
	if err := ensureDirWithinRoot(destDir, destRoot, "skill destination"); err != nil {
		return nil, err
	}
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return nil, err
	}
	sourceAbs, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, err
	}
	if destAbs == sourceAbs {
		return nil, fmt.Errorf("skill '%s' is already installed at %s", skill.Name, destDir)
	}
	if _, err := os.Stat(destDir); err == nil {
		return nil, fmt.Errorf("skill '%s' is already installed", skill.Name)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := copyDirNoSymlinks(sourceDir, destDir); err != nil {
		_ = os.RemoveAll(destDir)
		return nil, err
	}
	return loadSkill(destDir)
}

func parseRemoteSkillPath(path string) (string, bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false, nil
	}

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		u, err := url.Parse(path)
		if err != nil {
			return "", false, err
		}
		if u.Host != "github.com" {
			return "", false, nil
		}
		trimmed := strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")
		parts := strings.Split(trimmed, "/")
		if len(parts) < 6 || parts[0] != "thinktwiceco" || parts[1] != "agent-forge" {
			return "", false, nil
		}
		if parts[2] != "tree" && parts[2] != "blob" {
			return "", false, nil
		}
		if parts[3] != githubBranch {
			return "", false, fmt.Errorf("remote GitHub skill paths must target the %s branch", githubBranch)
		}
		return extractRemoteSkillSlug(strings.Join(parts[4:], "/"))
	}

	return extractRemoteSkillSlug(filepath.ToSlash(path))
}

func extractRemoteSkillSlug(path string) (string, bool, error) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return "", false, nil
	}
	if !strings.HasPrefix(path, remoteSkillsRoot) {
		return "", false, nil
	}

	remainder := strings.Trim(strings.TrimPrefix(path, remoteSkillsRoot), "/")
	if remainder == "" {
		return "", true, fmt.Errorf("remote skill path must include a skill folder under %s", remoteSkillsRoot)
	}
	parts := strings.Split(remainder, "/")
	slug := parts[0]
	if !validSkillName.MatchString(slug) {
		return "", true, fmt.Errorf("remote skill path must include a valid skill folder name")
	}
	if len(parts) > 1 {
		last := parts[len(parts)-1]
		if len(parts) != 2 || last != skillFileName {
			return "", true, fmt.Errorf("remote skill path must point to a skill folder or %s", skillFileName)
		}
	}
	return slug, true, nil
}
