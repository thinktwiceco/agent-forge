package web

import (
	"encoding/json"
	"testing"

	"github.com/chromedp/cdproto/accessibility"
)

func TestAXTreeResponse_AllowsUnknownPropertyNames(t *testing.T) {
	payload := []byte(`{
		"nodes": [
			{
				"nodeId": "1",
				"ignored": false,
				"role": {"value": "RootWebArea"},
				"name": {"value": "Example"},
				"childIds": ["2"]
			},
			{
				"nodeId": "2",
				"parentId": "1",
				"ignored": false,
				"role": {"value": "button"},
				"name": {"value": "Submit"},
				"ignoredReasons": [
					{"name": "uninteresting", "value": {"value": true}}
				],
				"properties": [
					{"name": "focused", "value": {"value": true}},
					{"name": "futureProp", "value": {"value": "kept-tolerant"}}
				]
			}
		]
	}`)

	var res axTreeResponse
	if err := json.Unmarshal(payload, &res); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(res.Nodes) != 2 {
		t.Fatalf("len(res.Nodes) = %d, want 2", len(res.Nodes))
	}

	got := flattenAXTree(res.Nodes)
	if len(got) != 1 {
		t.Fatalf("len(flattenAXTree(...)) = %d, want 1 root", len(got))
	}

	children, ok := got[0]["children"].([]map[string]any)
	if !ok || len(children) != 1 {
		t.Fatalf("root children = %#v, want one child", got[0]["children"])
	}

	if children[0]["role"] != "button" {
		t.Fatalf("child role = %#v, want %q", children[0]["role"], "button")
	}
	if children[0]["name"] != "Submit" {
		t.Fatalf("child name = %#v, want %q", children[0]["name"], "Submit")
	}
	if children[0]["focused"] != "true" {
		t.Fatalf("child focused = %#v, want %q", children[0]["focused"], "true")
	}
}

func TestRemapAXNodeIDs(t *testing.T) {
	nodes := []*axNode{
		{NodeID: "1", ChildIDs: []accessibility.NodeID{"2"}},
		{NodeID: "2", ParentID: "1"},
	}
	remapAXNodeIDs(nodes, "mf0:")
	if string(nodes[0].NodeID) != "mf0:1" || string(nodes[1].NodeID) != "mf0:2" {
		t.Fatalf("node ids not remapped: %#v, %#v", nodes[0].NodeID, nodes[1].NodeID)
	}
	if string(nodes[1].ParentID) != "mf0:1" {
		t.Fatalf("parent id = %q, want mf0:1", nodes[1].ParentID)
	}
	if len(nodes[0].ChildIDs) != 1 || string(nodes[0].ChildIDs[0]) != "mf0:2" {
		t.Fatalf("child ids = %#v", nodes[0].ChildIDs)
	}
}

func TestRemapAXNodeIDs_EmptyPrefixNoOp(t *testing.T) {
	nodes := []*axNode{{NodeID: "1", ChildIDs: []accessibility.NodeID{"2"}}, {NodeID: "2", ParentID: "1"}}
	remapAXNodeIDs(nodes, "")
	if string(nodes[0].NodeID) != "1" {
		t.Fatalf("expected no remap, got %q", nodes[0].NodeID)
	}
}

func TestAxJSONTreeLooksUseful(t *testing.T) {
	if axJSONTreeLooksUseful([]map[string]any{{"role": "RootWebArea"}}) {
		t.Fatal("bare RootWebArea should not be useful")
	}
	if !axJSONTreeLooksUseful([]map[string]any{{
		"role": "RootWebArea",
		"children": []map[string]any{
			{"role": "generic", "children": []map[string]any{{"role": "textbox", "name": "u"}}},
		},
	}}) {
		t.Fatal("nested textbox should be useful")
	}
}
