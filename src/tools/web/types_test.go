package web

import (
	"strings"
	"testing"
)

func TestWebTypes_String(t *testing.T) {
	// Navigate
	navResp := &navigateResponse{
		Operation: "navigate",
		URL:       "http://example.com",
		Title:     "Example",
		Success:   true,
		Error:     "",
	}
	navStr := navResp.String()
	if !strings.Contains(navStr, "Web Browser Operation: Navigate") {
		t.Error("Navigate String() mismatch")
	}
	if !strings.Contains(navStr, "Example") {
		t.Error("Navigate String() missing title")
	}

	// Navigate Error
	navErr := &navigateResponse{
		Success: false,
		Error:   "timeout",
	}
	if !strings.Contains(navErr.String(), "Error: timeout") {
		t.Error("Navigate String() error mismatch")
	}

	// Click
	clickResp := &clickResponse{
		Operation: "click",
		Selector:  "#btn",
		Success:   true,
	}
	if !strings.Contains(clickResp.String(), "Web Browser Operation: Click") {
		t.Error("Click String() mismatch")
	}

	// Click Error
	clickErr := &clickResponse{
		Success: false,
		Error:   "not found",
	}
	if !strings.Contains(clickErr.String(), "Error: not found") {
		t.Error("Click String() error mismatch")
	}

	// Content
	contentResp := &contentResponse{
		Operation: "content",
		Type:      "text",
		Content:   "some content",
		Success:   true,
	}
	if !strings.Contains(contentResp.String(), "Web Browser Operation: Get Content") {
		t.Error("Content String() mismatch")
	}
	if !strings.Contains(contentResp.String(), "some content") {
		t.Error("Content String() missing content")
	}

	// Content Long
	longContent := &contentResponse{
		Content: strings.Repeat("a", 600),
		Success: true,
	}
	if !strings.Contains(longContent.String(), "(truncated)") {
		t.Error("Content String() truncation mismatch")
	}

	// Save Content
	saveResp := &saveContentResponse{
		Operation: "save",
		Filename:  "test.txt",
		Path:      "/tmp/test.txt",
		Success:   true,
	}
	saveStr := saveResp.String()
	if !strings.Contains(saveStr, "Web Browser Operation: Save Content") {
		t.Error("Save String() mismatch")
	}
	if !strings.Contains(saveStr, "test.txt") {
		t.Error("Save String() missing filename")
	}

	// Save Error
	saveErr := &saveContentResponse{
		Success: false,
		Error:   "permission denied",
	}
	if !strings.Contains(saveErr.String(), "Error: permission denied") {
		t.Error("Save String() error mismatch")
	}
}
