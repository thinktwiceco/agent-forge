package skills

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const SKILL_TOOL = "skill"

func newSkillTool(plugin *SkillsPlugin) llms.Tool {
	detailsAbout := func(item string) string {
		switch item {
		case "list_skills":
			return `list_skills: Return frontloaded installed skill metadata only.
- No additional parameters required.
- Returns each installed skill's name, description, optional version, and a short usage summary.`
		case "load_skill":
			return `load_skill: Return the full markdown body of one installed skill.
- Required: name (string) - exact skill name from list_skills.
- Returns the SKILL.md body without YAML frontmatter.`
		case "list_skill_references":
			return `list_skill_references: List available reference files for one installed skill.
- Required: name (string) - exact skill name from list_skills.
- Returns relative file paths under references/.`
		case "load_skill_reference":
			return `load_skill_reference: Read one reference file for an installed skill.
- Required: name (string) - exact skill name from list_skills.
- Required: referencePath (string) - relative path under references/.
- Path traversal and absolute paths are rejected.`
		case "list_installable":
			return `list_installable: List remote skills available under repository/skills on the main branch.
- No additional parameters required.
- Returns remote skill metadata parsed from each remote SKILL.md.`
		case "install_skill":
			return `install_skill: Install a skill into skills/<name>.
- Required: path (string).
- Local paths may point to a skill folder or SKILL.md.
- Remote paths may point to repository/skills/<slug> or the matching GitHub URL on the main branch.`
		case "delete_skill":
			return `delete_skill: Remove one installed skill from skills/<name>.
- Required: name (string) - exact installed skill name to delete.`
		default:
			return fmt.Sprintf("Nothing to add about %s", item)
		}
	}

	return core.NewTool(core.ToolConfig{
		Name:        SKILL_TOOL,
		Description: "Discover installed SKILL.md packages, load their content on demand, list remote installable skills, install new skills, delete installed skills, and read optional reference files.",
		AdvanceDesc: `[ACTIONS]
- list_skills: Return frontloaded installed skill metadata only
- load_skill: Return the full markdown body for one installed skill. Required: name
- list_skill_references: List files under references/ for one installed skill. Required: name
- load_skill_reference: Return one references/ file. Required: name, referencePath
- list_installable: Return remote skill metadata from repository/skills on the main branch
- install_skill: Install one skill into skills/<name>. Required: path
- delete_skill: Delete one installed skill from skills/<name>. Required: name

[SKILL PACKAGES]
- Installed skills are discovered from skills/<slug>/SKILL.md.
- SKILL.md frontmatter must include:
  * name: lowercase letters, numbers, and hyphens only, max 64 chars
  * description: activation guidance and purpose, max 1024 chars
  * version: optional
- Optional deeper material can live under skills/<slug>/references/.

[INSTALL PATHS]
- install_skill accepts a local folder path, a local SKILL.md path, or a repository/skills path on GitHub main.
- Installed skills are copied into skills/<skill-name>.
- delete_skill removes the installed skill directory after validating the requested skill name.`,
		DetailsAboutFunc: detailsAbout,
		TroubleshootingInfo: `Troubleshooting:
- Ensure 'action' is one of: 'list_skills', 'load_skill', 'list_skill_references', 'load_skill_reference', 'list_installable', 'install_skill', or 'delete_skill'.
- 'name' is required for load_skill, list_skill_references, load_skill_reference, and delete_skill.
- 'path' is required for install_skill and must resolve to a local skill or repository/skills path.
- 'referencePath' is required for load_skill_reference and must be a relative path under references/.
- Skill names are matched exactly.
- Invalid SKILL.md frontmatter causes the skill to be skipped during discovery and rejected during install.
- If a remote skill is missing, list_installable first to confirm the available slugs.`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action: 'list_skills', 'load_skill', 'list_skill_references', 'load_skill_reference', 'list_installable', 'install_skill', or 'delete_skill'",
				Required:    true,
			},
			{
				Name:        "name",
				Type:        "string",
				Description: "The exact installed skill name (required for 'load_skill', 'list_skill_references', 'load_skill_reference', and 'delete_skill')",
				Required:    false,
			},
			{
				Name:        "path",
				Type:        "string",
				Description: "Local skill path or repository/skills GitHub path (required for 'install_skill')",
				Required:    false,
			},
			{
				Name:        "referencePath",
				Type:        "string",
				Description: "Reference file path relative to references/ (required for 'load_skill_reference')",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			action, ok := args["action"].(string)
			if !ok {
				return core.NewErrorResponse("action parameter is required and must be a string")
			}

			switch action {
			case "list_skills":
				return plugin.handleListSkills()
			case "load_skill":
				return plugin.handleLoadSkill(args)
			case "list_skill_references":
				return plugin.handleListSkillReferences(args)
			case "load_skill_reference":
				return plugin.handleLoadSkillReference(args)
			case "list_installable":
				return plugin.handleListInstallable()
			case "install_skill":
				return plugin.handleInstallSkill(args)
			case "delete_skill":
				return plugin.handleDeleteSkill(args)
			default:
				return core.NewErrorResponse(fmt.Sprintf(
					"unknown action '%s'. Valid actions are: 'list_skills', 'load_skill', 'list_skill_references', 'load_skill_reference', 'list_installable', 'install_skill', 'delete_skill'", action,
				))
			}
		},
	})
}

