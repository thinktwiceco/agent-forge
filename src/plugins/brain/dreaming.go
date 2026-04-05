package brain

// ─── DreamingRunner ───────────────────────────────────────────────────────────
//
// DreamingRunner distils raw conversation JSON files into concise memory notes, then
// optionally cleans brain/MEMORY.md and may promote at most one durable line into the graph.
//
// Architecture overview:
//
//	data/conversations/<agentName>/<conv_id>.json  ← raw chat history (unchanged)
//	        │
//	        │  DreamingRunner.RunPending()
//	        │  1. getDreamedIDs()            — find conv_ids already distilled
//	        │  2. getPendingConversations()  — find undistilled ones within threshold
//	        │  3. distillConversation()      — LLM call → write markdown + graph node
//	        │  4. recategorizePendingOnly() — LLM reassigns topics for nodes only on [pending]
//	        │  5. distillMemoryMD()          — LLM cleanup of MEMORY.md; optional synthetic node
//	        ▼
//	brain/persistence/YYYY-MM-DD/<conv_id>.md      ← distilled notes
//	        │
//	        │  updateConversationNodeSummary()
//	        ▼
//	brain.db conversation node                     ← graph index with summary content
//
// Idempotency: presence of brain/persistence/<date>/<conv_id>.md means that
// conversation is considered dreamed; it is never processed again.
//
// Threshold: conversations whose JSON file is older than DreamingThreshold are
// ignored — they are either already dreamed or too stale to be worth processing.
//
// Triggers (production): RunPending is only reached via runDreaming — (1) on the daily
// brain_plugin.dreamTime timer, (2) the dream tool handler. Nothing else should call into this pipeline.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const dreamingNothing = "NOTHING_TO_DISTILL"

// syntheticMemoryConvIDPrefix marks graph nodes created from MEMORY.md promotion.
// Real session conv_ids are UUIDs; this prefix avoids collisions and keeps getPendingConversations
// from treating these nodes as JSON transcripts to distill.
const syntheticMemoryConvIDPrefix = "brain-memmd-"

// DreamingThreshold is the maximum age of a conversation JSON file that will be
// considered for dreaming. Conversations older than this are ignored — they are
// either already dreamed or too stale to be worth processing.
const DreamingThreshold = 30 * 24 * time.Hour

// rawMessage is a minimal struct for parsing conversation JSON without pulling in
// history or persistence packages. Only role and content are needed for distillation.
type rawMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// pendingConversation holds the metadata needed to distil one conversation.
type pendingConversation struct {
	convID   string
	jsonPath string
	modTime  time.Time // used as the persistence date (YYYY-MM-DD folder)
}

type distilledConversation struct {
	Summary            string   `json:"summary"`
	Topics             []string `json:"topics"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	DistillationReason string   `json:"distillation_reason"`
}

// memoryMDDistillResult is the LLM output for the MEMORY.md cleanup phase (separate from transcript distillation).
type memoryMDDistillResult struct {
	CleanedShortTerm   string   `json:"cleaned_short_term"`
	PromoteLongTerm    bool     `json:"promote_long_term"`
	Summary            string   `json:"summary"`
	Topics             []string `json:"topics"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	DistillationReason string   `json:"distillation_reason"`
}

// DreamingRunner reads conversation JSON files and distils them into memory notes.
type DreamingRunner struct {
	plugin    *BrainPlugin
	llmEngine llms.LLMEngine
}

// newDreamingRunner constructs a runner bound to the given plugin and LLM engine.
func newDreamingRunner(p *BrainPlugin, engine llms.LLMEngine) *DreamingRunner {
	return &DreamingRunner{plugin: p, llmEngine: engine}
}

// runDreaming executes pending distillation. Holds dreamMu so scheduled runs and the dream
// tool never execute RunPending concurrently.
func (p *BrainPlugin) runDreaming(ctx context.Context) error {
	p.dreamMu.Lock()
	defer p.dreamMu.Unlock()
	if p.workingDir == "" {
		return fmt.Errorf("brain: workingDir not set; cannot run dreaming")
	}
	if p.llmEngine == nil {
		return fmt.Errorf("brain: LLM engine not available")
	}
	return newDreamingRunner(p, p.llmEngine).RunPending(ctx)
}

