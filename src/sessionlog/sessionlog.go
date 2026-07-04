package sessionlog

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"sync"
)

var (
	files sync.Map // chatId -> *os.File
	bound sync.Map // goroutine id -> chatId
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// LogPath returns the session log file path for a conversation.
func LogPath(workingDir, agentName, chatId string) string {
	var baseDir string
	if workingDir != "" {
		baseDir = filepath.Join(workingDir, "data", "conversations", agentName)
	} else {
		baseDir = filepath.Join("data", "conversations", agentName)
	}
	return filepath.Join(baseDir, chatId+".logs")
}

// Ensure opens (or creates) the append-only session log file for chatId.
func Ensure(workingDir, agentName, chatId string) error {
	if chatId == "" || agentName == "" {
		return nil
	}
	if _, ok := files.Load(chatId); ok {
		return nil
	}

	path := LogPath(workingDir, agentName, chatId)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("sessionlog: create dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("sessionlog: open %s: %w", path, err)
	}

	if _, loaded := files.LoadOrStore(chatId, f); loaded {
		_ = f.Close()
	}
	return nil
}

// BindTurn binds the current goroutine to chatId for the duration of a turn.
// Framework logs written while bound are appended to that session's .logs file.
func BindTurn(chatId, workingDir, agentName string) func() {
	if chatId == "" || agentName == "" {
		return func() {}
	}
	_ = Ensure(workingDir, agentName, chatId)

	gid := goroutineID()
	bound.Store(gid, chatId)
	return func() {
		bound.Delete(gid)
	}
}

// Write appends plain text to the session log for chatId.
func Write(chatId, text string) {
	if chatId == "" || text == "" {
		return
	}
	f, ok := files.Load(chatId)
	if !ok {
		return
	}
	file := f.(*os.File)
	_, _ = file.WriteString(StripANSI(text))
}

// WriteBound appends text to the session log bound to the current goroutine.
func WriteBound(text string) {
	if chatId, ok := boundChatID(); ok {
		Write(chatId, text)
	}
}

func boundChatID() (string, bool) {
	v, ok := bound.Load(goroutineID())
	if !ok {
		return "", false
	}
	chatId, ok := v.(string)
	return chatId, ok && chatId != ""
}

// StripANSI removes ANSI escape sequences from s.
func StripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func goroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	i := bytes.IndexByte(b, ' ')
	if i < 0 {
		return 0
	}
	id, _ := strconv.ParseUint(string(b[:i]), 10, 64)
	return id
}
