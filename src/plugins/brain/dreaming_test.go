package brain

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/mocks"
)

// queuedJSONLLM returns one ChatStream completion chunk per call with the next JSON body.
// It avoids mocks.MockLLMEngine's 5-byte streaming, which can truncate JSON before a completion chunk is delivered.
type queuedJSONLLM struct {
	mu    sync.Mutex
	queue []string
}

func (q *queuedJSONLLM) ChatStream(messages []*llms.UnifiedMessage, tools []llms.Tool) *llms.ResponseCh {
	q.mu.Lock()
	var body string
	if len(q.queue) > 0 {
		body = q.queue[0]
		q.queue = q.queue[1:]
	}
	q.mu.Unlock()

	rc := llms.NewResponseCh()
	go func() {
		defer rc.Close()
		payload, err := json.Marshal(llms.ChunkResponse{
			FullContent: body,
			Status:      llms.StatusCompleted,
			Type:        llms.TypeCompletion,
		})
		if err != nil {
			return
		}
		rc.Response <- payload
	}()
	return rc
}

func (q *queuedJSONLLM) Model() string             { return "test-json" }
func (q *queuedJSONLLM) Provider() string          { return "test" }
func (q *queuedJSONLLM) ModelInfo() llms.ModelInfo { return llms.ModelInfo{} }

func TestDistillMemoryMD_NoFile(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	mock := mocks.NewMockLLMEngine()
	plugin.llmEngine = mock
	d := newDreamingRunner(plugin, mock)
	if err := d.distillMemoryMD(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(mock.RecordedCalls) != 0 {
		t.Fatalf("LLM should not run without MEMORY.md, got %d calls", len(mock.RecordedCalls))
	}
}

func TestDistillMemoryMD_EmptyFile(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(plugin.dir, "MEMORY.md"), []byte("  \n"), 0644); err != nil {
		t.Fatal(err)
	}
	mock := mocks.NewMockLLMEngine()
	plugin.llmEngine = mock
	d := newDreamingRunner(plugin, mock)
	if err := d.distillMemoryMD(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(mock.RecordedCalls) != 0 {
		t.Fatal("LLM should not run for whitespace-only MEMORY.md")
	}
}

func TestDistillMemoryMD_CleanupOnly(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(plugin.dir, "MEMORY.md"), []byte("a: b\n\nc: d\n"), 0644); err != nil {
		t.Fatal(err)
	}
	llm := &queuedJSONLLM{queue: []string{`{"cleaned_short_term":"- kept","promote_long_term":false,"summary":"","topics":[],"title":"","description":"","distillation_reason":""}`}}
	plugin.llmEngine = llm
	d := newDreamingRunner(plugin, llm)
	if err := d.distillMemoryMD(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(plugin.dir, "MEMORY.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b)) != "- kept" {
		t.Fatalf("MEMORY.md: %q", string(b))
	}
	matches, _ := filepath.Glob(filepath.Join(plugin.dir, "persistence", "*", "brain-memmd-*.md"))
	if len(matches) != 0 {
		t.Fatalf("unexpected promotion files: %v", matches)
	}
}

func TestDistillMemoryMD_PromotesAtMostOncePerFingerprint(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(plugin.dir, "MEMORY.md"), []byte("topic: fact\n"), 0644); err != nil {
		t.Fatal(err)
	}
	promoJSON := `{"cleaned_short_term":"clean","promote_long_term":true,"summary":"- User prefers Go","topics":["preferences"],"title":"Language preference","description":"User works in Go.","distillation_reason":"Stable preference across sessions for tooling choices."}`
	llm := &queuedJSONLLM{queue: []string{promoJSON, promoJSON}}
	plugin.llmEngine = llm
	d := newDreamingRunner(plugin, llm)
	if err := d.distillMemoryMD(context.Background()); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(plugin.dir, "persistence", "*", "brain-memmd-*.md"))
	if len(matches) != 1 {
		t.Fatalf("wanted 1 promotion file, got %v", matches)
	}
	if err := d.distillMemoryMD(context.Background()); err != nil {
		t.Fatal(err)
	}
	matches2, _ := filepath.Glob(filepath.Join(plugin.dir, "persistence", "*", "brain-memmd-*.md"))
	if len(matches2) != 1 {
		t.Fatalf("wanted still 1 promotion file, got %d", len(matches2))
	}
	synthID := strings.TrimSuffix(filepath.Base(matches2[0]), ".md")
	if !strings.HasPrefix(synthID, syntheticMemoryConvIDPrefix) {
		t.Fatalf("expected synthetic prefix, got %q", synthID)
	}
	node, err := plugin.getConversationNodeByConvID(synthID)
	if err != nil {
		t.Fatal(err)
	}
	// Content is cleared; verify the .md file has the summary.
	mdContent, err := os.ReadFile(matches2[0])
	if err != nil {
		t.Fatalf("read promotion file: %v", err)
	}
	if !strings.Contains(string(mdContent), "Go") {
		t.Fatalf("promotion file content: %q", string(mdContent))
	}
	meta := node.Metadata
	if meta["source"] != "memory_md" {
		t.Fatalf("expected source memory_md, got %v", meta["source"])
	}
}

