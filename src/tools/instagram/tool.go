package instagram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/tools/api"
)

const toolName = "instagram"

type paramKind int

const (
	kindURL   paramKind = iota // replaces {param} in the URL path
	kindQuery                  // appended as query string
	kindBody                   // included in the JSON request body
)

type actionDef struct {
	ep     api.Endpoint
	params map[string]paramKind
}

// actions maps each Instagram action to its endpoint and parameter routing.
var actions map[string]actionDef

func init() {
	epByName := make(map[string]api.Endpoint, len(endpoints))
	for _, ep := range endpoints {
		epByName[ep.Name] = ep
	}
	actions = map[string]actionDef{
		"get_profile": {
			ep:     epByName["get_profile"],
			params: map[string]paramKind{"fields": kindQuery},
		},
		"list_media": {
			ep:     epByName["list_media"],
			params: map[string]paramKind{"fields": kindQuery, "limit": kindQuery, "after": kindQuery},
		},
		"get_media": {
			ep:     epByName["get_media"],
			params: map[string]paramKind{"media_id": kindURL, "fields": kindQuery},
		},
		"create_media_container": {
			ep: epByName["create_media_container"],
			params: map[string]paramKind{
				"image_url": kindBody, "video_url": kindBody, "media_type": kindBody,
				"caption": kindBody, "is_carousel_item": kindBody, "children": kindBody, "alt_text": kindBody,
			},
		},
		"publish_media": {
			ep:     epByName["publish_media"],
			params: map[string]paramKind{"creation_id": kindBody},
		},
		"get_account_insights": {
			ep:     epByName["get_account_insights"],
			params: map[string]paramKind{"metric": kindQuery, "period": kindQuery, "since": kindQuery, "until": kindQuery},
		},
		"get_media_insights": {
			ep:     epByName["get_media_insights"],
			params: map[string]paramKind{"media_id": kindURL, "metric": kindQuery},
		},
		"get_comments": {
			ep:     epByName["get_comments"],
			params: map[string]paramKind{"media_id": kindURL, "fields": kindQuery, "limit": kindQuery, "after": kindQuery},
		},
		"reply_to_comment": {
			ep:     epByName["reply_to_comment"],
			params: map[string]paramKind{"comment_id": kindURL, "message": kindBody},
		},
	}
}

type instagramTool struct {
	headers map[string]string
}

// NewInstagramTool creates a flat-action Instagram tool.
// headers must include at minimum "Authorization: Bearer <token>".
func NewInstagramTool(headers map[string]string) llms.Tool {
	if headers == nil {
		headers = map[string]string{}
	}
	t := &instagramTool{headers: headers}
	return &core.Tool{
		Name:        toolName,
		Description: "Instagram Graph API. Call any action with flat parameters — no nested objects needed.",
		AdvanceDesc: buildAdvancedDescription(),
		Parameters:  buildParameters(),
		Handler:     t.handler,
	}
}

func buildAdvancedDescription() string {
	var b strings.Builder
	b.WriteString("Instagram Graph API — specify an action and pass its parameters directly.\n\n")
	b.WriteString("Actions:\n")
	for name, def := range actions {
		params := make([]string, 0, len(def.params))
		for p := range def.params {
			params = append(params, p)
		}
		fmt.Fprintf(&b, "  %-28s %s\n    params: %s\n", name, def.ep.Description, strings.Join(params, ", "))
	}
	return b.String()
}

