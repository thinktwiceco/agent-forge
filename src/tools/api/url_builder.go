package api

import (
	"fmt"
	neturl "net/url"
	"strings"
)

// buildURL builds the complete URL by replacing URL parameters with actual values
func (a *Api) buildURL(endpoint *Endpoint, args map[string]any) (string, error) {
	url := endpoint.URL

	// Replace URL parameters like {user_id} with actual values
	if urlParams, ok := args["url_params"].(map[string]any); ok {
		for key, value := range urlParams {
			placeholder := fmt.Sprintf("{%s}", key)
			url = strings.ReplaceAll(url, placeholder, fmt.Sprint(value))
		}
	}

	// Validate that all {param} placeholders are replaced.
	// Ignore ${ENV_VAR} patterns — those are resolved later by resolveString.
	stripped := strings.ReplaceAll(url, "${", "")
	if strings.Contains(stripped, "{") && strings.Contains(stripped, "}") {
		return "", fmt.Errorf("missing required URL parameters in URL: %s", url)
	}

	return url, nil
}

// addQueryParams adds query parameters to the URL
func (a *Api) addQueryParams(url string, queryParams any) string {
	if queryParams == nil {
		return url
	}

	params, ok := queryParams.(map[string]any)
	if !ok || len(params) == 0 {
		return url
	}

	// Build query string
	values := neturl.Values{}
	for key, value := range params {
		values.Add(key, fmt.Sprint(value))
	}

	separator := "?"
	if strings.Contains(url, "?") {
		separator = "&"
	}

	return url + separator + values.Encode()
}
