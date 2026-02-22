package procedures

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const PROCEDURE_TOOL = "procedure"

func newProcedureTool(plugin *ProceduresPlugin) llms.Tool {
	return &core.Tool{
		Name: PROCEDURE_TOOL,
		Description: `Execute structured procedures that guide you through multi-step tasks.
Use start_procedure to begin a named procedure, then next_phase to advance through its phases.
Each phase contains instructions you must follow before moving to the next one.`,
		AdvanceDesc: `Advanced Details:
- Actions:
  * start_procedure: Begin a procedure from phase 0. Requires the 'name' parameter.
  * next_phase: Advance to the next phase of the currently active procedure.

- Phase content: Each action returns the files contained in the phase folder.
  Read them carefully — they contain the instructions for that phase.

- Procedure names are defined in each procedure's manifest.yaml.
  The system prompt lists all available procedures with their descriptions.`,
		TroubleshootingInfo: `Troubleshooting:
- Ensure 'action' is either 'start_procedure' or 'next_phase'.
- 'name' is required for start_procedure and must match a known procedure name exactly.
- Call start_procedure before next_phase; there is no active procedure otherwise.
- next_phase returns an error when the last phase has already been reached.`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform: 'start_procedure' or 'next_phase'",
				Required:    true,
			},
			{
				Name:        "name",
				Type:        "string",
				Description: "The procedure name to start (required for start_procedure)",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			action, ok := args["action"].(string)
			if !ok {
				return core.NewErrorResponse("action parameter is required and must be a string")
			}

			switch action {
			case "start_procedure":
				return plugin.handleStartProcedure(args)
			case "next_phase":
				return plugin.handleNextPhase()
			default:
				return core.NewErrorResponse(fmt.Sprintf(
					"unknown action '%s'. Valid actions are: 'start_procedure', 'next_phase'", action,
				))
			}
		},
	}
}

func (p *ProceduresPlugin) handleStartProcedure(args map[string]any) llms.ToolReturn {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return core.NewErrorResponse("name parameter is required for start_procedure")
	}

	proc, exists := p.procedures[name]
	if !exists {
		available := make([]string, 0, len(p.procedures))
		for n := range p.procedures {
			available = append(available, n)
		}
		return core.NewErrorResponse(fmt.Sprintf(
			"procedure '%s' not found. Available procedures: %v", name, available,
		))
	}

	p.activeProcedure = proc
	p.currentPhase = 0

	content, err := p.loadPhaseContent(proc, 0)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to load phase 0 of procedure '%s': %v", name, err))
	}

	return core.NewEphemeralResponse(fmt.Sprintf(
		"Started procedure '%s' — Phase 0 of %d\n\n%s",
		proc.Name, proc.PhaseCount-1, content,
	))
}

func (p *ProceduresPlugin) handleNextPhase() llms.ToolReturn {
	if p.activeProcedure == nil {
		return core.NewErrorResponse("no active procedure. Call start_procedure first")
	}

	nextPhase := p.currentPhase + 1
	if nextPhase >= p.activeProcedure.PhaseCount {
		return core.NewErrorResponse(fmt.Sprintf(
			"procedure '%s' has no phase %d. It ended at phase %d",
			p.activeProcedure.Name, nextPhase, p.currentPhase,
		))
	}

	p.currentPhase = nextPhase

	content, err := p.loadPhaseContent(p.activeProcedure, nextPhase)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf(
			"failed to load phase %d of procedure '%s': %v",
			nextPhase, p.activeProcedure.Name, err,
		))
	}

	isLast := nextPhase == p.activeProcedure.PhaseCount-1
	status := fmt.Sprintf("Phase %d of %d", nextPhase, p.activeProcedure.PhaseCount-1)
	if isLast {
		status += " (final phase)"
	}

	return core.NewEphemeralResponse(fmt.Sprintf(
		"Procedure '%s' — %s\n\n%s",
		p.activeProcedure.Name, status, content,
	))
}
