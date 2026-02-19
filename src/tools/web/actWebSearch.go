package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const braveSearchURL = "https://api.search.brave.com/res/v1/web/search"

// braveSearchResponse is the top-level response from the Brave Search API.
type braveSearchResponse struct {
	Web struct {
		Results []braveSearchResult `json:"results"`
	} `json:"web"`
}

type braveSearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// webSearch performs a web search using the Brave Search API.
func (w *WebBrowser) webSearch(_ map[string]any, args map[string]any) llms.ToolReturn {
	apiKey := os.Getenv("AF_BRAVE_API_KEY")
	if apiKey == "" {
		return core.NewErrorResponse("web_search requires a Brave API key (AF_BRAVE_API_KEY environment variable not set)")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return core.NewErrorResponse("query parameter is required for web_search action and must be a non-empty string")
	}

	params := url.Values{}
	params.Set("q", query)

	count := 10
	if v, ok := args["count"]; ok {
		switch c := v.(type) {
		case float64:
			count = int(c)
		case int:
			count = c
		}
		if count < 1 {
			count = 1
		}
		if count > 20 {
			count = 20
		}
	}
	params.Set("count", strconv.Itoa(count))

	if v, ok := args["offset"]; ok {
		switch o := v.(type) {
		case float64:
			params.Set("offset", strconv.Itoa(int(o)))
		case int:
			params.Set("offset", strconv.Itoa(o))
		}
	}

	if v, ok := args["country"].(string); ok && v != "" {
		params.Set("country", v)
	}

	if v, ok := args["search_lang"].(string); ok && v != "" {
		params.Set("search_lang", v)
	}

	if v, ok := args["freshness"].(string); ok && v != "" {
		params.Set("freshness", v)
	}

	reqURL := braveSearchURL + "?" + params.Encode()
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to build request: %v", err))
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("brave search request failed: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to read response: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		return core.NewErrorResponse(fmt.Sprintf("brave search API error (status %d): %s", resp.StatusCode, string(body)))
	}

	var apiResp braveSearchResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to parse response: %v", err))
	}

	results := make([]webSearchResult, len(apiResp.Web.Results))
	for i, r := range apiResp.Web.Results {
		results[i] = webSearchResult(r)
	}

	response := &webSearchResponse{
		Query:   query,
		Results: results,
		Success: true,
	}

	agentforge.Info("Web search for %q returned %d results", query, len(results))
	return core.NewSuccessResponse(response.String())
}
