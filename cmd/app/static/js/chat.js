// Generic tool call formatting - no tool-specific logic
const SENSITIVE_KEYS = new Set([
  "password", "passwd", "secret", "token", "apikey", "api_key", "apiKey",
  "key", "authorization", "auth", "credentials"
]);

function truncate(str, maxLen = 50) {
  if (typeof str !== "string") return String(str);
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen) + "…";
}

function formatArguments(args) {
  if (!args || typeof args !== "object") return "";
  const parts = [];
  for (const [key, value] of Object.entries(args)) {
    const k = String(key).toLowerCase();
    if (SENSITIVE_KEYS.has(k)) continue;
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

export class ChatManager {
  constructor(appState) {
    this.state = appState;
    this.messagesEl = document.getElementById("chat-messages");
    this.formEl = document.getElementById("chat-form");
    this.inputEl = document.getElementById("chat-input");
    this.statusEl = document.getElementById("chat-status");
    this.stopBtn = document.getElementById("stop-btn");
    this.currentAssistantEl = null;
    this.currentIteration = null; // Track current iteration for message grouping
    this.abortController = null; // AbortController for cancelling streams

    this.formEl.addEventListener("submit", (event) => {
      event.preventDefault();
      this.handleSubmit();
    });

    // Handle Enter key to send message (Shift+Enter for new line)
    this.inputEl.addEventListener("keydown", (event) => {
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        this.handleSubmit();
      }
    });

    // Handle stop button click
    if (this.stopBtn) {
      this.stopBtn.addEventListener("click", () => {
        this.stopStream();
      });
    }
  }

  setStatus(text) {
    if (this.statusEl) {
      this.statusEl.textContent = text;
    }
  }

  showStopButton() {
    if (this.stopBtn) {
      this.stopBtn.style.display = "block";
    }
  }

  hideStopButton() {
    if (this.stopBtn) {
      this.stopBtn.style.display = "none";
    }
  }

  async stopStream() {
    // Abort the fetch request
    if (this.abortController) {
      this.abortController.abort();
    }

    // Send stop request to backend
    if (this.state.conversationId) {
      try {
        await fetch("/api/chat/stop", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ conversationId: this.state.conversationId }),
        });
      } catch (err) {
        // Ignore errors from stop endpoint (request might already be done)
        console.log("Stop request failed:", err);
      }
    }

    // Update UI
    this.hideStopButton();
    this.setStatus("Stopped");
    this.appendMessage("System", "Agent stopped", [
      "message",
      "msg-tool-result-error",
    ]);
  }

  clearMessages() {
    this.messagesEl.innerHTML = "";
  }

  async handleSubmit() {
    const message = this.inputEl.value.trim();
    if (!message) {
      return;
    }
    this.inputEl.value = "";
    this.appendMessage("You", message, ["message", "msg-user"]);
    await this.startStream(message);
  }

  async startNewConversation() {
    this.state.conversationId = "";
    localStorage.removeItem('currentConversationId');
    this.currentAssistantEl = null;
    this.currentIteration = null;
    this.setStatus("Idle");
    this.clearMessages();
  }

  async loadConversation(conversationId) {
    this.state.conversationId = conversationId;
    localStorage.setItem('currentConversationId', conversationId);
    this.currentAssistantEl = null;
    this.currentIteration = null;
    this.clearMessages();
    this.setStatus("Loading");

    try {
      const res = await fetch(`/api/conversations/${conversationId}`);
      if (!res.ok) {
        throw new Error("failed to load conversation");
      }
      const history = await res.json();
      history.forEach((msg) => {
        this.renderHistoryMessage(msg);
      });
      this.setStatus("Idle");
    } catch (err) {
      this.setStatus("Error");
      this.appendMessage("System", "Failed to load conversation", [
        "message",
        "msg-tool-result-error",
      ]);
    }
  }

  renderHistoryMessage(msg) {
    const role = msg.role || "assistant";
    const content = msg.content || "";
    if (role === "user") {
      this.appendMessage("You", content, ["message", "msg-user"]);
      return;
    }
    if (role === "assistant") {
      const el = this.appendMessage("Assistant", content, [
        "message",
        "msg-assistant",
      ], true); // Enable markdown for assistant messages
      if (msg.toolCalls && msg.toolCalls.length) {
        const body = msg.toolCalls
          .map((call) => formatToolCallSummary(call))
          .join(" | ");
        this.appendMessage("Tool call", body, [
          "message",
          "msg-tool-call",
        ]);
      }
      return;
    }
    if (role === "tool") {
      this.appendMessage("Tool result", content, [
        "message",
        "msg-tool-result-success",
      ]);
      return;
    }
    this.appendMessage("System", content, ["message"]);
  }

  async startStream(message) {
    this.setStatus("Streaming");
    this.showStopButton();
    
    // Create new AbortController for this request
    this.abortController = new AbortController();
    
    const query =
      this.state.conversationId !== ""
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
        this.appendMessage("System", "Failed to start stream", [
          "message",
          "msg-tool-result-error",
        ]);
        return;
      }

      await parseSSEStream(res.body, (eventType, payload) => {
        this.handleStreamEvent(eventType, payload);
      });
    } catch (err) {
      if (err.name === "AbortError") {
        // Request was aborted by user, this is expected
        console.log("Stream aborted by user");
      } else {
        this.setStatus("Error");
        this.hideStopButton();
        this.appendMessage("System", `Stream error: ${err.message}`, [
          "message",
          "msg-tool-result-error",
        ]);
      }
    } finally {
      this.abortController = null;
    }
  }

  handleStreamEvent(eventType, payload) {
    if (!payload) {
      return;
    }

    if (payload.chatId && payload.chatId !== this.state.conversationId) {
      this.state.conversationId = payload.chatId;
      // Persist to localStorage
      localStorage.setItem('currentConversationId', payload.chatId);
      this.state.events.dispatchEvent(
        new CustomEvent("conversationIdUpdated", {
          detail: { id: payload.chatId },
        })
      );
    }

    if (eventType === "content") {
      this.handleContentEvent(payload);
    } else if (eventType === "tool_call") {
      this.handleToolCallEvent(payload);
    } else if (eventType === "tool_executing") {
      this.handleToolExecutingEvent(payload);
    } else if (eventType === "tool_result") {
      this.handleToolResultEvent(payload);
    } else if (eventType === "completed") {
      this.setStatus("Idle");
      this.hideStopButton();
      this.state.events.dispatchEvent(
        new CustomEvent("conversationUpdated", {
          detail: { id: this.state.conversationId },
        })
      );
    } else if (eventType === "error") {
      this.setStatus("Error");
      this.hideStopButton();
      this.appendMessage("Error", payload.content || "Unknown error", [
        "message",
        "msg-tool-result-error",
      ]);
    }
  }

  handleContentEvent(payload) {
    const delta = payload.delta || payload.content || "";
    if (!delta) {
      return;
    }
    
    // Check if iteration has changed (new response turn)
    const iteration = payload.iteration !== undefined ? payload.iteration : null;
    if (iteration !== null && iteration !== this.currentIteration) {
      // New iteration detected, create a new message bubble
      this.currentIteration = iteration;
      this.currentAssistantEl = null;
    }
    
    if (!this.currentAssistantEl) {
      this.currentAssistantEl = this.appendMessage(
        this.formatAgentLabel(payload),
        "",
        ["message", "msg-assistant", this.subagentClass(payload)],
        true // Enable markdown rendering
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
    const calls = payload.toolCalls || [];
    const body =
      calls.length > 0
        ? calls.map((call) => formatToolCallSummary(call)).join(" | ")
        : "Tool call requested";
    this.appendMessage(this.formatAgentLabel(payload), body, [
      "message",
      "msg-tool-call",
      this.subagentClass(payload),
    ]);
    // Reset assistant element so next content appears after tool calls
    this.currentAssistantEl = null;
  }

  handleToolExecutingEvent(payload) {
    const call = payload.toolExecuting;
    if (!call) return;
    const body = `Running: ${formatToolCallSummary(call)}`;
    this.appendMessage(this.formatAgentLabel(payload), body, [
      "message",
      "msg-tool-executing",
      this.subagentClass(payload),
    ]);
  }

  handleToolResultEvent(payload) {
    const results = payload.toolResults || [];
    if (results.length === 0) {
      return;
    }

    results.forEach((result) => {
      if (!result.success) {
        const toolName = result.toolName || "Tool";
        const errorMsg = result.error || "Unknown error";
        this.appendMessage(
          this.formatAgentLabel(payload),
          `✗ ${toolName}: ${errorMsg}`,
          ["message", "msg-tool-result-error", this.subagentClass(payload)]
        );
      } else if (!result.ephemeral) {
        const summary = formatToolResultSummary(result);
        if (summary) {
          this.appendMessage(
            this.formatAgentLabel(payload),
            summary,
            ["message", "msg-tool-result-success", this.subagentClass(payload)]
          );
        }
      }
    });
  }

  subagentClass(payload) {
    if (payload.agentName && this.state.agentName) {
      if (payload.agentName !== this.state.agentName) {
        return "msg-subagent";
      }
    }
    return "";
  }

  formatAgentLabel(payload) {
    if (payload.agentName) {
      return payload.agentName;
    }
    return "Assistant";
  }

  renderMarkdown(text) {
    if (typeof marked === 'undefined' || typeof DOMPurify === 'undefined') {
      return text; // Fallback to plain text if libraries not loaded
    }
    const html = marked.parse(text);
    return DOMPurify.sanitize(html);
  }

  appendMessage(label, content, classes, enableMarkdown = false) {
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
    this.messagesEl.appendChild(messageEl);
    this.scrollToBottom();
    return messageEl;
  }

  scrollToBottom() {
    this.messagesEl.scrollTop = this.messagesEl.scrollHeight;
  }
}

async function parseSSEStream(body, onEvent) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { value, done } = await reader.read();
    if (done) {
      break;
    }
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split("\n\n");
    buffer = parts.pop() || "";

    parts.forEach((part) => {
      const lines = part.split("\n");
      let eventType = "message";
      let data = "";
      lines.forEach((line) => {
        if (line.startsWith("event:")) {
          eventType = line.slice(6).trim();
        } else if (line.startsWith("data:")) {
          data += line.slice(5).trim();
        }
      });
      if (!data) {
        return;
      }
      try {
        const payload = JSON.parse(data);
        onEvent(eventType, payload);
      } catch (err) {
        onEvent("error", { content: "Failed to parse stream data" });
      }
    });
  }
}