func buildParameters() []core.Parameter {
	return []core.Parameter{
		{
			Name:     "action",
			Type:     "string",
			Required: true,
			Description: "Action to perform. One of: get_profile, list_media, get_media, " +
				"create_media_container, publish_media, get_account_insights, get_media_insights, " +
				"get_comments, reply_to_comment",
		},
		{
			Name:        "fields",
			Type:        "string",
			Description: "Comma-separated fields to return. Used by: get_profile, list_media, get_media, get_comments",
		},
		{
			Name:        "limit",
			Type:        "number",
			Description: "Max items to return. Used by: list_media (max 100), get_comments",
		},
		{
			Name:        "after",
			Type:        "string",
			Description: "Pagination cursor for forward pagination. Used by: list_media, get_comments",
		},
		{
			Name:        "media_id",
			Type:        "string",
			Description: "ID of the media post. Required for: get_media, get_media_insights, get_comments",
		},
		{
			Name:        "comment_id",
			Type:        "string",
			Description: "ID of the comment to reply to. Required for: reply_to_comment",
		},
		{
			Name:        "metric",
			Type:        "string",
			Description: "Comma-separated metrics (e.g. impressions,reach). Required for: get_account_insights, get_media_insights",
		},
		{
			Name:        "period",
			Type:        "string",
			Description: "Aggregation period: day, week, month, lifetime. Used by: get_account_insights",
		},
		{
			Name:        "since",
			Type:        "string",
			Description: "Unix timestamp start of range. Used by: get_account_insights",
		},
		{
			Name:        "until",
			Type:        "string",
			Description: "Unix timestamp end of range. Used by: get_account_insights",
		},
		{
			Name:        "image_url",
			Type:        "string",
			Description: "Public HTTPS URL of the image (JPEG). Used by: create_media_container",
		},
		{
			Name:        "video_url",
			Type:        "string",
			Description: "Public HTTPS URL of the video. Used by: create_media_container",
		},
		{
			Name:        "media_type",
			Type:        "string",
			Description: "Media type: IMAGE (default), VIDEO, REELS, STORIES, CAROUSEL. Used by: create_media_container",
		},
		{
			Name:        "caption",
			Type:        "string",
			Description: "Post caption text. Used by: create_media_container",
		},
		{
			Name:        "is_carousel_item",
			Type:        "boolean",
			Description: "True if this container is a carousel item. Used by: create_media_container",
		},
		{
			Name:        "children",
			Type:        "string",
			Description: "Comma-separated container IDs for carousel items. Used by: create_media_container with CAROUSEL media_type",
		},
		{
			Name:        "alt_text",
			Type:        "string",
			Description: "Accessible alt text for image posts. Used by: create_media_container",
		},
		{
			Name:        "creation_id",
			Type:        "string",
			Description: "Container ID from create_media_container. Required for: publish_media",
		},
		{
			Name:        "message",
			Type:        "string",
			Description: "Reply text. Required for: reply_to_comment",
		},
	}
}

func (t *instagramTool) handler(_ map[string]any, args map[string]any) llms.ToolReturn {
	action, _ := args["action"].(string)
	def, ok := actions[action]
	if !ok {
		names := make([]string, 0, len(actions))
		for n := range actions {
			names = append(names, n)
		}
		return core.NewErrorResponse(fmt.Sprintf("unknown action %q. Available: %s", action, strings.Join(names, ", ")))
	}

	rawURL := def.ep.URL
	queryVals := url.Values{}
	bodyMap := map[string]any{}

	for param, kind := range def.params {
		val, exists := args[param]
		if !exists || val == nil {
			continue
		}
		switch kind {
		case kindURL:
			rawURL = strings.ReplaceAll(rawURL, "{"+param+"}", fmt.Sprint(val))
		case kindQuery:
			queryVals.Set(param, fmt.Sprint(val))
		case kindBody:
			bodyMap[param] = val
		}
	}

	if len(queryVals) > 0 {
		if strings.Contains(rawURL, "?") {
			rawURL += "&" + queryVals.Encode()
		} else {
			rawURL += "?" + queryVals.Encode()
		}
	}

	var bodyStr string
	if len(bodyMap) > 0 {
		bodyBytes, err := json.Marshal(bodyMap)
		if err != nil {
			return core.NewErrorResponse(fmt.Sprintf("failed to serialize body: %v", err))
		}
		bodyStr = string(bodyBytes)
	}

	return t.execute(def.ep, rawURL, bodyStr)
}

func (t *instagramTool) execute(ep api.Endpoint, rawURL, body string) llms.ToolReturn {
	client := &http.Client{Timeout: 30 * time.Second}

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(ep.Method, rawURL, reqBody)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to create request: %v", err))
	}

	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("request failed: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to read response: %v", err))
	}

	result := fmt.Sprintf("Status: %d\nAction: %s\nURL: %s\n\n%s", resp.StatusCode, ep.Name, rawURL, string(respBody))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return core.NewSuccessResponse(result)
	}
	return core.NewFailureResponse(fmt.Sprintf("HTTP %d", resp.StatusCode), result)
}
