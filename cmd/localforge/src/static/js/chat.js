// ─── Sensitive key filtering ───────────────────────────────────────────────
const SENSITIVE_KEYS = new Set([
  "password", "passwd", "secret", "token", "apikey", "api_key", "apiKey",
  "key", "authorization", "auth", "credentials",
]);

function truncate(str, maxLen = 50) {
  if (typeof str !== "string") return String(str);
  return str.length <= maxLen ? str : str.slice(0, maxLen) + "…";
}

function formatArguments(args) {
  if (!args || typeof args !== "object") return "";
  const parts = [];
  for (const [key, value] of Object.entries(args)) {
    if (SENSITIVE_KEYS.has(String(key).toLowerCase())) continue;
    const str = typeof value === "object" ? JSON.stringify(value) : String(value ?? "");
    parts.push(`${key}=${truncate(str, 40)}`);
  }
  return truncate(parts.join(", "), 100);
}

function formatToolCallSummary(call) {
  const name = call?.function?.name || call?.name || "Unknown tool";
  const args = call?.function?.arguments ?? call?.arguments ?? {};
  const argSummary = formatArguments(args);
  return argSummary ? `${name}: ${argSummary}` : name;
}

function formatToolResultSummary(result) {
  const name = result?.toolName || "Tool";
  if (!result?.success) return null;
  const resultStr = (result?.result || "").trim();
  const summary = resultStr ? truncate(resultStr, 80) : "";
  return summary ? `✓ ${name}: ${summary}` : `✓ ${name}`;
}

// ─── Empty state HTML ──────────────────────────────────────────────────────
const EMPTY_STATE_HTML = `
  <div class="empty-state">
    <div class="empty-icon">✦</div>
    <h2>Start a conversation</h2>
    <p>Select one from the sidebar or send a message to begin.</p>
  </div>
`;

// ─── Thinking indicator HTML ───────────────────────────────────────────────
function makeThinkingEl() {
  const el = document.createElement("div");
  el.className = "message msg-thinking";
  el.innerHTML = `
    <div class="message-meta">Thinking</div>
    <div class="message-body">
      <span class="thinking-indicator">
        <span class="thinking-dots">
          <span class="thinking-dot"></span>
          <span class="thinking-dot"></span>
          <span class="thinking-dot"></span>
        </span>
      </span>
    </div>`;
  return el;
}

// ─── ChatManager ───────────────────────────────────────────────────────────
export class ChatManager {
  constructor(appState) {
    this.state = appState;
    this.messagesEl = document.getElementById("chat-messages");
    this.formEl = document.getElementById("chat-form");
    this.inputEl = document.getElementById("chat-input");
    this.statusEl = document.getElementById("chat-status");
    this.statusTextEl = this.statusEl?.querySelector(".status-text");
    this.stopBtn = document.getElementById("stop-btn");
    this.attachBtn = document.getElementById("attach-btn");
    this.attachBadge = document.getElementById("attach-badge");
    this.imageInput = document.getElementById("image-input");
    this.imagePreviewEl = document.getElementById("image-preview");
    this.scrollBtn = document.getElementById("scroll-to-bottom");

    this.currentAssistantEl = null;
    this.currentIteration = null;
    this.currentAgentName = null;
    this.currentToolGroupEl = null;
    this.currentToolGroupBodyEl = null;
    this.thinkingEl = null;
    this.abortController = null;
    this.pushController = null;
    this.pushChatId = null;
    this.pendingUploads = [];

    this._bindEvents();
    this._bindScrollButton();
    this._showEmptyState();
  }

