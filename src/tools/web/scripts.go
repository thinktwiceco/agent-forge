package web

import (
	_ "embed"
)

//go:embed scripts/interactive_tree.js
var interactiveTreeScript string

//go:embed scripts/strip_unwanted_content.js
var stripUnwantedContentScript string

//go:embed scripts/clear_input.js
var clearInputScript string

//go:embed scripts/network_idle.js
var networkIdleScript string

//go:embed scripts/stealth_patch.js
var stealthPatchScript string

// getScript returns the JavaScript code for the specified script name.
// Scripts are embedded at compile time using go:embed directives.
func getScript(name string) string {
	switch name {
	case "interactive_tree":
		return interactiveTreeScript
	case "strip_unwanted_content":
		return stripUnwantedContentScript
	case "clear_input":
		return clearInputScript
	case "network_idle":
		return networkIdleScript
	case "stealth_patch":
		return stealthPatchScript
	default:
		return ""
	}
}
