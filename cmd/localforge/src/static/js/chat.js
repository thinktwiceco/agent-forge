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

function escapeHtml(s) {
  if (typeof s !== "string") return "";
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

let markedCodeRendererInstalled = false;

function ensureMarkedCodeRenderer() {
  if (markedCodeRendererInstalled || typeof marked === "undefined") return;
  marked.use({
    renderer: {
      // Marked v11+ invokes renderers with positional args (same as default Renderer.code).
      code(src, infostring, escaped) {
        const langRaw =
          ((infostring || "").match(/^\S*/)?.[0] || "").trim() || "text";
        const langDisplay = langRaw.toLowerCase();
        const langSlug =
          langRaw.replace(/[^a-zA-Z0-9_.-]/g, "").replace(/^\.+/, "") || "text";
        const body = escaped ? src : escapeHtml(src);
        return `<div class="code-block"><div class="code-header"><span class="code-lang">${escapeHtml(langDisplay)}</span><button type="button" class="code-copy-btn">Copy</button></div><pre><code class="language-${escapeHtml(langSlug)}">${body}</code></pre></div>`;
      },
    },
  });
  markedCodeRendererInstalled = true;
}

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

    // activeTurns: map from chatId → open assistant response container.
    // Each unique chatId gets its own section; events are routed by chatId.
    // { el, textEl, toolGroupEl, toolGroupBodyEl, toolCallEntries, rawText }
    this.activeTurns = new Map();
    this.thinkingEl = null;
    this.abortController = null;
    this.pushController = null;
    this.pushChatId = null;
    this.heartbeatPushController = null;
    this.pendingUploads = [];

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

  // ── Heartbeat push listener (permanent background SSE) ────────────────────
  // Subscribes to the fixed "heartbeat-live" channel so heartbeat responses
  // are always visible in the active conversation view, regardless of which
  // conversation is currently open.
  startHeartbeatListener() {
    if (this.heartbeatPushController) return;
    this.heartbeatPushController = new AbortController();
    const controller = this.heartbeatPushController;

    (async () => {
      while (!controller.signal.aborted) {
        try {
          const res = await fetch(
            "/api/chat/push?conversationId=heartbeat-live",
            { signal: controller.signal }
          );
          if (!res.ok || !res.body) {
            await new Promise((r) => setTimeout(r, 5000));
            continue;
          }
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

  handlePushEvent(eventType, payload) {
    if (!payload) return;
    if (eventType === "content") this.handleContentEvent(payload);
    else if (eventType === "tool_call") this.handleToolCallEvent(payload);
    else if (eventType === "tool_executing") this.handleToolExecutingEvent(payload);
    else if (eventType === "tool_result") this.handleToolResultEvent(payload);
    else if (eventType === "completed") {
      const chatId = payload.chatId || this.state.conversationId;
      this.activeTurns.delete(chatId);
      if (chatId) {
        this.state.events.dispatchEvent(
          new CustomEvent("conversationUpdated", { detail: { id: chatId } })
        );
      }
    }
  }

  // ── Conversation lifecycle ────────────────────────────────────────────────
  clearMessages() {
    this.messagesEl.innerHTML = "";
    this.activeTurns.clear();
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

    this.activeTurns.clear();

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

  // ── History rendering (single source of truth) ────────────────────────────
  renderHistoryMessage(msg) {
    const role = msg.role || "assistant";

    // Hide system messages
    if (role === "system") return;

    // User messages: close all open turns, render user bubble
    if (role === "user") {
      this.activeTurns.clear();
      let content = msg.content || "";
      const sep = content.indexOf("\n\n");
      if (sep !== -1 && /^sender:/m.test(content.slice(0, sep))) {
        const headers = content.slice(0, sep);
        // Browser chat uses sender: user; Telegram/webhook inbound uses sender: webhook.
        // Other senders (scheduler, heartbeat, etc.) stay hidden here.
        const showAsYou =
          /^sender:\s*user\s*$/m.test(headers) ||
          /^sender:\s*webhook\s*$/m.test(headers);
        if (!showAsYou) return;
        content = content.slice(sep + 2);
      }
      this.appendUserMessage(content, []);
      return;
    }

    const agentName = msg.name || "Assistant";

    // Assistant content + optional tool calls
    if (role === "assistant") {
      if (msg.content) {
        this.handleContentEvent({ agentName, content: msg.content });
      }
      const toolCalls = msg.toolCalls || msg.tool_calls;
      if (toolCalls && toolCalls.length > 0) {
        this.handleToolCallEvent({ agentName, toolCalls });
      }
      return;
    }

    // Tool results
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
          error: isError ? resultText : undefined
        }]
      });
      return;
    }
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

  // ── Tool group helpers ────────────────────────────────────────────────────
  _appendToolGroupEntry(bodyEl, type, text, callId = null) {
    const entry = document.createElement("div");
    entry.className = `tool-group-entry entry-${type}`;
    entry.textContent = text;
    if (callId) entry.dataset.callId = callId;
    bodyEl.appendChild(entry);
    this.scrollToBottom();
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

    if (eventType === "content") this.handleContentEvent(payload);
    else if (eventType === "tool_call") this.handleToolCallEvent(payload);
    else if (eventType === "tool_executing") this.handleToolExecutingEvent(payload);
    else if (eventType === "tool_result") this.handleToolResultEvent(payload);
    else if (eventType === "completed") {
      this.setStatus("Idle");
      this.hideStopButton();
      this._removeThinking();
      this.activeTurns.delete(this.state.conversationId);
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

  // ── Turn management ────────────────────────────────────────────────────────
  //
  // One "turn" = one persistent container that holds all content and tool calls
  // produced by the agent for a single user request. Events append *inside*
  // the turn rather than creating new top-level bubbles, preventing fragmentation.

  _startTurn(label) {
    this._clearEmptyState();

    const turnEl = document.createElement("div");
    turnEl.className = "message msg-assistant";

    // Header
    const metaEl = document.createElement("div");
    metaEl.className = "message-meta";
    metaEl.textContent = label;
    turnEl.appendChild(metaEl);

    // Text region — streams content here continuously
    const textEl = document.createElement("div");
    textEl.className = "message-body turn-text";
    turnEl.appendChild(textEl);

    // Copy button (copies the raw text)
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

    this.messagesEl.appendChild(turnEl);
    this.scrollToBottom();

    return {
      el: turnEl,
      textEl,
      toolGroupEl: null,
      toolGroupBodyEl: null,
      toolCallEntries: new Map(),
      rawText: "",
    };
  }

  _ensureTurn(chatId, label) {
    if (!this.activeTurns.has(chatId)) {
      const turn = this._startTurn(label);
      this.activeTurns.set(chatId, turn);
    }
    return this.activeTurns.get(chatId);
  }

  _ensureToolGroup(chatId, label) {
    const turn = this._ensureTurn(chatId, label);
    if (!turn.toolGroupEl) {
      // Insert tool group before the copy button
      const details = document.createElement("details");
      details.className = "tool-group turn-tool-group";

      const summary = document.createElement("summary");
      summary.dataset.label = label;
      summary.textContent = `${label} — tool calls`;
      details.appendChild(summary);

      const body = document.createElement("div");
      body.className = "tool-group-body";
      details.appendChild(body);

      // Insert before the copy button (last child)
      const copyBtn = turn.el.querySelector(".copy-btn");
      turn.el.insertBefore(details, copyBtn);

      turn.toolGroupEl = details;
      turn.toolGroupBodyEl = body;
      this.scrollToBottom();
    }
    return turn;
  }

  handleContentEvent(payload) {
    const delta = payload.delta || payload.content || "";
    if (!delta) return;

    this._removeThinking();

    const chatId = payload.chatId || this.state.conversationId || "";
    const label = this.formatAgentLabel(payload);
    const turn = this._ensureTurn(chatId, label);

    turn.rawText += delta;
    turn.textEl.setAttribute("data-raw-content", turn.rawText);
    turn.textEl.innerHTML = this.renderMarkdown(turn.rawText);
    this.scrollToBottom();
  }

  handleToolCallEvent(payload) {
    this._removeThinking();

    const chatId = payload.chatId || this.state.conversationId || "";
    const label = this.formatAgentLabel(payload);
    const turn = this._ensureToolGroup(chatId, label);
    const calls = payload.toolCalls || [];

    calls.forEach((call, i) => {
      const callId = call.id || `${formatToolCallSummary(call)}-${i}`;
      if (!turn.toolCallEntries.has(callId)) {
        const entry = this._appendToolGroupEntry(turn.toolGroupBodyEl, "call", formatToolCallSummary(call), callId);
        turn.toolCallEntries.set(callId, entry);
      }
    });
    this._updateToolGroupSummary(chatId);
  }

  handleToolExecutingEvent(payload) {
    const call = payload.toolExecuting;
    if (!call) return;

    const chatId = payload.chatId || this.state.conversationId || "";
    const label = this.formatAgentLabel(payload);
    const turn = this._ensureToolGroup(chatId, label);

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

  handleToolResultEvent(payload) {
    const results = payload.toolResults || [];
    if (results.length === 0) return;

    const chatId = payload.chatId || this.state.conversationId || "";
    const label = this.formatAgentLabel(payload);
    const turn = this._ensureToolGroup(chatId, label);

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
        // Ephemeral tool completed — remove its entry entirely (no result to show)
        turn.toolCallEntries.get(callId).remove();
        turn.toolCallEntries.delete(callId);
      }
    });
    this._updateToolGroupSummary(chatId);
    // Close this chatId's turn — next content from the same chatId opens a fresh bubble.
    this.activeTurns.delete(chatId);
  }

  // ── Helpers ───────────────────────────────────────────────────────────────
  formatAgentLabel(payload) {
    // Always use the main agent's name for continuity, silencing sub-agent identity switches.
    return this.state.agentName || "Assistant";
  }

  renderMarkdown(text) {
    if (typeof marked === "undefined" || typeof DOMPurify === "undefined") return text;
    ensureMarkedCodeRenderer();
    const raw = marked.parse(text);
    return DOMPurify.sanitize(raw, {
      ADD_TAGS: ["button", "div", "span"],
      ADD_ATTR: ["class", "type"],
    });
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
        if (line.startsWith("event:")) eventType = line.slice(6).trim();
        else if (line.startsWith("data:")) data += line.slice(5).trim();
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
