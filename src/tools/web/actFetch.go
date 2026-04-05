package web

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

var fetchHTTPClient = &http.Client{Timeout: 30 * time.Second}

// fetch performs a lightweight HTTP GET without launching Chrome (static pages, APIs).
func (w *WebBrowser) fetch(_ map[string]any, args map[string]any) llms.ToolReturn {
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return core.NewErrorResponse("url is required for fetch action")
	}
	normalized, err := normalizeURL(urlStr)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("invalid URL: %v", err))
	}

	req, err := http.NewRequest(http.MethodGet, normalized, nil)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to build request: %v", err))
	}
	ua := os.Getenv("WEB_FETCH_USER_AGENT")
	if ua == "" {
		ua = "Mozilla/5.0 (compatible; AgentForge/1.0)"
	}
	req.Header.Set("User-Agent", ua)

	resp, err := fetchHTTPClient.Do(req)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("fetch failed: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	maxBytes := int64(2 << 20)
	if v := os.Getenv("WEB_FETCH_MAX_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxBytes = n
		}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to read response: %v", err))
	}

	ct := resp.Header.Get("Content-Type")
	var content string
	if strings.Contains(strings.ToLower(ct), "text/html") {
		content, err = htmltomarkdown.ConvertString(string(body))
		if err != nil {
			content = string(body)
		}
	} else {
		content = string(body)
	}

	return core.NewSuccessResponse(fmt.Sprintf(
		"fetch %s\nStatus: %d\n\n%s",
		normalized, resp.StatusCode, strings.TrimSpace(content),
	))
}