// getDreamedIDs returns the set of conversation IDs that already have a distilled
// file under brain/persistence/. Used to skip already-processed conversations.
func (d *DreamingRunner) getDreamedIDs() (map[string]bool, error) {
	result := make(map[string]bool)
	persistenceDir := filepath.Join(d.plugin.dir, "persistence")

	dateDirs, err := os.ReadDir(persistenceDir)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("brain: could not read persistence directory: %w", err)
	}

	for _, dateDir := range dateDirs {
		if !dateDir.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(persistenceDir, dateDir.Name()))
		if err != nil {
			agentforge.Debug("🧠 [Dreaming] Warning: could not read date dir %s: %v", dateDir.Name(), err)
			continue
		}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".md") {
				result[strings.TrimSuffix(entry.Name(), ".md")] = true
			}
		}
	}

	return result, nil
}

// getPendingConversations returns all conversations that have not yet been dreamed
// and whose JSON file was modified within the given threshold.
func (d *DreamingRunner) getPendingConversations(threshold time.Duration) ([]pendingConversation, error) {
	dreamed, err := d.getDreamedIDs()
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-threshold)
	convRoot := filepath.Join(d.plugin.workingDir, "data", "conversations")

	agentDirs, err := os.ReadDir(convRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("brain: could not read conversations directory: %w", err)
	}

	var pending []pendingConversation
	for _, agentEntry := range agentDirs {
		if !agentEntry.IsDir() {
			continue
		}
		agentDir := filepath.Join(convRoot, agentEntry.Name())
		entries, err := os.ReadDir(agentDir)
		if err != nil {
			agentforge.Debug("🧠 [Dreaming] Warning: could not read agent dir %s: %v", agentDir, err)
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			convID := strings.TrimSuffix(entry.Name(), ".json")

			if dreamed[convID] {
				continue // already processed
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				continue // too old; skip
			}

			pending = append(pending, pendingConversation{
				convID:   convID,
				jsonPath: filepath.Join(agentDir, entry.Name()),
				modTime:  info.ModTime(),
			})
		}
	}

	return pending, nil
}

// RunPending distils all conversations that have not yet been dreamed and are
// within DreamingThreshold. It is the primary entry point for the background goroutine.
func (d *DreamingRunner) RunPending(ctx context.Context) error {
	if d.plugin.workingDir == "" {
		return fmt.Errorf("brain: workingDir not set; cannot run dreaming")
	}

	pending, err := d.getPendingConversations(DreamingThreshold)
	if err != nil {
		return fmt.Errorf("brain: failed to enumerate pending conversations: %w", err)
	}

	if len(pending) == 0 {
		agentforge.Debug("🧠 [Dreaming] No pending conversations to process")
	} else {
		agentforge.Debug("🧠 [Dreaming] Processing %d pending conversation(s)", len(pending))
		for _, p := range pending {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err := d.distillConversation(ctx, p.convID, p.jsonPath, p.modTime); err != nil {
				agentforge.Debug("🧠 [Dreaming] Failed to distil %s: %v", p.convID, err)
				// Non-fatal; continue with remaining conversations.
			}
		}
	}

	if err := d.recategorizePendingOnly(ctx); err != nil {
		agentforge.Debug("🧠 [Dreaming] Pending recategorize phase: %v", err)
	}

	if err := d.distillMemoryMD(ctx); err != nil {
		agentforge.Debug("🧠 [Dreaming] MEMORY.md phase: %v", err)
	}
	return nil
}

