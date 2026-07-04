import { MarkdownRenderer, renderMarkdown } from "./markdown.js";
import { formatToolCallSummary, formatToolResultSummary } from "./tool-format.js";

// One "turn" = one persistent container for content and tool calls from a single request.
export class TurnManager {
  constructor(host) {
    this.host = host;
    this.activeTurns = new Map();
  }

  clear() {
    this.activeTurns.clear();
  }

  deleteTurn(chatId) {
    this.activeTurns.delete(chatId);
  }

  flushAllMarkdown() {
    for (const turn of this.activeTurns.values()) {
      turn.markdownRenderer?.flush(turn.rawText);
    }
  }

  renderHistoryMessage(msg, { insertBeforeEl = null } = {}) {
    const role = msg.role || "assistant";
    if (role === "system") return;

    if (role === "user") {
      if (!insertBeforeEl) this.clear();
      let content = msg.content || "";
      const sep = content.indexOf("\n\n");
      if (sep !== -1 && /^sender:/m.test(content.slice(0, sep))) {
        const headers = content.slice(0, sep);
        const showAsYou =
          /^sender:\s*user\s*$/m.test(headers) ||
          /^sender:\s*webhook\s*$/m.test(headers);
        if (!showAsYou) return;
        content = content.slice(sep + 2);
      }
      this.host.appendUserMessage(content, [], insertBeforeEl);
      return;
    }

    const agentName = msg.name || "Assistant";

    if (role === "assistant") {
      if (msg.content) {
        this.handleContentEvent({ agentName, content: msg.content }, insertBeforeEl);
      }
      const toolCalls = msg.toolCalls || msg.tool_calls;
      if (toolCalls && toolCalls.length > 0) {
        this.handleToolCallEvent({ agentName, toolCalls }, insertBeforeEl);
      }
      return;
    }

    if (role === "tool") {
      let isError = false;
      let resultText = msg.content || "";
      if (typeof resultText === "string" && resultText.startsWith("Error:")) {
        isError = true;
        resultText = resultText.substring(6).trim();
      }
      this.handleToolResultEvent({
        agentName,
        toolResults: [{
          toolName: msg.toolName || msg.name || "Tool",
          success: !isError,
          result: resultText,
          error: isError ? resultText : undefined,
        }],
      }, insertBeforeEl);
    }
  }

  handleContentEvent(payload, insertBeforeEl = null) {
    const delta = payload.delta || payload.content || "";
    if (!delta) return;

    this.host.removeThinking();

    const chatId = payload.chatId || this.host.getConversationId() || "";
    const label = this.host.formatAgentLabel(payload);
    const turn = this._ensureTurn(chatId, label, insertBeforeEl);

    turn.rawText += delta;
    if (payload.content && !payload.delta) {
      turn.markdownRenderer.flush(turn.rawText);
    } else {
      turn.markdownRenderer.schedule(turn.rawText);
    }
  }

  handleToolCallEvent(payload, insertBeforeEl = null) {
    this.host.removeThinking();

    const chatId = payload.chatId || this.host.getConversationId() || "";
    const label = this.host.formatAgentLabel(payload);
    const turn = this._ensureToolGroup(chatId, label, insertBeforeEl);
    const calls = payload.toolCalls || [];

    calls.forEach((call, i) => {
      const callId = call.id || `${formatToolCallSummary(call)}-${i}`;
      if (!turn.toolCallEntries.has(callId)) {
        const entry = this._appendToolGroupEntry(
          turn.toolGroupBodyEl, "call", formatToolCallSummary(call), callId
        );
        turn.toolCallEntries.set(callId, entry);
      }
    });
    this._updateToolGroupSummary(chatId);
  }

  handleToolExecutingEvent(payload, insertBeforeEl = null) {
    const call = payload.toolExecuting;
    if (!call) return;

    const chatId = payload.chatId || this.host.getConversationId() || "";
    const label = this.host.formatAgentLabel(payload);
    const turn = this._ensureToolGroup(chatId, label, insertBeforeEl);

    const callId = call.id || formatToolCallSummary(call);
    const text = formatToolCallSummary(call);

    if (turn.toolCallEntries.has(callId)) {
      const entry = turn.toolCallEntries.get(callId);
      entry.className = "tool-group-entry entry-running";
      entry.textContent = text;
    } else {
      const entry = this._appendToolGroupEntry(turn.toolGroupBodyEl, "running", text, callId);
      turn.toolCallEntries.set(callId, entry);
      this._updateToolGroupSummary(chatId);
    }
  }

