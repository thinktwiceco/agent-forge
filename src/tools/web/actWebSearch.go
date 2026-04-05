package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

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

type tavilySearchResponse struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	} `json:"results"`
}

type webSearchCacheEntry struct {
	payload string
	expires time.Time
}

func webSearchCacheTTL() time.Duration {
	raw := os.Getenv("WEB_SEARCH_CACHE_TTL_S")
	if raw == "" {
		return 5 * time.Minute
	}
	s, err := strconv.Atoi(raw)
	if err != nil || s < 0 {
		return 5 * time.Minute
	}
	if s == 0 {
		return 0
	}
	return time.Duration(s) * time.Second
}

func webSearchCacheKey(args map[string]any) string {
	m := map[string]any{}
	for _, k := range []string{"query", "count", "offset", "country", "search_lang", "freshness"} {
		if v, ok := args[k]; ok {
			m[k] = v
		}
	}
	b, _ := json.Marshal(m)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// webSearch performs a web search using Brave Search API, or Tavily if Brave is not configured.
func (w *WebBrowser) webSearch(_ map[string]any, args map[string]any) llms.ToolReturn {
	ttl := webSearchCacheTTL()
	key := webSearchCacheKey(args)
	if ttl > 0 && w.searchCache != nil {
		w.searchMu.Lock()
		if ent, ok := w.searchCache[key]; ok && time.Now().Before(ent.expires) {
			w.searchMu.Unlock()
			return core.NewSuccessResponse(ent.payload)
		}
		w.searchMu.Unlock()
	}

	if k := os.Getenv("AF_BRAVE_API_KEY"); k != "" {
		resp, err := braveSearchRequest(k, args)
		if err != nil {
			return core.NewErrorResponse(err.Error())
		}
		out := resp.String()
		w.cacheSearchResult(key, out, ttl)
		agentforge.Info("Web search (Brave) returned %d results", len(resp.Results))
		return core.NewSuccessResponse(out)
	}
	if k := os.Getenv("AF_TAVILY_API_KEY"); k != "" {
		resp, err := tavilySearchRequest(k, args)
		if err != nil {
			return core.NewErrorResponse(err.Error())
		}
		out := resp.String()
		w.cacheSearchResult(key, out, ttl)
		agentforge.Info("Web search (Tavily) returned %d results", len(resp.Results))
		return core.NewSuccessResponse(out)
	}
	return core.NewErrorResponse("web_search requires AF_BRAVE_API_KEY or AF_TAVILY_API_KEY")
}

func (w *WebBrowser) cacheSearchResult(cacheKey, payload string, ttl time.Duration) {
	if ttl <= 0 || w.searchCache == nil {
		return
	}
	w.searchMu.Lock()
	defer w.searchMu.Unlock()
	w.searchCache[cacheKey] = webSearchCacheEntry{
		payload: payload,
		expires: time.Now().Add(ttl),
	}
}

func braveSearchRequest(apiKey string, args map[string]any) (*webSearchResponse, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required for web_search action and must be a non-empty string")
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
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("brave search request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave search API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp braveSearchResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	results := make([]webSearchResult, len(apiResp.Web.Results))
	for i, r := range apiResp.Web.Results {
		results[i] = webSearchResult(r)
	}

	return &webSearchResponse{
		Query:   query,
		Results: results,
		Success: true,
	}, nil
}

func tavilySearchRequest(apiKey string, args map[string]any) (*webSearchResponse, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required for web_search action and must be a non-empty string")
	}

	maxResults := 10
	if v, ok := args["count"]; ok {
		switch c := v.(type) {
		case float64:
			maxResults = int(c)
		case int:
			maxResults = c
		}
		if maxResults < 1 {
			maxResults = 1
		}
		if maxResults > 20 {
			maxResults = 20
		}
	}

	payload := map[string]any{
		"api_key":     apiKey,
		"query":       query,
		"max_results": maxResults,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily search request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read tavily response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily search API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var tv tavilySearchResponse
	if err := json.Unmarshal(respBody, &tv); err != nil {
		return nil, fmt.Errorf("failed to parse tavily response: %w", err)
	}

	results := make([]webSearchResult, len(tv.Results))
	for i, r := range tv.Results {
		results[i] = webSearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Description: r.Content,
		}
	}

	return &webSearchResponse{
		Query:   query,
		Results: results,
		Success: true,
	}, nil
}