// promotionFingerprint builds a stable string from promotion fields for deterministic synthetic conv ids.
func promotionFingerprint(summary, title, description, distillationReason string, topics []string) string {
	normTopics := normalizeTopicNames(topics)
	sort.Strings(normTopics)
	var b strings.Builder
	b.WriteString(strings.TrimSpace(summary))
	b.WriteByte('\n')
	b.WriteString(strings.TrimSpace(title))
	b.WriteByte('\n')
	b.WriteString(strings.TrimSpace(description))
	b.WriteByte('\n')
	b.WriteString(strings.TrimSpace(distillationReason))
	b.WriteByte('\n')
	b.WriteString(strings.Join(normTopics, ","))
	return b.String()
}

// syntheticMemoryPromotionConvID returns a reserved-prefix id so unchanged promotion payloads map to one node.
func syntheticMemoryPromotionConvID(fingerprint string) string {
	sum := sha256.Sum256([]byte(fingerprint))
	return syntheticMemoryConvIDPrefix + hex.EncodeToString(sum[:])
}

// distillMemoryMD cleans brain/MEMORY.md and optionally promotes at most one durable memory into the graph.
func (d *DreamingRunner) distillMemoryMD(ctx context.Context) error {
	if d.plugin == nil || d.plugin.workingDir == "" {
		return nil
	}
	if d.llmEngine == nil {
		return nil
	}

	rawBytes, err := d.plugin.readMemoryMDRaw()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read MEMORY.md: %w", err)
	}
	raw := string(rawBytes)
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	out, err := d.callMemoryMDDistillLLM(ctx, raw)
	if err != nil {
		return fmt.Errorf("MEMORY.md LLM distillation: %w", err)
	}
	if out == nil {
		return nil
	}

	cleaned := strings.TrimSpace(out.CleanedShortTerm)
	wrote, err := d.plugin.writeMemoryMDFullIfUnchanged(rawBytes, cleaned)
	if err != nil {
		return fmt.Errorf("write MEMORY.md: %w", err)
	}
	if !wrote {
		agentforge.Debug("🧠 [Dreaming] MEMORY.md changed during distillation; skipping cleanup rewrite and promotion")
		return nil
	}
	agentforge.Debug("🧠 [Dreaming] Rewrote MEMORY.md (%d bytes cleaned)", len(cleaned))

	if !out.PromoteLongTerm {
		return nil
	}

	summary := strings.TrimSpace(out.Summary)
	title := strings.TrimSpace(out.Title)
	description := strings.TrimSpace(out.Description)
	distillationReason := strings.TrimSpace(out.DistillationReason)
	topics := normalizeTopicNames(out.Topics)

	if summary == "" || summary == dreamingNothing || title == "" || description == "" || distillationReason == "" {
		agentforge.Debug("🧠 [Dreaming] MEMORY.md promotion skipped (incomplete promotion fields)")
		return nil
	}

	fp := promotionFingerprint(summary, title, description, distillationReason, topics)
	synthID := syntheticMemoryPromotionConvID(fp)

	dreamed, err := d.getDreamedIDs()
	if err != nil {
		return fmt.Errorf("get dreamed ids: %w", err)
	}
	if dreamed[synthID] {
		agentforge.Debug("🧠 [Dreaming] MEMORY.md promotion already persisted for synthetic id (idempotent skip)")
		return nil
	}

	now := time.Now()
	if d.plugin.db != nil {
		if err := d.plugin.upsertConversationNodeWithMetadata(synthID, now, map[string]any{"source": "memory_md"}); err != nil {
			return fmt.Errorf("upsert synthetic conversation node: %w", err)
		}
		if err := d.plugin.writeMemoryPromotionPersistence(synthID, now, title, description, distillationReason, topics, summary); err != nil {
			agentforge.Debug("🧠 [Dreaming] Warning: could not write promotion persistence for %s: %v", synthID, err)
		}
		if err := d.plugin.updateConversationNodeSummary(synthID, summary, topics, title, description, distillationReason, now); err != nil {
			return fmt.Errorf("update synthetic node summary: %w", err)
		}
		agentforge.Debug("🧠 [Dreaming] Promoted MEMORY.md → graph node %s", synthID)
	}

	return nil
}

