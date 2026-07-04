import { renderMarkdown } from "./markdown.js";
import { parseSSEStream, subscribeSSE } from "./sse.js";
import { TurnManager } from "./chat-turns.js";

// ─── Empty / welcome state HTML ────────────────────────────────────────────
const EMPTY_STATE_HTML = `
  <div class="welcome-state" id="welcome-state">
    <div class="welcome-icon" aria-hidden="true">
      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="var(--accent)"
        stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 2L2 7l10 5 10-5-10-5z"/>
        <path d="M2 17l10 5 10-5"/>
        <path d="M2 12l10 5 10-5"/>
      </svg>
    </div>
    <h2 class="welcome-title" id="welcome-agent-name">ThinkTwice</h2>
    <p class="welcome-sub">An AI agent with tools, memory, and reasoning.</p>
    <div class="welcome-chips">
      <button type="button" class="welcome-chip">Explain this codebase</button>
      <button type="button" class="welcome-chip">Draft a plan for…</button>
      <button type="button" class="welcome-chip">Search the web for…</button>
      <button type="button" class="welcome-chip">Review recent changes</button>
    </div>
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

    this.turns = new TurnManager({
      messagesEl: this.messagesEl,
      scrollToBottom: () => this.scrollToBottom(),
      clearEmptyState: () => this._clearEmptyState(),
      removeThinking: () => this._removeThinking(),
      formatAgentLabel: (payload) => this.formatAgentLabel(payload),
      getConversationId: () => this.state.conversationId,
      appendUserMessage: (...args) => this.appendUserMessage(...args),
    });
    this.thinkingEl = null;
    this.abortController = null;
    this.pushController = null;
    this.pushChatId = null;
    this.heartbeatPushController = null;
    this.pendingUploads = [];
    this.historyPageSize = 100;
    this.historyOffset = 0;
    this.historyHasMore = false;
    this.loadEarlierBtn = null;

    this._bindEvents();
    this._bindScrollButton();
    this._bindMessagesClickDelegation();
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

  _bindMessagesClickDelegation() {
    if (!this.messagesEl) return;
    this.messagesEl.addEventListener("click", (e) => {
      const chip = e.target.closest(".welcome-chip");
      if (chip && this.messagesEl.contains(chip)) {
        e.preventDefault();
        this.inputEl.value = chip.textContent.trim();
        this._resizeTextarea();
        this.inputEl.focus();
        return;
      }
      const copyBtn = e.target.closest(".code-copy-btn");
      if (!copyBtn || !this.messagesEl.contains(copyBtn)) return;
      e.preventDefault();
      const block = copyBtn.closest(".code-block");
      const codeEl = block?.querySelector("pre code");
      if (!codeEl) return;
      const t = codeEl.textContent ?? "";
      navigator.clipboard.writeText(t).then(() => {
        const prev = copyBtn.textContent;
        copyBtn.textContent = "Copied!";
        setTimeout(() => {
          copyBtn.textContent = prev || "Copy";
        }, 1500);
      });
    });
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
    this.messagesEl.querySelector(".welcome-state")?.remove();
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
    this.state.events.dispatchEvent(
      new CustomEvent("agentStatusChanged", { detail: { status: text } })
    );
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
    subscribeSSE(
      `/api/chat/push?conversationId=${encodeURIComponent(chatId)}`,
      {
        signal: this.pushController.signal,
        breakOnNotOk: true,
        onEvent: (eventType, payload) => this.handlePushEvent(eventType, payload),
      }
    );
  }

  stopPushListener() {
    if (this.pushController) {
      this.pushController.abort();
      this.pushController = null;
      this.pushChatId = null;
    }
  }

  // ── Heartbeat push listener (permanent background SSE) ────────────────────
  // Subscribes to the fixed "heartbeat-live" channel so heartbeat responses
  // are always visible in the active conversation view, regardless of which
  // conversation is currently open.
  startHeartbeatListener() {
    if (this.heartbeatPushController) return;
    this.heartbeatPushController = new AbortController();
    subscribeSSE("/api/chat/push?conversationId=heartbeat-live", {
      signal: this.heartbeatPushController.signal,
      notOkDelayMs: 5000,
      onEvent: (eventType, payload) => this.handlePushEvent(eventType, payload),
    });
  }

  handlePushEvent(eventType, payload) {
    this._routeChunkEvent(eventType, payload, { source: "push" });
  }

  // ── Conversation lifecycle ────────────────────────────────────────────────
  clearMessages() {
    this.messagesEl.innerHTML = "";
    this.turns.clear();
    this.thinkingEl = null;
    this.historyOffset = 0;
    this.historyHasMore = false;
    this._removeLoadEarlierButton();
  }

  _removeLoadEarlierButton() {
    if (this.loadEarlierBtn) {
      this.loadEarlierBtn.remove();
      this.loadEarlierBtn = null;
    }
  }

  _ensureLoadEarlierButton() {
    if (!this.historyHasMore) {
      this._removeLoadEarlierButton();
      return;
    }
    if (this.loadEarlierBtn) return;

    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "btn load-earlier-btn";
    btn.textContent = "Load earlier messages";
    btn.addEventListener("click", () => this.loadEarlierMessages());
    this.messagesEl.prepend(btn);
    this.loadEarlierBtn = btn;
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

    this.turns.clear();

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
    this.historyOffset = 0;

    try {
      const res = await fetch(
        `/api/conversations/${conversationId}?limit=${this.historyPageSize}&offset=0`
      );
      if (!res.ok) throw new Error("failed to load conversation");
      const data = await res.json();
      const history = data.messages || [];
      this.historyHasMore = Boolean(data.hasMore);
      this._ensureLoadEarlierButton();

      if (history.length === 0) {
        this._showEmptyState();
      } else {
        history.forEach((msg) => this.turns.renderHistoryMessage(msg));
      }
      this.setStatus("Idle");
    } catch {
      this.setStatus("Error");
      this.appendMessage("System", "Failed to load conversation", [
        "message", "msg-tool-result-error",
      ]);
    }
  }

  async loadEarlierMessages() {
    if (!this.state.conversationId || !this.historyHasMore || !this.loadEarlierBtn) return;

    const prevHeight = this.messagesEl.scrollHeight;
    const prevTop = this.messagesEl.scrollTop;
    this.loadEarlierBtn.disabled = true;
    this.loadEarlierBtn.textContent = "Loading…";

    this.historyOffset += this.historyPageSize;

    try {
      const res = await fetch(
        `/api/conversations/${this.state.conversationId}?limit=${this.historyPageSize}&offset=${this.historyOffset}`
      );
      if (!res.ok) throw new Error("failed to load earlier messages");
      const data = await res.json();
      const history = data.messages || [];
      this.historyHasMore = Boolean(data.hasMore);

      this.turns.clear();
      const anchor = this.loadEarlierBtn.nextSibling;
      history.forEach((msg) => this.turns.renderHistoryMessage(msg, { insertBeforeEl: anchor }));

      this.messagesEl.scrollTop = this.messagesEl.scrollHeight - prevHeight + prevTop;
      this._ensureLoadEarlierButton();
    } catch {
      this.historyOffset -= this.historyPageSize;
      if (this.loadEarlierBtn) {
        this.loadEarlierBtn.textContent = "Load earlier messages (failed)";
      }
    } finally {
      if (this.loadEarlierBtn) {
        this.loadEarlierBtn.disabled = false;
        if (this.loadEarlierBtn.textContent === "Loading…") {
          this.loadEarlierBtn.textContent = "Load earlier messages";
        }
      }
    }
  }

  // ── Message constructors ──────────────────────────────────────────────────
  appendUserMessage(message, images = [], insertBeforeEl = null) {
    const el = this.appendMessage("You", message, ["message", "msg-user"], false, insertBeforeEl);
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

  appendMessage(label, content, classes, enableMarkdown = false, insertBeforeEl = null) {
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
      bodyEl.innerHTML = renderMarkdown(content);
    } else {
      bodyEl.textContent = content;
    }

    messageEl.appendChild(metaEl);
    messageEl.appendChild(bodyEl);

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

    if (insertBeforeEl) {
      this.messagesEl.insertBefore(messageEl, insertBeforeEl);
    } else {
      this.messagesEl.appendChild(messageEl);
      this.scrollToBottom();
    }
    return messageEl;
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
    this._routeChunkEvent(eventType, payload, { source: "stream" });
  }

  // Shared dispatch for stream and push SSE paths.
  // Stream-only: chatId binding, status UI, error handling.
  // Both: content/tool events, completed cleanup, conversationUpdated.
  _routeChunkEvent(eventType, payload, { source }) {
    if (!payload) return;

    if (source === "stream" && payload.chatId && payload.chatId !== this.state.conversationId) {
      this.state.conversationId = payload.chatId;
      localStorage.setItem("currentConversationId", payload.chatId);
      this.state.events.dispatchEvent(
        new CustomEvent("conversationIdUpdated", { detail: { id: payload.chatId } })
      );
      this.startPushListener(payload.chatId);
    }

    if (eventType === "content") this.turns.handleContentEvent(payload);
    else if (eventType === "tool_call") this.turns.handleToolCallEvent(payload);
    else if (eventType === "tool_executing") this.turns.handleToolExecutingEvent(payload);
    else if (eventType === "tool_result") this.turns.handleToolResultEvent(payload);
    else if (eventType === "completed") {
      this.turns.flushAllMarkdown();
      const chatId = source === "push"
        ? (payload.chatId || this.state.conversationId)
        : this.state.conversationId;
      this.turns.deleteTurn(chatId);
      if (source === "stream") {
        this.setStatus("Idle");
        this.hideStopButton();
        this._removeThinking();
      }
      if (chatId) {
        this.state.events.dispatchEvent(
          new CustomEvent("conversationUpdated", {
            detail: { id: chatId, updatedAt: new Date().toISOString() },
          })
        );
      }
    } else if (eventType === "error" && source === "stream") {
      this.setStatus("Error");
      this.hideStopButton();
      this._removeThinking();
      this.appendMessage("Error", payload.content || "Unknown error", ["message", "msg-tool-result-error"]);
    }
  }

  formatAgentLabel(_payload) {
    // Always use the main agent's name for continuity, silencing sub-agent identity switches.
    return this.state.agentName || "Assistant";
  }

  scrollToBottom() {
    this.messagesEl.scrollTop = this.messagesEl.scrollHeight;
    // Hide scroll button when we programmatically scroll to bottom
    if (this.scrollBtn) this.scrollBtn.style.display = "none";
  }
}
