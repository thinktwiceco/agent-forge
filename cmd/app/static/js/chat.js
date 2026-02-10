export class ChatManager {
  constructor(appState) {
    this.state = appState;
    this.messagesEl = document.getElementById("chat-messages");
    this.formEl = document.getElementById("chat-form");
    this.inputEl = document.getElementById("chat-input");
    this.statusEl = document.getElementById("chat-status");
    this.currentAssistantEl = null;
    this.currentIteration = null; // Track current iteration for message grouping

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
  }

  setStatus(text) {
    if (this.statusEl) {
      this.statusEl.textContent = text;
    }
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
        const toolNames = msg.toolCalls
          .map((call) => call.function?.name || call.name || "Unknown tool")
          .join(", ");
        this.appendMessage("Tool call", toolNames, [
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
    const query =
      this.state.conversationId !== ""
        ? `?conversationId=${encodeURIComponent(this.state.conversationId)}`
        : "";

    const res = await fetch(`/api/chat${query}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ message }),
    });

    if (!res.ok || !res.body) {
      this.setStatus("Error");
      this.appendMessage("System", "Failed to start stream", [
        "message",
        "msg-tool-result-error",
      ]);
      return;
    }

    await parseSSEStream(res.body, (eventType, payload) => {
      this.handleStreamEvent(eventType, payload);
    });
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
      this.state.events.dispatchEvent(
        new CustomEvent("conversationUpdated", {
          detail: { id: this.state.conversationId },
        })
      );
    } else if (eventType === "error") {
      this.setStatus("Error");
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
        ? calls
            .map((call) => call.function?.name || call.name || "Unknown tool")
            .join(", ")
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
    // Skip displaying tool executing events to reduce verbosity
    // Tool calls already show what's being executed
  }

  handleToolResultEvent(payload) {
    const results = payload.toolResults || [];
    if (results.length === 0) {
      return;
    }
    
    // Show only errors, skip successful tool results to reduce verbosity
    results.forEach((result) => {
      if (!result.success) {
        const toolName = result.toolName || "Tool";
        const errorMsg = result.error || "Unknown error";
        this.appendMessage(
          this.formatAgentLabel(payload),
          `✗ ${toolName}: ${errorMsg}`,
          ["message", "msg-tool-result-error", this.subagentClass(payload)]
        );
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