func (p *SkillsPlugin) handleListSkills() llms.ToolReturn {
	if err := p.loadSkills(); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to load skills: %v", err))
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.skills) == 0 {
		return core.NewEphemeralResponse("No skills found under skills/.")
	}

	names := make([]string, 0, len(p.skills))
	for name := range p.skills {
		names = append(names, name)
	}
	sort.Strings(names)

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d installed skill(s) under skills/:\n\n", len(names))
	for _, name := range names {
		skill := p.skills[name]
		fmt.Fprintf(&sb, "- %s\n", skill.Name)
		fmt.Fprintf(&sb, "  Description: %s\n", skill.Description)
		if skill.Version != "" {
			fmt.Fprintf(&sb, "  Version: %s\n", skill.Version)
		}
		if skill.Usage != "" {
			fmt.Fprintf(&sb, "  Usage: %s\n", skill.Usage)
		}
	}
	sb.WriteString("\nUse load_skill to read the full body, list_skill_references to inspect references/, or delete_skill to remove one.")
	return core.NewEphemeralResponse(sb.String())
}

func (p *SkillsPlugin) handleLoadSkill(args map[string]any) llms.ToolReturn {
	name, ok := args["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return core.NewErrorResponse("name parameter is required for load_skill")
	}

	skill, err := p.getSkill(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			names := make([]string, 0, len(p.skills))
			for skillName := range p.skills {
				names = append(names, skillName)
			}
			sort.Strings(names)
			return core.NewErrorResponse(fmt.Sprintf("%v. Available skills: %v", err, names))
		}
		return core.NewErrorResponse(fmt.Sprintf("failed to load skill '%s': %v", name, err))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Skill '%s'\n", skill.Name)
	if skill.Version != "" {
		fmt.Fprintf(&sb, "Version: %s\n", skill.Version)
	}
	fmt.Fprintf(&sb, "Description: %s\n", skill.Description)
	if skill.Usage != "" {
		fmt.Fprintf(&sb, "Usage: %s\n", skill.Usage)
	}
	sb.WriteString("\n")
	sb.WriteString(skill.Body)
	return core.NewEphemeralResponse(sb.String())
}

