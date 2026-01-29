package agents

import (
	"errors"
	"testing"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestAgentHooks_RegistrationAndTrigger(t *testing.T) {
	hooks := newAgentHooks()

	// Test OnNewUserMessage
	called := false
	hook := func(a *Agent, message string) error {
		called = true
		if message != "hello" {
			return errors.New("wrong message")
		}
		return nil
	}

	// Use the generic 'on' method
	hooks.on(core.EventNewUserMessage, OnNewUserMessageHook(hook))

	// Create dummy agent
	agent := &Agent{config: &AgentConfig{AgentName: "TestAgent", Trace: "test"}}

	// Trigger
	errs := hooks.newUserMessageEvent(agent, "hello")
	if !called {
		t.Error("Hook not called")
	}
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errs))
	}

	// Trigger error
	called = false
	errs = hooks.newUserMessageEvent(agent, "wrong")
	if len(errs) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errs))
	}
}

func TestAgentHooks_MultipleHooks(t *testing.T) {
	hooks := newAgentHooks()
	counter := 0

	h1 := func(a *Agent, ctx *core.AgentContext) error {
		counter++
		return nil
	}
	h2 := func(a *Agent, ctx *core.AgentContext) error {
		counter++
		return nil
	}

	hooks.on(core.EventContextBuild, OnContextBuildHook(h1))
	hooks.on(core.EventContextBuild, OnContextBuildHook(h2))

	agent := &Agent{config: &AgentConfig{AgentName: "TestAgent", Trace: "test"}}
	hooks.contextBuildEvent(agent, nil)

	if counter != 2 {
		t.Errorf("Expected counter 2, got %d", counter)
	}
}

func TestAgentHooks_ToolEvents(t *testing.T) {
	hooks := newAgentHooks()

	// Before Tool
	hooks.on(core.EventBeforeToolExecution, BeforeToolExecutionHook(func(a *Agent, tc *llms.ToolCall) error {
		return nil
	}))

	agent := &Agent{config: &AgentConfig{AgentName: "TestAgent", Trace: "test"}}

	errs := hooks.beforeToolExecutionEvent(agent, nil)
	if len(errs) != 0 {
		t.Error("Error in before tool hook")
	}

	// On Tool
	hooks.on(core.EventToolExecution, OnToolExecutionHook(func(a *Agent, tr *llms.ToolResult) error {
		return nil
	}))
	errs = hooks.toolExecutionEvent(agent, nil)
	if len(errs) != 0 {
		t.Error("Error in tool exec hook")
	}
}