  // ── Event bindings ────────────────────────────────────────────────────────
  _bindEvents() {
    this.formEl.addEventListener("submit", (e) => {
      e.preventDefault();
      this.handleSubmit();
    });

    this.inputEl.addEventListener("keydown", (e) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        this.handleSubmit();
      }
    });

    // Auto-resize textarea
    this.inputEl.addEventListener("input", () => this._resizeTextarea());

    if (this.attachBtn && this.imageInput) {
      this.attachBtn.addEventListener("click", () => this.imageInput.click());
      this.imageInput.addEventListener("change", () => {
        // Snapshot files into a stable Array before clearing the input.
        // The FileList is live — clearing value="" can wipe it on some browsers
        // before the async FileReader gets a chance to read the File objects.
        const files = Array.from(this.imageInput.files);
        this.imageInput.value = "";
        if (files.length > 0) this._uploadFiles(files);
      });
    }

    this.inputEl.addEventListener("paste", (e) => {
      const items = e.clipboardData?.items;
      if (!items) return;
      const imageFiles = [];
      for (const item of items) {
        if (item.type.startsWith("image/")) {
          const f = item.getAsFile();
          if (f) imageFiles.push(f);
        }
      }
      if (imageFiles.length > 0) {
        e.preventDefault();
        this._uploadFiles(imageFiles);
      }
    });

    if (this.stopBtn) {
      this.stopBtn.addEventListener("click", () => this.stopStream());
    }
  }

  _bindScrollButton() {
    if (!this.scrollBtn) return;
    this.messagesEl.addEventListener("scroll", () => {
      const nearBottom = this.messagesEl.scrollHeight - this.messagesEl.scrollTop - this.messagesEl.clientHeight < 80;
      this.scrollBtn.style.display = nearBottom ? "none" : "";
    });
    this.scrollBtn.addEventListener("click", () => this.scrollToBottom());
  }

  _resizeTextarea() {
    const el = this.inputEl;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 200) + "px";
  }

  _resetTextarea() {
    this.inputEl.style.height = "";
  }

  // ── Empty state ───────────────────────────────────────────────────────────
  _showEmptyState() {
    if (this.messagesEl.children.length === 0) {
      this.messagesEl.innerHTML = EMPTY_STATE_HTML;
    }
  }

  _clearEmptyState() {
    const empty = this.messagesEl.querySelector(".empty-state");
    if (empty) empty.remove();
  }

  // ── Image uploads ─────────────────────────────────────────────────────────
  async _uploadFiles(files) {
    for (const file of files) {
      if (!file) continue;
      // On Linux some file pickers return empty type; fall back to extension check
      const looksLikeImage = file.type.startsWith("image/") ||
        /\.(jpe?g|png|gif|webp|bmp|svg|ico|tiff?)$/i.test(file.name);
      if (!looksLikeImage) {
        console.warn("Skipped non-image file:", file.name, file.type);
        continue;
      }

      let dataUrl;
      try {
        dataUrl = await new Promise((resolve, reject) => {
          const reader = new FileReader();
          reader.onload = (e) => resolve(e.target.result);
          reader.onerror = (e) => reject(new Error(`FileReader error: ${e.target.error}`));
          reader.readAsDataURL(file);
        });
      } catch (err) {
        console.error("Could not read file:", err);
        // Show the entry as failed so user sees it rather than silence
        const failed = { dataUrl: "", path: null, status: "error", name: file.name };
        this.pendingUploads.push(failed);
        this._renderPreviews();
        continue;
      }

      // status: 'uploading' | 'done' | 'error'
      const entry = { dataUrl, path: null, status: "uploading", name: file.name };
      this.pendingUploads.push(entry);
      this._renderPreviews();

      try {
        const form = new FormData();
        form.append("file", file);
        const res = await fetch("/api/upload", { method: "POST", body: form });
        if (res.ok) {
          const data = await res.json();
          entry.path = data.path;
          entry.status = "done";
        } else {
          entry.status = "error";
        }
      } catch (err) {
        console.error("Upload failed:", err);
        entry.status = "error";
      }

      this._renderPreviews();
    }
  }

  _renderPreviews() {
    this.imagePreviewEl.innerHTML = "";
    this.pendingUploads.forEach((item, idx) => {
      const wrapper = document.createElement("div");
      wrapper.className = `image-thumb-wrapper image-thumb-wrapper--${item.status}`;

      const thumb = document.createElement("img");
      thumb.className = "image-thumb";
      if (item.dataUrl) thumb.src = item.dataUrl;
      thumb.alt = item.name || "image";

      wrapper.appendChild(thumb);

      if (item.status === "uploading") {
        const spinner = document.createElement("div");
        spinner.className = "image-thumb-spinner";
        wrapper.appendChild(spinner);
      } else if (item.status === "done") {
        const badge = document.createElement("span");
        badge.className = "image-thumb-status";
        badge.textContent = "✓";
        wrapper.appendChild(badge);
      } else if (item.status === "error") {
        const badge = document.createElement("span");
        badge.className = "image-thumb-status";
        badge.textContent = "⚠ failed";
        wrapper.title = "Upload failed — remove and try again";
        wrapper.appendChild(badge);
      }

      const removeBtn = document.createElement("button");
      removeBtn.className = "image-thumb-remove";
      removeBtn.type = "button";
      removeBtn.title = "Remove";
      removeBtn.textContent = "×";
      removeBtn.addEventListener("click", () => {
        this.pendingUploads.splice(idx, 1);
        this._renderPreviews();
      });

      wrapper.appendChild(removeBtn);
      this.imagePreviewEl.appendChild(wrapper);
    });

    this._updateAttachBadge();
  }

  _updateAttachBadge() {
    if (!this.attachBadge) return;
    const count = this.pendingUploads.length;
    if (count > 0) {
      this.attachBadge.textContent = count;
      this.attachBadge.style.display = "";
    } else {
      this.attachBadge.style.display = "none";
    }
  }

  _clearPendingImages() {
    this.pendingUploads = [];
    this.imagePreviewEl.innerHTML = "";
    this._updateAttachBadge();
  }

  // ── Status management ─────────────────────────────────────────────────────
  setStatus(text) {
    if (!this.statusEl) return;
    this.statusEl.className = `chat-status status--${text.toLowerCase()}`;
    if (this.statusTextEl) this.statusTextEl.textContent = text;
  }

  showStopButton() {
    if (this.stopBtn) this.stopBtn.style.display = "";
  }

  hideStopButton() {
    if (this.stopBtn) this.stopBtn.style.display = "none";
  }

  // ── Stop ──────────────────────────────────────────────────────────────────
  async stopStream() {
    if (this.abortController) this.abortController.abort();

    if (this.state.conversationId) {
      try {
        await fetch("/api/chat/stop", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ conversationId: this.state.conversationId }),
        });
      } catch {
        // ignore
      }
    }

    this.hideStopButton();
    this.setStatus("Stopped");
    this._removeThinking();
    this.appendMessage("System", "Agent stopped", ["message", "msg-tool-result-error"]);
  }

  // ── Push listener (background SSE) ───────────────────────────────────────
  startPushListener(chatId) {
    if (this.pushChatId === chatId && this.pushController) return;
    this.stopPushListener();
    this.pushChatId = chatId;
    this.pushController = new AbortController();
    const controller = this.pushController;

    (async () => {
      while (!controller.signal.aborted) {
        try {
          const res = await fetch(
            `/api/chat/push?conversationId=${encodeURIComponent(chatId)}`,
            { signal: controller.signal }
          );
          if (!res.ok || !res.body) break;
          await parseSSEStream(res.body, (eventType, payload) => {
            this.handlePushEvent(eventType, payload);
          });
        } catch (err) {
          if (err.name === "AbortError") break;
          await new Promise((r) => setTimeout(r, 2000));
        }
      }
    })();
  }

  stopPushListener() {
    if (this.pushController) {
      this.pushController.abort();
      this.pushController = null;
      this.pushChatId = null;
    }
  }

  handlePushEvent(eventType, payload) {
    if (!payload) return;
    if (eventType === "content")         this.handleContentEvent(payload);
    else if (eventType === "tool_call")  this.handleToolCallEvent(payload);
    else if (eventType === "tool_executing") this.handleToolExecutingEvent(payload);
    else if (eventType === "tool_result")    this.handleToolResultEvent(payload);
  }

  // ── Conversation lifecycle ────────────────────────────────────────────────
  clearMessages() {
    this.messagesEl.innerHTML = "";
    this.currentAssistantEl = null;
    this.currentIteration = null;
    this.currentAgentName = null;
    this.currentToolGroupEl = null;
    this.currentToolGroupBodyEl = null;
    this.thinkingEl = null;
  }

  async handleSubmit() {
    const text = this.inputEl.value.trim();
    if (!text && this.pendingUploads.length === 0) return;
    this.inputEl.value = "";
    this._resetTextarea();

    const stillUploading = this.pendingUploads.some((u) => u.status === "uploading");
    if (stillUploading) return; // wait for uploads to finish

    const uploads = this.pendingUploads.slice();
    this._clearPendingImages();

    const errored = uploads.filter((u) => u.status === "error");
    if (errored.length > 0 && uploads.filter((u) => u.status === "done").length === 0 && !text) return;

    const paths = uploads.filter((u) => u.path).map((u) => u.path);
    let message = text;
    if (paths.length > 0) {
      message = (text ? text + "\n\n" : "") + paths.join("\n");
    }

    const previewUrls = uploads.map((u) => u.dataUrl);
    this._clearEmptyState();
    this.appendUserMessage(text, previewUrls);

    this.currentAssistantEl = null;
    this.currentIteration = null;
    this.currentAgentName = null;
    this.currentToolGroupEl = null;
    this.currentToolGroupBodyEl = null;

    await this.startStream(message);
  }

  async startNewConversation() {
    this.state.conversationId = "";
    localStorage.removeItem("currentConversationId");
    this.stopPushListener();
    this.setStatus("Idle");
    this.clearMessages();
    this._showEmptyState();
  }

  async loadConversation(conversationId) {
    this.state.conversationId = conversationId;
    localStorage.setItem("currentConversationId", conversationId);
    this.stopPushListener();
    this.startPushListener(conversationId);
    this.clearMessages();
    this.setStatus("Loading");

    try {
      const res = await fetch(`/api/conversations/${conversationId}`);
      if (!res.ok) throw new Error("failed to load conversation");
      const history = await res.json();
      if (history.length === 0) {
        this._showEmptyState();
      } else {
        history.forEach((msg) => this.renderHistoryMessage(msg));
      }
      this.setStatus("Idle");
    } catch {
      this.setStatus("Error");
      this.appendMessage("System", "Failed to load conversation", [
        "message", "msg-tool-result-error",
      ]);
    }
  }

  // ── History rendering (simplified, no collapsible groups for past messages) ─
  renderHistoryMessage(msg) {
    const role = msg.role || "assistant";
    const content = msg.content || "";
    if (role === "user") {
      this.appendUserMessage(content, []);
      return;
    }
    if (role === "assistant") {
      this.appendMessage("Assistant", content, ["message", "msg-assistant"], true);
      if (msg.toolCalls?.length) {
        const body = msg.toolCalls.map(formatToolCallSummary).join(" | ");
        this.appendMessage("Tool call", body, ["message", "msg-tool-call"]);
      }
      return;
    }
    if (role === "tool") {
      this.appendMessage("Tool result", content, ["message", "msg-tool-result-success"]);
      return;
    }
    this.appendMessage("System", content, ["message"]);
  }

  // ── Message constructors ──────────────────────────────────────────────────
  appendUserMessage(message, images = []) {
    const el = this.appendMessage("You", message, ["message", "msg-user"]);
    if (images.length > 0) {
      const imagesEl = document.createElement("div");
      imagesEl.className = "message-images";
      images.forEach((dataUrl) => {
        const img = document.createElement("img");
        img.className = "message-image";
        img.src = dataUrl;
        img.addEventListener("click", () => window.open(dataUrl, "_blank"));
        imagesEl.appendChild(img);
      });
      el.appendChild(imagesEl);
    }
    return el;
  }

  appendMessage(label, content, classes, enableMarkdown = false) {
    this._clearEmptyState();
    const messageEl = document.createElement("div");
    classes.filter(Boolean).forEach((cls) => messageEl.classList.add(cls));

    const metaEl = document.createElement("div");
    metaEl.className = "message-meta";
    metaEl.textContent = label;

    const bodyEl = document.createElement("div");
    bodyEl.className = "message-body";

    if (enableMarkdown && content) {
      bodyEl.setAttribute("data-raw-content", content);
      bodyEl.innerHTML = this.renderMarkdown(content);
    } else {
      bodyEl.textContent = content;
    }

    messageEl.appendChild(metaEl);
    messageEl.appendChild(bodyEl);

    // Copy button
    const copyBtn = document.createElement("button");
    copyBtn.className = "copy-btn";
    copyBtn.textContent = "Copy";
    copyBtn.addEventListener("click", () => {
      const text = bodyEl.getAttribute("data-raw-content") || bodyEl.textContent || "";
      navigator.clipboard.writeText(text).then(() => {
        copyBtn.textContent = "Copied!";
        copyBtn.classList.add("copied");
        setTimeout(() => {
          copyBtn.textContent = "Copy";
          copyBtn.classList.remove("copied");
        }, 1500);
      });
    });
    messageEl.appendChild(copyBtn);

    this.messagesEl.appendChild(messageEl);
    this.scrollToBottom();
    return messageEl;
  }

  // ── Tool group (collapsible <details>) ────────────────────────────────────
  _appendToolGroup(agentLabel) {
    this._clearEmptyState();
    const details = document.createElement("details");
    details.className = "tool-group";

    const summary = document.createElement("summary");
    summary.dataset.label = agentLabel;
    summary.textContent = `${agentLabel} — tool calls`;
    details.appendChild(summary);

    const body = document.createElement("div");
    body.className = "tool-group-body";
    details.appendChild(body);

    this.messagesEl.appendChild(details);
    this.scrollToBottom();
    return { details, body };
  }

  _appendToolGroupEntry(bodyEl, type, text) {
    const entry = document.createElement("div");
    entry.className = `tool-group-entry entry-${type}`;
    entry.textContent = text;
    bodyEl.appendChild(entry);
    this.scrollToBottom();
  }

  _updateToolGroupSummary() {
    if (!this.currentToolGroupEl) return;
    const summary = this.currentToolGroupEl.querySelector("summary");
    if (!summary) return;
    const entryCount = this.currentToolGroupEl.querySelectorAll(".tool-group-entry").length;
    const label = summary.dataset.label || "Assistant";
    summary.textContent = `${label} — ${entryCount} tool action${entryCount !== 1 ? "s" : ""}`;
  }

  // ── Thinking indicator ────────────────────────────────────────────────────
  _showThinking(agentLabel) {
    if (this.thinkingEl) return;
    this.thinkingEl = makeThinkingEl();
    this.thinkingEl.querySelector(".message-meta").textContent = agentLabel;
    this.messagesEl.appendChild(this.thinkingEl);
    this.scrollToBottom();
  }

  _removeThinking() {
    if (this.thinkingEl) {
      this.thinkingEl.remove();
      this.thinkingEl = null;
    }
  }

  // ── Stream handling ───────────────────────────────────────────────────────
  async startStream(message) {
    this.setStatus("Streaming");
    this.showStopButton();
    this._showThinking("Assistant");
    this.abortController = new AbortController();

    const query = this.state.conversationId
      ? `?conversationId=${encodeURIComponent(this.state.conversationId)}`
      : "";

    try {
      const res = await fetch(`/api/chat${query}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message }),
        signal: this.abortController.signal,
      });

      if (!res.ok || !res.body) {
        this.setStatus("Error");
        this.hideStopButton();
        this._removeThinking();
        this.appendMessage("System", "Failed to start stream", ["message", "msg-tool-result-error"]);
        return;
      }

      await parseSSEStream(res.body, (eventType, payload) => {
        this.handleStreamEvent(eventType, payload);
      });
    } catch (err) {
      if (err.name !== "AbortError") {
        this.setStatus("Error");
        this.hideStopButton();
        this._removeThinking();
        this.appendMessage("System", `Stream error: ${err.message}`, ["message", "msg-tool-result-error"]);
      }
    } finally {
      this.abortController = null;
    }
  }

  handleStreamEvent(eventType, payload) {
    if (!payload) return;

    if (payload.chatId && payload.chatId !== this.state.conversationId) {
      this.state.conversationId = payload.chatId;
      localStorage.setItem("currentConversationId", payload.chatId);
      this.state.events.dispatchEvent(
        new CustomEvent("conversationIdUpdated", { detail: { id: payload.chatId } })
      );
      this.startPushListener(payload.chatId);
    }

    if (eventType === "content")              this.handleContentEvent(payload);
    else if (eventType === "tool_call")       this.handleToolCallEvent(payload);
    else if (eventType === "tool_executing")  this.handleToolExecutingEvent(payload);
    else if (eventType === "tool_result")     this.handleToolResultEvent(payload);
    else if (eventType === "completed") {
      this.setStatus("Idle");
      this.hideStopButton();
      this._removeThinking();
      this.state.events.dispatchEvent(
        new CustomEvent("conversationUpdated", { detail: { id: this.state.conversationId } })
      );
    } else if (eventType === "error") {
      this.setStatus("Error");
      this.hideStopButton();
      this._removeThinking();
      this.appendMessage("Error", payload.content || "Unknown error", ["message", "msg-tool-result-error"]);
    }
  }

  handleContentEvent(payload) {
    const delta = payload.delta || payload.content || "";
    if (!delta) return;

    // Remove thinking indicator on first content
    this._removeThinking();

    // New bubble when iteration or agent changes
    const iteration = payload.iteration !== undefined ? payload.iteration : null;
    const agentName = payload.agentName || null;
    const iterChanged = iteration !== null && iteration !== this.currentIteration;
    const agentChanged = agentName !== null && agentName !== this.currentAgentName;
    if (iterChanged || agentChanged) {
      this.currentIteration = iteration;
      this.currentAgentName = agentName;
      this.currentAssistantEl = null;
      // Close current tool group when new content begins
      this.currentToolGroupEl = null;
      this.currentToolGroupBodyEl = null;
    }

    if (!this.currentAssistantEl) {
      this.currentAssistantEl = this.appendMessage(
        this.formatAgentLabel(payload),
        "",
        ["message", "msg-assistant", this.subagentClass(payload)],
        true
      );
    }

    const bodyEl = this.currentAssistantEl.querySelector(".message-body");
    const currentText = bodyEl.getAttribute("data-raw-content") || "";
    const newText = currentText + delta;
    bodyEl.setAttribute("data-raw-content", newText);
    bodyEl.innerHTML = this.renderMarkdown(newText);
    this.scrollToBottom();
  }

  handleToolCallEvent(payload) {
    // Remove thinking — a tool is being called
    this._removeThinking();

    // Close previous assistant bubble so next content appears after the group
    this.currentAssistantEl = null;
    this.currentAgentName = null;

    const calls = payload.toolCalls || [];
    const agentLabel = this.formatAgentLabel(payload);

    if (!this.currentToolGroupEl) {
      const { details, body } = this._appendToolGroup(agentLabel);
      this.currentToolGroupEl = details;
      this.currentToolGroupBodyEl = body;
    }

    calls.forEach((call) => {
      this._appendToolGroupEntry(
        this.currentToolGroupBodyEl,
        "call",
        formatToolCallSummary(call)
      );
    });
    this._updateToolGroupSummary();
  }

  handleToolExecutingEvent(payload) {
    const call = payload.toolExecuting;
    if (!call) return;

    if (!this.currentToolGroupEl) {
      const agentLabel = this.formatAgentLabel(payload);
      const { details, body } = this._appendToolGroup(agentLabel);
      this.currentToolGroupEl = details;
      this.currentToolGroupBodyEl = body;
    }

    this._appendToolGroupEntry(
      this.currentToolGroupBodyEl,
      "running",
      `Running: ${formatToolCallSummary(call)}`
    );
    this._updateToolGroupSummary();
  }

  handleToolResultEvent(payload) {
    const results = payload.toolResults || [];
    if (results.length === 0) return;

    results.forEach((result) => {
      if (!result.success) {
        const toolName = result.toolName || "Tool";
        const errorMsg = result.error || "Unknown error";
        if (this.currentToolGroupEl) {
          this._appendToolGroupEntry(
            this.currentToolGroupBodyEl,
            "error",
            `✗ ${toolName}: ${errorMsg}`
          );
        } else {
          this.appendMessage(
            this.formatAgentLabel(payload),
            `✗ ${toolName}: ${errorMsg}`,
            ["message", "msg-tool-result-error", this.subagentClass(payload)]
          );
        }
      } else if (!result.ephemeral) {
        const summary = formatToolResultSummary(result);
        if (summary && this.currentToolGroupEl) {
          this._appendToolGroupEntry(this.currentToolGroupBodyEl, "success", summary);
        } else if (summary) {
          this.appendMessage(
            this.formatAgentLabel(payload),
            summary,
            ["message", "msg-tool-result-success", this.subagentClass(payload)]
          );
        }
      }
    });
    this._updateToolGroupSummary();
  }

  // ── Helpers ───────────────────────────────────────────────────────────────
  subagentClass(payload) {
    if (payload.agentName && this.state.agentName) {
      if (payload.agentName !== this.state.agentName) return "msg-subagent";
    }
    return "";
  }

  formatAgentLabel(payload) {
    return payload.agentName || "Assistant";
  }

  renderMarkdown(text) {
    if (typeof marked === "undefined" || typeof DOMPurify === "undefined") return text;
    return DOMPurify.sanitize(marked.parse(text));
  }

  scrollToBottom() {
    this.messagesEl.scrollTop = this.messagesEl.scrollHeight;
    // Hide scroll button when we programmatically scroll to bottom
    if (this.scrollBtn) this.scrollBtn.style.display = "none";
  }
}

// ─── SSE stream parser ─────────────────────────────────────────────────────
async function parseSSEStream(body, onEvent) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split("\n\n");
    buffer = parts.pop() || "";

    parts.forEach((part) => {
      const lines = part.split("\n");
      let eventType = "message";
      let data = "";
      lines.forEach((line) => {
        if (line.startsWith("event:"))      eventType = line.slice(6).trim();
        else if (line.startsWith("data:"))  data += line.slice(5).trim();
      });
      if (!data) return;
      try {
        onEvent(eventType, JSON.parse(data));
      } catch {
        onEvent("error", { content: "Failed to parse stream data" });
      }
    });
  }
}