func (p *SkillsPlugin) handleListSkillReferences(args map[string]any) llms.ToolReturn {
	name, ok := args["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return core.NewErrorResponse("name parameter is required for list_skill_references")
	}

	skill, err := p.getSkill(name)
	if err != nil {
		return core.NewErrorResponse(err.Error())
	}
	paths, err := listReferenceFiles(skill)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to list references for skill '%s': %v", name, err))
	}
	if len(paths) == 0 {
		return core.NewEphemeralResponse(fmt.Sprintf("Skill '%s' has no reference files under references/.", skill.Name))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Reference files for skill '%s':\n", skill.Name)
	for _, path := range paths {
		fmt.Fprintf(&sb, "- %s\n", path)
	}
	sb.WriteString("\nUse load_skill_reference with referencePath to read one file.")
	return core.NewEphemeralResponse(sb.String())
}

func (p *SkillsPlugin) handleLoadSkillReference(args map[string]any) llms.ToolReturn {
	name, ok := args["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return core.NewErrorResponse("name parameter is required for load_skill_reference")
	}
	referencePath, ok := args["referencePath"].(string)
	if !ok || strings.TrimSpace(referencePath) == "" {
		return core.NewErrorResponse("referencePath parameter is required for load_skill_reference")
	}

	skill, err := p.getSkill(name)
	if err != nil {
		return core.NewErrorResponse(err.Error())
	}
	content, err := readReferenceFile(skill, referencePath)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to load reference '%s' for skill '%s': %v", referencePath, name, err))
	}

	return core.NewEphemeralResponse(fmt.Sprintf(
		"Skill '%s' reference '%s'\n\n%s",
		skill.Name, filepath.ToSlash(referencePath), content,
	))
}

func (p *SkillsPlugin) handleListInstallable() llms.ToolReturn {
	skills, err := fetchInstallableSkills()
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to list installable skills: %v", err))
	}
	if len(skills) == 0 {
		return core.NewEphemeralResponse(fmt.Sprintf("No installable skills found under %s on GitHub.", remoteSkillsRoot))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Installable skills from %s on the %s branch:\n\n", remoteSkillsRoot, githubBranch)
	for _, skill := range skills {
		fmt.Fprintf(&sb, "- %s\n", skill.Slug)
		if skill.Name != "" && skill.Name != skill.Slug {
			fmt.Fprintf(&sb, "  Name: %s\n", skill.Name)
		}
		if skill.Description != "" {
			fmt.Fprintf(&sb, "  Description: %s\n", skill.Description)
		}
		if skill.Version != "" {
			fmt.Fprintf(&sb, "  Version: %s\n", skill.Version)
		}
		if skill.Usage != "" {
			fmt.Fprintf(&sb, "  Usage: %s\n", skill.Usage)
		}
	}
	sb.WriteString("\nUse install_skill with a local path or repository/skills/<slug>.")
	return core.NewEphemeralResponse(sb.String())
}

func (p *SkillsPlugin) handleInstallSkill(args map[string]any) llms.ToolReturn {
	path, ok := args["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return core.NewErrorResponse("path parameter is required for install_skill")
	}

	skill, err := p.installSkill(path)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to install skill from '%s': %v", path, err))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Installed skill '%s' into skills/%s.\n", skill.Name, skill.Name)
	fmt.Fprintf(&sb, "Description: %s\n", skill.Description)
	if skill.Version != "" {
		fmt.Fprintf(&sb, "Version: %s\n", skill.Version)
	}
	if skill.Usage != "" {
		fmt.Fprintf(&sb, "Usage: %s\n", skill.Usage)
	}
	return core.NewEphemeralResponse(sb.String())
}

func (p *SkillsPlugin) handleDeleteSkill(args map[string]any) llms.ToolReturn {
	name, ok := args["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return core.NewErrorResponse("name parameter is required for delete_skill")
	}

	if err := p.deleteSkill(name); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to delete skill '%s': %v", name, err))
	}

	return core.NewEphemeralResponse(fmt.Sprintf("Deleted skill '%s' from skills/%s.", name, name))
}
