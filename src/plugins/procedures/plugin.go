package procedures

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"gopkg.in/yaml.v3"
)

const PLUGIN_NAME = "procedures"

type manifest struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Procedure holds the loaded metadata and phase count for one procedure.
type Procedure struct {
	Name        string
	Description string
	PhaseCount  int
	BaseDir     string // absolute/relative path to the procedure folder
}

// ProceduresPlugin discovers procedures from a base directory and provides a
// tool that lets the agent walk through them phase by phase.
type ProceduresPlugin struct {
	baseDir         string
	procedures      map[string]*Procedure
	activeProcedure *Procedure
	currentPhase    int
}

// NewProceduresPlugin creates a new ProceduresPlugin.
// If baseDir is empty it defaults to "procedures/".
func NewProceduresPlugin(baseDir string) *ProceduresPlugin {
	if baseDir == "" {
		baseDir = "procedures"
	}
	return &ProceduresPlugin{
		baseDir:    baseDir,
		procedures: make(map[string]*Procedure),
	}
}

// Name implements core.Plugin.
func (p *ProceduresPlugin) Name() string {
	return PLUGIN_NAME
}

// Hooks implements core.HookProvider.
// Scans the procedures directory on EventAgentInitialized, then injects the
// system prompt contribution now that procedures are known.
func (p *ProceduresPlugin) Hooks() map[core.Event]core.AgentHookFn {
	return map[core.Event]core.AgentHookFn{
		core.EventAgentInitialized: agents.OnAgentInitializedHook(func(a *agents.Agent) error {
			if err := p.loadProcedures(); err != nil {
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
// Returns a prompt section listing all discovered procedures.
func (p *ProceduresPlugin) SystemPrompt() string {
	if len(p.procedures) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Available Procedures\n\n")
	sb.WriteString("You have access to structured procedures that guide you through multi-step tasks.\n")
	sb.WriteString("Use the `procedure` tool to execute them.\n\n")
	sb.WriteString("Procedures:\n")

	// Sort names for deterministic output
	names := make([]string, 0, len(p.procedures))
	for name := range p.procedures {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		proc := p.procedures[name]
		sb.WriteString(fmt.Sprintf("- **%s** (%d phases): %s\n", proc.Name, proc.PhaseCount, proc.Description))
	}

	return sb.String()
}

// Tools implements core.ToolProvider.
func (p *ProceduresPlugin) Tools() []llms.Tool {
	return []llms.Tool{
		newProcedureTool(p),
	}
}

// loadProcedures scans baseDir for procedure subfolders and parses their manifests.
func (p *ProceduresPlugin) loadProcedures() error {
	entries, err := os.ReadDir(p.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			agentforge.Info("procedures plugin: directory '%s' not found, no procedures loaded", p.baseDir)
			return nil
		}
		return fmt.Errorf("procedures plugin: failed to read directory '%s': %w", p.baseDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		procDir := filepath.Join(p.baseDir, entry.Name())
		proc, err := loadProcedure(procDir)
		if err != nil {
			agentforge.Info("procedures plugin: skipping '%s': %v", procDir, err)
			continue
		}

		p.procedures[proc.Name] = proc
		agentforge.Info("procedures plugin: loaded procedure '%s' with %d phases", proc.Name, proc.PhaseCount)
	}

	return nil
}

// loadProcedure parses a single procedure directory.
func loadProcedure(dir string) (*Procedure, error) {
	manifestPath := filepath.Join(dir, "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("missing manifest.yaml: %w", err)
	}

	var m manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest.yaml: %w", err)
	}
	if m.Name == "" {
		return nil, fmt.Errorf("manifest.yaml must have a non-empty 'name'")
	}

	phaseCount := countPhaseFolders(dir)
	if phaseCount == 0 {
		return nil, fmt.Errorf("procedure has no phase folders (0/, 1/, …)")
	}

	return &Procedure{
		Name:        m.Name,
		Description: m.Description,
		PhaseCount:  phaseCount,
		BaseDir:     dir,
	}, nil
}

// countPhaseFolders counts how many sequentially numbered folders exist (0, 1, 2, …).
func countPhaseFolders(dir string) int {
	count := 0
	for {
		phaseDir := filepath.Join(dir, strconv.Itoa(count))
		info, err := os.Stat(phaseDir)
		if err != nil || !info.IsDir() {
			break
		}
		count++
	}
	return count
}

// loadPhaseContent reads all files in the numbered phase folder and returns
// their concatenated content.
func (p *ProceduresPlugin) loadPhaseContent(proc *Procedure, phase int) (string, error) {
	phaseDir := filepath.Join(proc.BaseDir, strconv.Itoa(phase))

	entries, err := os.ReadDir(phaseDir)
	if err != nil {
		return "", fmt.Errorf("cannot read phase folder '%s': %w", phaseDir, err)
	}

	var sb strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(phaseDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("cannot read file '%s': %w", filePath, err)
		}

		sb.WriteString(fmt.Sprintf("=== %s ===\n", entry.Name()))
		sb.Write(content)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
