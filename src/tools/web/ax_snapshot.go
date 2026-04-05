package web

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Chrome can add AX property/value enum members before cdproto updates.
// Decode the AX tree into tolerant local structs so snapshotting keeps working.
type axValue struct {
	Value json.RawMessage `json:"value,omitempty"`
}

type axProperty struct {
	Name  string   `json:"name"`
	Value *axValue `json:"value,omitempty"`
}

type axNode struct {
	NodeID         accessibility.NodeID   `json:"nodeId"`
	Ignored        bool                   `json:"ignored"`
	IgnoredReasons []*axProperty          `json:"ignoredReasons,omitempty"`
	Role           *axValue               `json:"role,omitempty"`
	ChromeRole     *axValue               `json:"chromeRole,omitempty"`
	Name           *axValue               `json:"name,omitempty"`
	Description    *axValue               `json:"description,omitempty"`
	Value          *axValue               `json:"value,omitempty"`
	Properties     []*axProperty          `json:"properties,omitempty"`
	ParentID       accessibility.NodeID   `json:"parentId,omitempty"`
	ChildIDs       []accessibility.NodeID `json:"childIds,omitempty"`
}

type axTreeResponse struct {
	Nodes []*axNode `json:"nodes"`
}

// axValueString extracts a readable string from an AX value payload.
func axValueString(v *axValue) string {
	if v == nil || len(v.Value) == 0 {
		return ""
	}
	var raw any
	if err := json.Unmarshal(v.Value, &raw); err != nil {
		s := strings.TrimSpace(string(v.Value))
		return strings.Trim(s, `"`)
	}
	switch t := raw.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%g", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return string(v.Value)
		}
		return string(b)
	}
}

// flattenAXTree converts the flat AX node list into nested maps for JSON output.
func flattenAXTree(nodes []*axNode) []map[string]any {
	byID := make(map[accessibility.NodeID]*axNode)
	for _, n := range nodes {
		byID[n.NodeID] = n
	}
	var roots []*axNode
	for _, n := range nodes {
		if n.ParentID == "" || byID[n.ParentID] == nil {
			roots = append(roots, n)
		}
	}
	out := make([]map[string]any, 0, len(roots))
	for _, r := range roots {
		if m := axNodeToMap(r, byID); m != nil {
			out = append(out, m)
		}
	}
	return out
}

func axNodeToMap(n *axNode, byID map[accessibility.NodeID]*axNode) map[string]any {
	if n.Ignored {
		return nil
	}
	role := axValueString(n.Role)
	name := axValueString(n.Name)
	if (role == "none" || role == "generic") && name == "" {
		return mergeChildrenOnly(n, byID)
	}
	m := map[string]any{}
	if role != "" {
		m["role"] = role
	}
	if name != "" {
		m["name"] = name
	}
	if d := axValueString(n.Description); d != "" {
		m["description"] = d
	}
	if v := axValueString(n.Value); v != "" {
		m["value"] = v
	}
	for _, p := range n.Properties {
		if p == nil {
			continue
		}
		switch p.Name {
		case string(accessibility.PropertyNameChecked),
			string(accessibility.PropertyNameExpanded),
			string(accessibility.PropertyNameSelected),
			string(accessibility.PropertyNameDisabled),
			string(accessibility.PropertyNameFocused):
			if p.Value != nil {
				m[p.Name] = axValueString(p.Value)
			}
		}
	}
	if ch := childrenMaps(n, byID); len(ch) > 0 {
		m["children"] = ch
	}
	return m
}

func mergeChildrenOnly(n *axNode, byID map[accessibility.NodeID]*axNode) map[string]any {
	ch := childrenMaps(n, byID)
	if len(ch) == 0 {
		return nil
	}
	if len(ch) == 1 {
		return ch[0]
	}
	return map[string]any{"children": ch}
}