// callMemoryMDDistillLLM asks the model to compress MEMORY.md and optionally emit one long-term bundle.
func (d *DreamingRunner) callMemoryMDDistillLLM(ctx context.Context, memoryBody string) (*memoryMDDistillResult, error) {
	currentTopicsLine := d.rulesSectionCurrentTopicsLine()
	systemMsg := llms.SystemMessage(`You are a short-term working memory editor for brain/MEMORY.md.
Your job:
1. Produce [cleaned_short_term]: remove redundant, stale, or low-signal lines; keep useful daily context in compact markdown or short lines.
2. Decide [promote_long_term]: true only if there is exactly one clearly durable fact or preference that should survive across many sessions (not routine daily noise). Otherwise false.
3. When [promote_long_term] is true, fill summary, topics, title, description, distillation_reason for that single item (at most one). When false, leave those strings empty and topics [].

Return strict JSON:
{"cleaned_short_term":"markdown or plain text","promote_long_term":false,"summary":"","topics":[],"title":"","description":"","distillation_reason":""}

Rules:
` + currentTopicsLine + `- Topic assignment: if promoting, review CURRENT TOPICS first; reuse exact lowercase labels when they fit. Only invent new topic labels when needed.
- summary: markdown bullets or short paragraph for the promoted item only (required when promote_long_term is true).
- topics: 1-5 lowercase labels when promoting; otherwise [].
- title, description, distillation_reason: required when promote_long_term is true; same quality rules as session distillation (why this matters for future sessions).
- Never set promote_long_term true without all of summary, title, description, distillation_reason non-empty.
- If nothing should be promoted, promote_long_term must be false.`)

	userMsg := llms.UserMessage("CURRENT brain/MEMORY.md:\n\n" + memoryBody)

	responseCh := d.llmEngine.ChatStream([]*llms.UnifiedMessage{systemMsg, userMsg}, nil)

	var fullContent string
	for chunk := range responseCh.Start() {
		if chunk.Status == llms.StatusError {
			return nil, fmt.Errorf("llm stream error: %s", chunk.Content)
		}
		if len(chunk.FullContent) > len(fullContent) {
			fullContent = chunk.FullContent
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	fullContent = strings.TrimSpace(fullContent)
	fullContent = strings.TrimPrefix(fullContent, "```json")
	fullContent = strings.TrimPrefix(fullContent, "```")
	fullContent = strings.TrimSuffix(fullContent, "```")
	fullContent = strings.TrimSpace(fullContent)

	var out memoryMDDistillResult
	if err := json.Unmarshal([]byte(fullContent), &out); err != nil {
		return nil, fmt.Errorf("parse MEMORY.md distillation JSON: %w", err)
	}
	out.CleanedShortTerm = strings.TrimSpace(out.CleanedShortTerm)
	out.Summary = strings.TrimSpace(out.Summary)
	out.Title = strings.TrimSpace(out.Title)
	out.Description = strings.TrimSpace(out.Description)
	out.DistillationReason = strings.TrimSpace(out.DistillationReason)
	out.Topics = normalizeTopicNames(out.Topics)
	return &out, nil
}

// distillConversation reads one conversation JSON file, calls the LLM to extract
// key points, and writes the result to brain/persistence/<date>/<conv_id>.md.
// It also updates the conversation graph node's content with the summary.
func (d *DreamingRunner) distillConversation(ctx context.Context, convID, jsonPath string, date time.Time) error {
	// Parse the conversation JSON.
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read conversation JSON: %w", err)
	}
	var msgs []rawMessage
	if err := json.Unmarshal(data, &msgs); err != nil {
		return fmt.Errorf("parse conversation JSON: %w", err)
	}

	// Build a clean transcript (user + assistant only; skip system and tool messages).
	transcript := buildTranscript(msgs)
	if transcript == "" {
		agentforge.Debug("🧠 [Dreaming] Skipping empty transcript for %s", convID)
		return nil
	}

	// Call the LLM to distil key points and topics.
	distilled, err := d.callDistillLLM(ctx, transcript)
	if err != nil {
		return fmt.Errorf("LLM distillation failed: %w", err)
	}
	if distilled == nil || strings.TrimSpace(distilled.Summary) == dreamingNothing || strings.TrimSpace(distilled.Summary) == "" {
		agentforge.Debug("🧠 [Dreaming] Nothing to distil for %s", convID)
		return nil
	}

	title := strings.TrimSpace(distilled.Title)
	description := strings.TrimSpace(distilled.Description)
	distillationReason := strings.TrimSpace(distilled.DistillationReason)
	if title == "" || description == "" || distillationReason == "" {
		agentforge.Debug("🧠 [Dreaming] Incomplete distillation (title, description, and distillation_reason required) for %s; skipping save", convID)
		return nil
	}

	topics := normalizeTopicNames(distilled.Topics)

	// Write the distilled file.
	destDir, err := d.plugin.distilledDir(date)
	if err != nil {
		return fmt.Errorf("create persistence dir: %w", err)
	}
	destPath := filepath.Join(destDir, convID+".md")
	content := fmt.Sprintf("# Distilled — %s\n\nConversation: %s\n\n## Title\n\n%s\n\n## Description\n\n%s\n\n## Distillation reason\n\n%s\n\nTopics: %s\n\n## Summary\n\n%s\n",
		date.Format("2006-01-02"), convID, title, description, distillationReason, strings.Join(topics, ", "), distilled.Summary)
	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write distilled file: %w", err)
	}

	agentforge.Debug("🧠 [Dreaming] Distilled %s → %s", convID, destPath)

	// Ensure the skeleton node exists before updating it — covers the case where brain.db was
	// wiped or recreated after the conversation was already started (or never indexed).
	if d.plugin.db != nil {
		if err := d.plugin.upsertConversationNode(convID, date); err != nil {
			agentforge.Debug("🧠 [Dreaming] Warning: could not upsert graph node for %s: %v", convID, err)
		}
		if err := d.plugin.updateConversationNodeSummary(convID, distilled.Summary, topics, title, description, distillationReason, date); err != nil {
			agentforge.Debug("🧠 [Dreaming] Warning: could not update graph node for %s: %v", convID, err)
		}
	}

	return nil
}

