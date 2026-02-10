// Backup implementation of chat.js with thinking indicator
// This file demonstrates the changes needed for the thinking indicator feature

export class ChatManager {
  constructor(state) {
    this.state = state;
    this.statusEl = document.querySelector(".chat-status");
    this.messagesEl = document.querySelector(".chat-messages");
    this.inputEl = document.querySelector(".chat-input textarea");
    this.formEl = document.querySelector(".chat-input form");
    this.currentAssistantEl = null;
    this.currentIteration = null;
    this.thinkingMessageEl = null; // Track thinking message element
    
    this.initializeEventListeners();
  }

  setStatus(text) {
    if (this.statusEl) {
      this.statusEl.textContent = text;
    }
  }

  // Create thinking indicator HTML
  createThinkingIndicator() {
    return `
      <div class="thinking-indicator">
        Thinking
        <div class="thinking-dots">
          <div class="thinking-dot"></div>
          <div class="thinking-dot"></div>
          <div class="thinking-dot"></div>
        </div>  
      </div>`;
  }

  // Remove thinking indicator
  removeThinkingIndicator() {
    if (this.thinkingMessageEl) {
      this.thinkingMessageEl.remove();
      this.thinkingMessageEl = null;
    }
  }

  async startStream(message) {
    this.setStatus("Thinking");
    
    // Create and show thinking indicator immediately
    this.thinkingMessageEl = this.appendMessage(
      this.formatAgentLabel({agentName: "Assistant"}), 
      this.createThinkingIndicator(), 
      ["message", "msg-assistant", "msg-thinking"], 
      false
    );
    
    const query = this.state.conversationId !== "" 
      ? `?conversationId=${encodeURIComponent(this.state.conversationId)}` 
      : "";

    try {
      const res = await fetch(`/api/chat${query}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message }),
      });

      if (!res.ok || !res.body) {
        this.setStatus("Error");
        this.removeThinkingIndicator(); // Remove thinking indicator on error
        this.appendMessage("System", "Failed to start stream", [
          "message",
          "msg-tool-result-error",
        ]);
        return;
      }

      await parseSSEStream(res.body, (eventType, payload) => {
        this.handleStreamEvent(eventType, payload);
      });
    } catch (error) {
      this.setStatus("Error");
      this.removeThinkingIndicator(); // Remove thinking indicator on error
      this.appendMessage("System", "Stream error: " + error.message, [
        "message",
        "msg-tool-result-error",
      ]);
    }
  }

  handleStreamEvent(eventType, payload) {
    if (!payload) {
      return;
    }

    if (payload.chatId && payload.chatId !== this.state.conversationId) {
      this.state.conversationId = payload.chatId;
      localStorage.setItem('currentConversationId', payload.chatId);
      this.state.events.dispatchEvent(
        new CustomEvent("conversationIdUpdated", {
          detail: { id: payload.chatId },
        })
      );
    }

    // Handle thinking events from backend (optional enhancement)
    if (eventType === "thinking") {
      // Already handled by frontend, but could be used for backend-triggered thinking state
      return;
    }
    
    // Remove thinking indicator when first content arrives
    if (eventType === "content" && this.thinkingMessageEl) {
      this.removeThinkingIndicator();
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
      this.removeThinkingIndicator(); // Ensure thinking indicator is removed on completion
      this.setStatus("Idle");
      this.state.events.dispatchEvent(
        new CustomEvent("conversationUpdated", {
          detail: { id: this.state.conversationId },
        })
      );
    } else if (eventType === "error") {
      this.removeThinkingIndicator(); // Remove thinking indicator on error
      this.setStatus("Error");
      this.appendMessage("Error", payload.content || "Unknown error", [
        "message",
        "msg-tool-result-error",
      ]);
    }
  }

  // Helper method to format agent label (if not already exists)
  formatAgentLabel(payload) {
    return payload.agentName || "Assistant";
  }

  // Rest of the existing methods would go here...
  // handleContentEvent, handleToolCallEvent, handleToolExecutingEvent, handleToolResultEvent, etc.
}