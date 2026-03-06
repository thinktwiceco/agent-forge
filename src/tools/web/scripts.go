package web

import (
	_ "embed"
)

//go:embed scripts/interactive_tree.js
var interactiveTreeScript string

//go:embed scripts/strip_unwanted_content.js
var stripUnwantedContentScript string

// getScript returns the JavaScript code for the specified script name.
// Scripts are embedded at compile time using go:embed directives.
func getScript(name string) string {
	switch name {
	case "interactive_tree":
		return interactiveTreeScript
	case "strip_unwanted_content":
		return stripUnwantedContentScript
	default:
		return ""
	}
}
