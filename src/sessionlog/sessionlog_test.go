package sessionlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogPath(t *testing.T) {
	got := LogPath("/work", "Smith", "abc-123")
	want := filepath.Join("/work", "data", "conversations", "Smith", "abc-123.logs")
	if got != want {
		t.Fatalf("LogPath() = %q, want %q", got, want)
	}

	got = LogPath("", "Smith", "abc-123")
	want = filepath.Join("data", "conversations", "Smith", "abc-123.logs")
	if got != want {
		t.Fatalf("LogPath() empty working dir = %q, want %q", got, want)
	}
}

func TestEnsureWriteBindTurn(t *testing.T) {
	dir := t.TempDir()
	chatId := "conv-1"
	agentName := "Smith"

	release := BindTurn(chatId, dir, agentName)
	defer release()

	WriteBound("[INFO] hello\n")
	Write(chatId, "plain line\n")
	release()

	path := LogPath(dir, agentName, chatId)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read session log: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "[INFO] hello") {
		t.Fatalf("expected bound write in log, got: %q", body)
	}
	if !strings.Contains(body, "plain line") {
		t.Fatalf("expected direct write in log, got: %q", body)
	}
}

func TestStripANSI(t *testing.T) {
	in := "\033[36mhello\033[0m world"
	if got := StripANSI(in); got != "hello world" {
		t.Fatalf("StripANSI() = %q", got)
	}
}