  handleToolResultEvent(payload, insertBeforeEl = null) {
    const results = payload.toolResults || [];
    if (results.length === 0) return;

    const chatId = payload.chatId || this.host.getConversationId() || "";
    const label = this.host.formatAgentLabel(payload);
    const turn = this._ensureToolGroup(chatId, label, insertBeforeEl);

    results.forEach((result) => {
      const callId = result.toolCallId || result.toolName;
      if (!result.success) {
        const toolName = result.toolName || "Tool";
        const errorMsg = result.error || "Unknown error";
        const text = `✗ ${toolName}: ${errorMsg}`;
        if (callId && turn.toolCallEntries.has(callId)) {
          const entry = turn.toolCallEntries.get(callId);
          entry.className = "tool-group-entry entry-error";
          entry.textContent = text;
        } else {
          this._appendToolGroupEntry(turn.toolGroupBodyEl, "error", text, callId);
        }
      } else if (!result.ephemeral) {
        const summary = formatToolResultSummary(result);
        if (summary) {
          if (callId && turn.toolCallEntries.has(callId)) {
            const entry = turn.toolCallEntries.get(callId);
            entry.className = "tool-group-entry entry-success";
            entry.textContent = summary;
          } else {
            this._appendToolGroupEntry(turn.toolGroupBodyEl, "success", summary, callId);
          }
        }
      } else if (result.ephemeral && callId && turn.toolCallEntries.has(callId)) {
        turn.toolCallEntries.get(callId).remove();
        turn.toolCallEntries.delete(callId);
      }
    });
    this._updateToolGroupSummary(chatId);
    this.deleteTurn(chatId);
  }

  _startTurn(label, insertBeforeEl = null) {
    this.host.clearEmptyState();

    const turnEl = document.createElement("div");
    turnEl.className = "message msg-assistant";

    const metaEl = document.createElement("div");
    metaEl.className = "message-meta";
    metaEl.textContent = label;
    turnEl.appendChild(metaEl);

    const textEl = document.createElement("div");
    textEl.className = "message-body turn-text";
    turnEl.appendChild(textEl);

    const copyBtn = document.createElement("button");
    copyBtn.className = "copy-btn";
    copyBtn.textContent = "Copy";
    copyBtn.addEventListener("click", () => {
      const text = textEl.getAttribute("data-raw-content") || textEl.textContent || "";
      navigator.clipboard.writeText(text).then(() => {
        copyBtn.textContent = "Copied!";
        copyBtn.classList.add("copied");
        setTimeout(() => {
          copyBtn.textContent = "Copy";
          copyBtn.classList.remove("copied");
        }, 1500);
      });
    });
    turnEl.appendChild(copyBtn);

    const { messagesEl } = this.host;
    if (insertBeforeEl) {
      messagesEl.insertBefore(turnEl, insertBeforeEl);
    } else {
      messagesEl.appendChild(turnEl);
      this.host.scrollToBottom();
    }

    const markdownRenderer = new MarkdownRenderer((text) => {
      textEl.setAttribute("data-raw-content", text);
      textEl.innerHTML = renderMarkdown(text);
      if (!insertBeforeEl) this.host.scrollToBottom();
    });

    return {
      el: turnEl,
      textEl,
      toolGroupEl: null,
      toolGroupBodyEl: null,
      toolCallEntries: new Map(),
      rawText: "",
      markdownRenderer,
    };
  }

  _ensureTurn(chatId, label, insertBeforeEl = null) {
    if (!this.activeTurns.has(chatId)) {
      this.activeTurns.set(chatId, this._startTurn(label, insertBeforeEl));
    }
    return this.activeTurns.get(chatId);
  }

  _ensureToolGroup(chatId, label, insertBeforeEl = null) {
    const turn = this._ensureTurn(chatId, label, insertBeforeEl);
    if (!turn.toolGroupEl) {
      const details = document.createElement("details");
      details.className = "tool-group turn-tool-group";

      const summary = document.createElement("summary");
      summary.dataset.label = label;
      summary.textContent = `${label} — tool calls`;
      details.appendChild(summary);

      const body = document.createElement("div");
      body.className = "tool-group-body";
      details.appendChild(body);

      const copyBtn = turn.el.querySelector(".copy-btn");
      turn.el.insertBefore(details, copyBtn);

      turn.toolGroupEl = details;
      turn.toolGroupBodyEl = body;
      this.host.scrollToBottom();
    }
    return turn;
  }

  _appendToolGroupEntry(bodyEl, type, text, callId = null) {
    const entry = document.createElement("div");
    entry.className = `tool-group-entry entry-${type}`;
    entry.textContent = text;
    if (callId) entry.dataset.callId = callId;
    bodyEl.appendChild(entry);
    this.host.scrollToBottom();
    return entry;
  }

  _updateToolGroupSummary(chatId) {
    const turn = this.activeTurns.get(chatId);
    if (!turn?.toolGroupEl) return;
    const summary = turn.toolGroupEl.querySelector("summary");
    if (!summary) return;
    const entryCount = turn.toolGroupEl.querySelectorAll(".tool-group-entry").length;
    const label = summary.dataset.label || "Assistant";
    summary.textContent = `${label} — ${entryCount} tool action${entryCount !== 1 ? "s" : ""}`;
  }
}
