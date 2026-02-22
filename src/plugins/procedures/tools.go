package procedures

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const PROCEDURE_TOOL = "procedure"

func newProcedureTool(plugin *ProceduresPlugin) llms.Tool {
	return &core.Tool{
		Name:        PROCEDURE_TOOL,
		Description: `Execute structured multi-step procedures. start_procedure: begin named procedure (requires name). next_step: advance to next step. goto_step: jump to a specific step (requires stepNumber). Each action returns files with instructions.`,
		AdvanceDesc: `[ACTIONS]
- start_procedure: Begin from step 0. Required: name
- next_step: Advance to next step of active procedure
- goto_step: Jump to a specific step by number. Required: stepNumber (0-based)

[STEP CONTENT]
- Each action returns step folder files. Read for instructions.
- Procedure names in manifest.yaml. System prompt lists available procedures.

[CREATING PROCEDURES]
- New procedures MUST be created inside procedures/ (e.g. procedures/my-procedure/manifest.yaml, procedures/my-procedure/0/instructions.md). Never create at working dir root.`,
		TroubleshootingInfo: `Troubleshooting:
- Ensure 'action' is 'start_procedure', 'next_step', or 'goto_step'.
- 'name' is required for start_procedure and must match a known procedure name exactly.
- 'stepNumber' is required for goto_step (0-based index).
- Call start_procedure before next_step or goto_step; there is no active procedure otherwise.
- next_step returns an error when the last step has already been reached.`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action: 'start_procedure', 'next_step', or 'goto_step'",
				Required:    true,
			},
			{
				Name:        "name",
				Type:        "string",
				Description: "The procedure name to start (required for start_procedure)",
				Required:    false,
			},
			{
				Name:        "stepNumber",
				Type:        "number",
				Description: "0-based step index (required for goto_step)",
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
			case "next_step":
				return plugin.handleNextStep()
			case "goto_step":
				return plugin.handleGotoStep(args)
			default:
				return core.NewErrorResponse(fmt.Sprintf(
					"unknown action '%s'. Valid actions are: 'start_procedure', 'next_step', 'goto_step'", action,
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
		"Current step: 0\nProcedure '%s' — Step 0 of %d\n\n%s",
		proc.Name, proc.PhaseCount-1, content,
	))
}

func (p *ProceduresPlugin) handleNextStep() llms.ToolReturn {
	if p.activeProcedure == nil {
		return core.NewErrorResponse("no active procedure. Call start_procedure first")
	}

	nextPhase := p.currentPhase + 1
	if nextPhase >= p.activeProcedure.PhaseCount {
		return core.NewErrorResponse(fmt.Sprintf(
			"procedure '%s' has no step %d. It ended at step %d",
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
	status := fmt.Sprintf("Step %d of %d", nextPhase, p.activeProcedure.PhaseCount-1)
	if isLast {
		status += " (final step)"
	}

	return core.NewEphemeralResponse(fmt.Sprintf(
		"Current step: %d\nProcedure '%s' — %s\n\n%s",
		nextPhase, p.activeProcedure.Name, status, content,
	))
}

func (p *ProceduresPlugin) handleGotoStep(args map[string]any) llms.ToolReturn {
	if p.activeProcedure == nil {
		return core.NewErrorResponse("no active procedure. Call start_procedure first")
	}

	stepVal, ok := args["stepNumber"]
	if !ok || stepVal == nil {
		return core.NewErrorResponse("stepNumber is required for goto_step")
	}
	var step int
	switch v := stepVal.(type) {
	case float64:
		step = int(v)
	case int:
		step = v
	default:
		return core.NewErrorResponse("stepNumber must be a number")
	}

	if step < 0 || step >= p.activeProcedure.PhaseCount {
		return core.NewErrorResponse(fmt.Sprintf(
			"step %d is out of range. Procedure '%s' has steps 0 to %d",
			step, p.activeProcedure.Name, p.activeProcedure.PhaseCount-1,
		))
	}

	p.currentPhase = step

	content, err := p.loadPhaseContent(p.activeProcedure, step)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf(
			"failed to load step %d of procedure '%s': %v",
			step, p.activeProcedure.Name, err,
		))
	}

	status := fmt.Sprintf("Step %d of %d", step, p.activeProcedure.PhaseCount-1)
	if step == p.activeProcedure.PhaseCount-1 {
		status += " (final step)"
	}

	return core.NewEphemeralResponse(fmt.Sprintf(
		"Current step: %d\nProcedure '%s' — %s\n\n%s",
		step, p.activeProcedure.Name, status, content,
	))
}
