package procedures

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
	"gopkg.in/yaml.v3"
)

//go:embed default learn-procedure fill-form
var defaultProcedures embed.FS

const (
	defaultProcedureDir  = "create-procedure"
	learnProcedureDir    = "learn-procedure"
	fillFormProcedureDir = "fill-form"
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
	Dir         string // directory containing this procedure's phase folders
}

// ProceduresPlugin discovers procedures from a directory and provides a
// tool that lets the agent walk through them phase by phase.
type ProceduresPlugin struct {
	// dir is the directory for user-created procedures (workingDir/procedures).
	dir string
	// repositoryDir is the directory for remotely installed procedures (workingDir/repository/procedures).
	repositoryDir   string
	procedures      map[string]*Procedure
	activeProcedure *Procedure
	currentPhase    int
}

// NewProceduresPlugin creates a new ProceduresPlugin.
// User procedures live in workingDir/procedures.
// Repository-installed procedures live in workingDir/repository/procedures.
func NewProceduresPlugin(workingDir string) *ProceduresPlugin {
	dir := filepath.Join(workingDir, "procedures")
	repositoryDir := filepath.Join(workingDir, "repository", "procedures")
	_ = os.MkdirAll(dir, 0755)
	_ = os.MkdirAll(repositoryDir, 0755)
	p := &ProceduresPlugin{
		dir:           dir,
		repositoryDir: repositoryDir,
		procedures:    make(map[string]*Procedure),
	}
	p.ensureDefaultProcedures()
	return p
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
	sb.WriteString("[PROCEDURES]\n")
	sb.WriteString("- Tool: procedure\n")
	sb.WriteString("- Structured multi-step tasks. Actions: start_procedure, next_step, goto_step (jump to step by number).\n")
	sb.WriteString("- User-created procedures live in procedures/. Repository-installed procedures live in repository/procedures/. When creating procedures, always use paths under procedures/ (e.g. procedures/my-procedure/).\n\n")
	sb.WriteString("[PROCEDURE EXECUTION RULE — MANDATORY]\n")
	sb.WriteString("At ANY step or tool call, if the outcome is not exactly what the step describes, or a Tool returne any error or unexpected result as expected:\n")
	sb.WriteString("1. STOP immediately. Do not continue, retry, guess, or attempt to work around the problem.\n")
	sb.WriteString("2. Report to the user with:\n")
	sb.WriteString("   - Step: which step you were on\n")
	sb.WriteString("   - Expected: what you expected to happen or find\n")
	sb.WriteString("   - Found: what actually happened or was present\n")
	sb.WriteString("   - Evidence: any tool output, error message, or file path relevant to the failure\n")
	sb.WriteString("This rule is absolute and overrides any instinct to recover autonomously.\n\n")
	sb.WriteString("[AVAILABLE]\n")

	// Sort names for deterministic output
	names := make([]string, 0, len(p.procedures))
	for name := range p.procedures {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		proc := p.procedures[name]
		fmt.Fprintf(&sb, "- %s (%d phases): %s\n", proc.Name, proc.PhaseCount, proc.Description)
	}

	return sb.String()
}

// Tools implements core.ToolProvider.
func (p *ProceduresPlugin) Tools() []llms.Tool {
	return []llms.Tool{
		newProcedureTool(p),
	}
}

// loadProcedures scans both the user procedures dir and the repository dir,
// then merges the results (repository entries overwrite on name collision).
func (p *ProceduresPlugin) loadProcedures() error {
	p.procedures = make(map[string]*Procedure)
	for _, dir := range []string{p.dir, p.repositoryDir} {
		if err := p.loadProceduresFromDir(dir); err != nil {
			return err
		}
	}
	return nil
}

// loadProceduresFromDir scans a single directory for procedure subfolders.
func (p *ProceduresPlugin) loadProceduresFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			agentforge.Info("procedures plugin: directory '%s' not found, skipping", dir)
			return nil
		}
		return fmt.Errorf("procedures plugin: failed to read directory '%s': %w", dir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		procDir := filepath.Join(dir, entry.Name())
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
		Dir:         dir,
	}, nil
}

// ensureDefaultProcedures copies embedded default procedures (create-procedure, learn-procedure)
// to the procedures directory when they do not exist.
func (p *ProceduresPlugin) ensureDefaultProcedures() {
	p.ensureProcedure("default", defaultProcedureDir)
	p.ensureProcedure("learn-procedure", learnProcedureDir)
	p.ensureProcedure("fill-form", fillFormProcedureDir)
}

func (p *ProceduresPlugin) ensureProcedure(embedDir, destName string) {
	destDir := filepath.Join(p.dir, destName)
	if _, err := os.Stat(destDir); err == nil {
		return // procedure already exists
	}

	fsys, err := fs.Sub(defaultProcedures, embedDir)
	if err != nil {
		agentforge.Info("procedures plugin: failed to open %s: %v", embedDir, err)
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
		agentforge.Info("procedures plugin: failed to copy %s: %v", embedDir, err)
		return
	}
	agentforge.Info("procedures plugin: copied %s to %s", embedDir, destDir)
}

func init() {
	registry.Register(PLUGIN_NAME, func(workingDir string) core.Plugin {
		return NewProceduresPlugin(workingDir)
	})
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
	phaseDir := filepath.Join(proc.Dir, strconv.Itoa(phase))

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

		fmt.Fprintf(&sb, "=== %s ===\n", entry.Name())
		sb.Write(content)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