func TestRecategorizePendingOnly_MovesOffPending(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	convID := "11111111-1111-1111-1111-111111111111"
	if err := plugin.upsertConversationNode(convID, time.Now()); err != nil {
		t.Fatal(err)
	}
	dreamed := time.Now().UTC().Truncate(time.Second)
	if err := plugin.updateConversationNodeSummary(convID,
		"- User prefers Go for services",
		[]string{defaultConversationTopic},
		"Go preference",
		"The user asked to standardize on Go.",
		"Stable tooling preference across sessions.",
		dreamed,
	); err != nil {
		t.Fatal(err)
	}

	llm := &queuedJSONLLM{queue: []string{`{"topics":["preferences"]}`}}
	plugin.llmEngine = llm
	d := newDreamingRunner(plugin, llm)
	if err := d.recategorizePendingOnly(context.Background()); err != nil {
		t.Fatal(err)
	}

	node, err := plugin.getConversationNodeByConvID(convID)
	if err != nil {
		t.Fatal(err)
	}
	topics, err := plugin.getTopicsForConversationNodeID(node.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 1 {
		t.Fatalf("expected 1 topic edge, got %d", len(topics))
	}
	if normalizeTopicName(topics[0].GetTitle()) != "preferences" {
		t.Fatalf("topic: %q", topics[0].GetTitle())
	}
}

func TestRunPending_RunsMemoryPhaseWhenNoPendingJSON(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	if err := os.WriteFile(filepath.Join(plugin.dir, "MEMORY.md"), []byte("note: hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	llm := &queuedJSONLLM{queue: []string{`{"cleaned_short_term":"hello","promote_long_term":false,"summary":"","topics":[],"title":"","description":"","distillation_reason":""}`}}
	plugin.llmEngine = llm
	d := newDreamingRunner(plugin, llm)
	if err := d.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(plugin.dir, "MEMORY.md"))
	if strings.TrimSpace(string(b)) != "hello" {
		t.Fatalf("MEMORY.md: %q", string(b))
	}
}

func TestSyntheticMemoryPromotionConvID_Deterministic(t *testing.T) {
	fp := promotionFingerprint("s", "t", "d", "r", []string{"a", "b"})
	id1 := syntheticMemoryPromotionConvID(fp)
	id2 := syntheticMemoryPromotionConvID(fp)
	if id1 != id2 {
		t.Fatalf("ids differ: %q vs %q", id1, id2)
	}
	if !strings.HasPrefix(id1, syntheticMemoryConvIDPrefix) {
		t.Fatalf("prefix: %q", id1)
	}
}

func TestReadWriteMemoryMD_Locked(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	if err := plugin.writeMemoryMDFull("alpha"); err != nil {
		t.Fatal(err)
	}
	raw, err := plugin.readMemoryMDRaw()
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "alpha" {
		t.Fatalf("read: %q", raw)
	}
	if plugin.readMemoryMDForInjection() != "alpha" {
		t.Fatalf("injection: %q", plugin.readMemoryMDForInjection())
	}
}

func TestWriteMemoryMDFullIfUnchanged_SkipsStaleRewrite(t *testing.T) {
	plugin, cleanup := setupTestPlugin(t)
	defer cleanup()

	if err := plugin.writeMemoryMDFull("alpha"); err != nil {
		t.Fatal(err)
	}
	if err := plugin.writeMemoryMDFull("beta"); err != nil {
		t.Fatal(err)
	}
	wrote, err := plugin.writeMemoryMDFullIfUnchanged([]byte("alpha"), "gamma")
	if err != nil {
		t.Fatal(err)
	}
	if wrote {
		t.Fatal("expected stale compare-and-swap write to be skipped")
	}
	raw, err := plugin.readMemoryMDRaw()
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "beta" {
		t.Fatalf("expected latest content to win, got %q", raw)
	}
}
