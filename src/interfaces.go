package agentforge

// Discoverable is an interface for components that can be progressively discovered.
// This interface enables agents to learn more about other agents or tools through
// multiple levels of information disclosure.
type Discoverable interface {
	// BasicDescription returns a short one-line description of the component.
	BasicDescription() string

	// AdvanceDescription returns detailed information about the component's
	// capabilities, parameters, usage patterns, and behavior.
	AdvanceDescription() string

	// Troubleshooting returns information about common issues, debugging tips,
	// validation errors, and configuration guidance.
	Troubleshooting() string

	// DetailsAbout returns detailed information about a specific item (action,
	// operation, endpoint, etc.). Returns "Nothing to add about <item>" if the
	// component has no per-item details.
	DetailsAbout(item string) string
}
