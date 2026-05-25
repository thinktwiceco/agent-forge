package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
)

//go:embed seeds
var defaultSkills embed.FS

// defaultSkillSeeds lists embedded skill trees (each a subfolder under seeds/)
// copied into workingDir/skills/<destName> when that folder does not yet exist.
var defaultSkillSeeds = []struct {
	embedDir string
	destName string
}{
	{
		embedDir: "seeds/web-navigation",
		destName: "web-navigation",
	},
}

const PLUGIN_NAME = "skills"

// Skill holds the metadata and on-demand content for one local skill package.
type Skill struct {
	Name          string
	Description   string
	Version       string
	Usage         string
	Body          string
	Dir           string
	SkillFile     string
	ReferencesDir string
}

// SkillsPlugin discovers skill packages from skills/ and provides a tool that
// lets the agent frontload metadata, load full skill bodies, mutate installed
// skills, and inspect references on demand.
type SkillsPlugin struct {
	dir    string
	mu     sync.RWMutex
	skills map[string]*Skill
}

// NewSkillsPlugin creates a new SkillsPlugin.
func NewSkillsPlugin(workingDir string) *SkillsPlugin {
	dir := filepath.Join(workingDir, "skills")
	_ = os.MkdirAll(dir, 0755)
	p := &SkillsPlugin{
		dir:    dir,
		skills: make(map[string]*Skill),
	}
	p.ensureDefaultSkills()
	return p
}

// Name implements core.Plugin.
func (p *SkillsPlugin) Name() string {
	return PLUGIN_NAME
}

// Hooks implements core.HookProvider.
// Scans the skills directory on EventAgentInitialized, then injects the system
// prompt contribution now that available skills are known.
func (p *SkillsPlugin) Hooks() map[core.Event]core.AgentHookFn {
	return map[core.Event]core.AgentHookFn{
		core.EventAgentInitialized: agents.OnAgentInitializedHook(func(a *agents.Agent) error {
			if err := p.loadSkills(); err != nil {
				return err
			}
			if sp := p.SystemPrompt(); sp != "" {
				a.AppendSystemPrompt(fmt.Sprintf("[PLUGIN TOOLS INSTRUCTIONS]\n [%s plugin]:\n%s\n\n", p.Name(), sp))
			}
			return nil
		}),
	}
}

// SystemPrompt implements core.PromptProvider.
// Returns a prompt section listing all discovered skills.
func (p *SkillsPlugin) SystemPrompt() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("[SKILLS]\n")
	sb.WriteString("- Tool: skill\n")
	sb.WriteString("- Use list_skills to frontload installed skill metadata before loading full instructions.\n")
	sb.WriteString("- Load the most relevant skill before using its domain tool instead of improvising from scratch.\n")
	sb.WriteString("- If a task requires web navigation, browser interaction, login flows, uploads, or page inspection, always load_skill for web-navigation before using the web tool.\n")
	sb.WriteString("- Use list_installable to discover remote skills under repository/skills on the main branch.\n")
	sb.WriteString("- Use install_skill to install a skill from a local path or a repository/skills GitHub path.\n")
	sb.WriteString("- Use delete_skill to remove an installed skill from skills/<name>.\n")
	sb.WriteString("- Use list_skill_references and load_skill_reference for deeper reference material on demand.\n")
	sb.WriteString("- Local skill packages live under skills/<slug>/ with SKILL.md and optional references/.\n\n")
	sb.WriteString("[AVAILABLE]\n")

	names := make([]string, 0, len(p.skills))
	for name := range p.skills {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		skill := p.skills[name]
		fmt.Fprintf(&sb, "- %s: %s\n", skill.Name, skill.Description)
	}

	return sb.String()
}

// Tools implements core.ToolProvider.
func (p *SkillsPlugin) Tools() []llms.Tool {
	return []llms.Tool{
		newSkillTool(p),
	}
}

// loadSkills scans skills/ for SKILL.md-based skill packages.
func (p *SkillsPlugin) loadSkills() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.skills = make(map[string]*Skill)

	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			agentforge.Info("skills plugin: directory '%s' not found, skipping", p.dir)
			return nil
		}
		return fmt.Errorf("skills plugin: failed to read directory '%s': %w", p.dir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(p.dir, entry.Name())
		skill, err := loadSkill(skillDir)
		if err != nil {
			agentforge.Info("skills plugin: skipping '%s': %v", skillDir, err)
			continue
		}
		if _, exists := p.skills[skill.Name]; exists {
			return fmt.Errorf("skills plugin: duplicate skill name '%s'", skill.Name)
		}

		p.skills[skill.Name] = skill
		agentforge.Info("skills plugin: loaded skill '%s'", skill.Name)
	}

	return nil
}

func (p *SkillsPlugin) getSkill(name string) (*Skill, error) {
	if err := p.loadSkills(); err != nil {
		return nil, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	skill, exists := p.skills[name]
	if !exists {
		return nil, fmt.Errorf("skill '%s' not found", name)
	}
	return skill, nil
}

// ensureDefaultSkills copies embedded default skills to the skills directory
// when the destination folder does not exist.
func (p *SkillsPlugin) ensureDefaultSkills() {
	for _, c := range defaultSkillSeeds {
		p.ensureSkillSeed(c.embedDir, c.destName)
	}
}

func (p *SkillsPlugin) ensureSkillSeed(embedDir, destName string) {
	destDir := filepath.Join(p.dir, destName)
	if _, err := os.Stat(destDir); err == nil {
		return
	}

	fsys, err := fs.Sub(defaultSkills, embedDir)
	if err != nil {
		agentforge.Info("skills plugin: failed to open %s: %v", embedDir, err)
		return
	}

	if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, path)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		_ = os.MkdirAll(filepath.Dir(destPath), 0755)
		return os.WriteFile(destPath, data, 0644)
	}); err != nil {
		agentforge.Info("skills plugin: failed to copy %s: %v", embedDir, err)
		return
	}
	agentforge.Info("skills plugin: copied %s to %s", embedDir, destDir)
}

func init() {
	registry.Register(PLUGIN_NAME, func(workingDir string) core.Plugin {
		return NewSkillsPlugin(workingDir)
	})
}