func childrenMaps(n *axNode, byID map[accessibility.NodeID]*axNode) []map[string]any {
	out := make([]map[string]any, 0, len(n.ChildIDs))
	for _, cid := range n.ChildIDs {
		child := byID[cid]
		if child == nil {
			continue
		}
		if cm := axNodeToMap(child, byID); cm != nil {
			out = append(out, cm)
		}
	}
	return out
}

// formatAccessibilitySnapshotText renders the AX tree as indented text for inline get_content.
func formatAccessibilitySnapshotText(nodes []*axNode) string {
	byID := make(map[accessibility.NodeID]*axNode)
	for _, n := range nodes {
		byID[n.NodeID] = n
	}
	var b strings.Builder
	var walk func(n *axNode, depth int)
	walk = func(n *axNode, depth int) {
		if n.Ignored {
			return
		}
		role := axValueString(n.Role)
		name := axValueString(n.Name)
		if (role == "none" || role == "generic") && name == "" {
			for _, cid := range n.ChildIDs {
				if ch := byID[cid]; ch != nil {
					walk(ch, depth)
				}
			}
			return
		}
		pad := strings.Repeat("  ", depth)
		line := pad + role
		if name != "" {
			line += ": " + name
		}
		if v := axValueString(n.Value); v != "" {
			line += " [" + v + "]"
		}
		b.WriteString(line)
		b.WriteByte('\n')
		for _, cid := range n.ChildIDs {
			if ch := byID[cid]; ch != nil {
				walk(ch, depth+1)
			}
		}
	}
	for _, n := range nodes {
		if n.ParentID == "" || byID[n.ParentID] == nil {
			walk(n, 0)
		}
	}
	return strings.TrimSpace(b.String())
}

const (
	snapshotPollAttempts = 20
	snapshotPollInterval = 350 * time.Millisecond
)

func fetchFullAXTreeSingleFrame(ctx context.Context, frameID cdp.FrameID) ([]*axNode, error) {
	var res axTreeResponse
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
		p := accessibility.GetFullAXTree().WithFrameID(frameID)
		return cdp.Execute(c, accessibility.CommandGetFullAXTree, p, &res)
	}))
	if err != nil {
		return nil, err
	}
	return res.Nodes, nil
}

func remapAXNodeIDs(nodes []*axNode, prefix string) {
	if prefix == "" || len(nodes) == 0 {
		return
	}
	oldToNew := make(map[accessibility.NodeID]accessibility.NodeID, len(nodes))
	for _, n := range nodes {
		oldToNew[n.NodeID] = accessibility.NodeID(prefix + string(n.NodeID))
	}
	for _, n := range nodes {
		n.NodeID = oldToNew[n.NodeID]
		if n.ParentID != "" {
			if nid, ok := oldToNew[n.ParentID]; ok {
				n.ParentID = nid
			}
		}
		for i, cid := range n.ChildIDs {
			if nid, ok := oldToNew[cid]; ok {
				n.ChildIDs[i] = nid
			}
		}
	}
}

func collectFrameIDs(ft *page.FrameTree) []cdp.FrameID {
	if ft == nil {
		return nil
	}
	var out []cdp.FrameID
	var walk func(*page.FrameTree)
	walk = func(t *page.FrameTree) {
		if t == nil {
			return
		}
		out = append(out, t.Frame.ID)
		for _, ch := range t.ChildFrames {
			walk(ch)
		}
	}
	walk(ft)
	return out
}

// fetchFullAXTree returns the merged accessibility tree for all frames in the
// current target. The root-document-only CDP call misses OOPIF / subframe
// documents; per-frame getFullAXTree matches what users see when the login UI
// lives in an iframe.
func fetchFullAXTree(ctx context.Context) ([]*axNode, error) {
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
		return accessibility.Enable().Do(c)
	})); err != nil {
		return nil, err
	}
	var ft *page.FrameTree
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
		var err error
		ft, err = page.GetFrameTree().Do(c)
		return err
	})); err != nil {
		return nil, err
	}
	frames := collectFrameIDs(ft)
	if len(frames) == 0 {
		return nil, fmt.Errorf("empty frame tree")
	}
	if len(frames) == 1 {
		return fetchFullAXTreeSingleFrame(ctx, frames[0])
	}
	var all []*axNode
	for i, fid := range frames {
		part, err := fetchFullAXTreeSingleFrame(ctx, fid)
		if err != nil {
			return nil, err
		}
		remapAXNodeIDs(part, fmt.Sprintf("mf%d:", i))
		all = append(all, part...)
	}
	return all, nil
}

