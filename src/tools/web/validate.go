package web

import (
	"fmt"
	"net/url"
	"strings"
)

// validateAction ensures that the action is one of the valid web browser actions.
func validateAction(value any) error {
	action, ok := value.(string)
	if !ok {
		return fmt.Errorf("action must be a string")
	}
	validActions := map[string]bool{
		"navigate":    true,
		"click":       true,
		"type":        true,
		"screenshot":  true,
		"get_content": true,
		"wait":        true,
		"back":        true,
		"forward":     true,
		"evaluate":    true,
	}
	if !validActions[action] {
		return fmt.Errorf("invalid action: %s. Must be one of: navigate, click, type, screenshot, get_content, wait, back, forward, evaluate", action)
	}
	return nil
}

// validateURL ensures that the URL is valid.
func validateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Add scheme if missing
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	return nil
}

// validateSelector ensures that the CSS selector is not empty.
func validateSelector(selector string) error {
	if selector == "" {
		return fmt.Errorf("selector cannot be empty")
	}
	return nil
}