// rulesSectionCurrentTopicsLine lists normalized topic names from the graph for the dreaming Rules section.
func (d *DreamingRunner) rulesSectionCurrentTopicsLine() string {
	if d.plugin == nil || d.plugin.db == nil {
		return "- CURRENT TOPICS: (none yet)\n"
	}
	nodes, err := d.plugin.getTopicsUnderRoot()
	if err != nil {
		agentforge.Debug("🧠 [Dreaming] Warning: could not load graph topics for prompt: %v", err)
		return "- CURRENT TOPICS: (unavailable)\n"
	}
	seen := make(map[string]bool)
	var names []string
	for _, n := range nodes {
		t := normalizeTopicName(n.GetTitle())
		if t == "" || t == defaultConversationTopic || seen[t] {
			continue
		}
		seen[t] = true
		names = append(names, t)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "- CURRENT TOPICS: (none yet)\n"
	}
	return "- CURRENT TOPICS: " + strings.Join(names, ", ") + "\n"
}

// callDistillLLM sends a distillation prompt to the LLM and collects the full response.
func (d *DreamingRunner) callDistillLLM(ctx context.Context, transcript string) (*distilledConversation, error) {
	currentTopicsLine := d.rulesSectionCurrentTopicsLine()
	systemMsg := llms.SystemMessage(`You are a memory distillation assistant. Be strict: most chats should NOT be retained.

Only retain content that clearly satisfies at least one criterion (skip greetings, filler, tests, one-off small talk, and routine tool noise):
1. User agent instructions or standing preferences (what the user told the agent to do, be, or prioritize across sessions).
2. Durable decisions or outcomes (commitments, conclusions, or facts that should influence future behavior).
3. Stable user preferences (tone, format, products, constraints) that recur or matter beyond this chat.

Return strict JSON with this shape:
{"summary":"markdown bullet list","topics":["topic one","topic two"],"title":"short listing title","description":"one or two sentences for recall lists","distillation_reason":"why this conversation is worth remembering across sessions"}

Field contract (when retaining — all must align; no padding):
` + currentTopicsLine + `- Topic assignment: use CURRENT TOPICS labels first (exact spelling, lowercase). New labels only if nothing fits.
- summary: the dense, canonical memory body. Markdown bullets; every bullet must map to criterion 1, 2, or 3. This is what gets searched and shown as "content" — it must not be weaker or shorter than description; include the same substance the user would need to recall the session.
- title: 3–12 words; browser heading only (no bullets).
- description: one or two plain sentences for recall lists; must add detail vs title, not repeat title wording.
- distillation_reason: 1–3 sentences naming which criterion (1, 2, or 3) applies and why this belongs in long-term memory. No restating the full summary.
- topics: 1–5 lowercase labels for what you kept.
- If nothing qualifies or you cannot justify all fields with the same substance, output exactly:
{"summary":"` + dreamingNothing + `","topics":[],"title":"","description":"","distillation_reason":""}`)

	userMsg := llms.UserMessage("CONVERSATION:\n\n" + transcript)

	responseCh := d.llmEngine.ChatStream([]*llms.UnifiedMessage{systemMsg, userMsg}, nil)

	var fullContent string
	for chunk := range responseCh.Start() {
		if chunk.Status == llms.StatusError {
			return nil, fmt.Errorf("llm stream error: %s", chunk.Content)
		}
		if len(chunk.FullContent) > len(fullContent) {
			fullContent = chunk.FullContent
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	fullContent = strings.TrimSpace(fullContent)
	fullContent = strings.TrimPrefix(fullContent, "```json")
	fullContent = strings.TrimPrefix(fullContent, "```")
	fullContent = strings.TrimSuffix(fullContent, "```")
	fullContent = strings.TrimSpace(fullContent)

	var distilled distilledConversation
	if err := json.Unmarshal([]byte(fullContent), &distilled); err != nil {
		return nil, fmt.Errorf("parse distillation JSON: %w", err)
	}
	distilled.Summary = strings.TrimSpace(distilled.Summary)
	distilled.Title = strings.TrimSpace(distilled.Title)
	distilled.Description = strings.TrimSpace(distilled.Description)
	distilled.DistillationReason = strings.TrimSpace(distilled.DistillationReason)
	distilled.Topics = normalizeTopicNames(distilled.Topics)
	return &distilled, nil
}

// buildTranscript formats user and assistant messages into a readable string for the LLM prompt.
func buildTranscript(msgs []rawMessage) string {
	var sb strings.Builder
	for _, m := range msgs {
		switch m.Role {
		case "user":
			// Strip the metadata header injected by the queue formatter (sender/timestamp lines).
			content := stripMessageHeader(m.Content)
			if content == "" {
				continue
			}
			sb.WriteString("User: ")
			sb.WriteString(content)
			sb.WriteString("\n\n")
		case "assistant":
			if m.Content == "" {
				continue // tool-call-only turn
			}
			sb.WriteString("Assistant: ")
			sb.WriteString(m.Content)
			sb.WriteString("\n\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

// stripMessageHeader removes the "sender: …\ntimestamp: …\n\n" prefix that the
// queue formatter prepends to user messages, leaving only the actual text.
func stripMessageHeader(content string) string {
	lines := strings.SplitN(content, "\n", 4)
	// Header pattern: "sender: …", "timestamp: …", "" (blank), then body.
	if len(lines) >= 4 &&
		strings.HasPrefix(lines[0], "sender:") &&
		strings.HasPrefix(lines[1], "timestamp:") &&
		lines[2] == "" {
		return strings.TrimSpace(lines[3])
	}
	return strings.TrimSpace(content)
}