func axJSONTreeLooksUseful(flat []map[string]any) bool {
	if len(flat) == 0 {
		return false
	}
	usefulRoles := map[string]struct{}{
		"button": {}, "link": {}, "textbox": {}, "TextField": {}, "searchbox": {},
		"checkbox": {}, "radio": {}, "combobox": {}, "listbox": {}, "menuitem": {},
		"heading": {}, "StaticText": {}, "image": {}, "main": {}, "navigation": {},
		"form": {}, "textField": {},
	}
	var walk func(map[string]any) bool
	walk = func(n map[string]any) bool {
		if r, ok := n["role"].(string); ok {
			if _, ok := usefulRoles[r]; ok {
				return true
			}
		}
		raw := n["children"]
		if raw == nil {
			return false
		}
		switch ch := raw.(type) {
		case []map[string]any:
			for _, c := range ch {
				if walk(c) {
					return true
				}
			}
		case []any:
			for _, e := range ch {
				cm, ok := e.(map[string]any)
				if ok && walk(cm) {
					return true
				}
			}
		}
		return false
	}
	for _, root := range flat {
		if walk(root) {
			return true
		}
	}
	return false
}

func injectInteractiveTreeFallback(ctx context.Context, flat []map[string]any) ([]map[string]any, error) {
	if len(flat) == 0 {
		return flat, nil
	}
	var lines string
	if err := chromedp.Run(ctx, chromedp.Evaluate(getScript("interactive_tree"), &lines)); err != nil {
		return flat, err
	}
	lines = strings.TrimSpace(lines)
	if lines == "" {
		return flat, nil
	}
	parts := strings.Split(lines, "\n")
	fallbackChildren := make([]map[string]any, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		fallbackChildren = append(fallbackChildren, map[string]any{
			"role":        "generic",
			"name":        p,
			"description": "dom_interactive_hint",
		})
	}
	if len(fallbackChildren) == 0 {
		return flat, nil
	}
	wrapper := map[string]any{
		"role":     "generic",
		"name":     "dom_interactive_fallback",
		"children": fallbackChildren,
	}
	root := flat[0]
	if raw, ok := root["children"]; ok && raw != nil {
		if ch, ok := raw.([]map[string]any); ok {
			root["children"] = append(ch, wrapper)
			return flat, nil
		}
	}
	root["children"] = []map[string]any{wrapper}
	return flat, nil
}

func prepareAccessibilitySnapshotJSONTree(ctx context.Context) ([]map[string]any, error) {
	var flat []map[string]any
	for attempt := 0; attempt < snapshotPollAttempts; attempt++ {
		nodes, err := fetchFullAXTree(ctx)
		if err != nil {
			return nil, err
		}
		flat = flattenAXTree(nodes)
		if axJSONTreeLooksUseful(flat) {
			break
		}
		if attempt < snapshotPollAttempts-1 {
			if err := chromedp.Run(ctx, chromedp.Sleep(snapshotPollInterval)); err != nil {
				return nil, err
			}
		}
	}
	if !axJSONTreeLooksUseful(flat) {
		var err error
		flat, err = injectInteractiveTreeFallback(ctx, flat)
		if err != nil {
			return nil, err
		}
	}
	return flat, nil
}

func accessibilitySnapshotText(ctx context.Context) (string, error) {
	nodes, err := fetchFullAXTree(ctx)
	if err != nil {
		return "", err
	}
	return formatAccessibilitySnapshotText(nodes), nil
}

func accessibilitySnapshotJSON(ctx context.Context) (string, error) {
	flat, err := prepareAccessibilitySnapshotJSONTree(ctx)
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(flat, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
