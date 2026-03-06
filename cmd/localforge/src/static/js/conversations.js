function formatRelativeTime(isoString) {
  if (!isoString) return "";
  const now = Date.now();
  const then = new Date(isoString).getTime();
  const diff = Math.floor((now - then) / 1000); // seconds

  if (diff < 60)      return "just now";
  if (diff < 3600)    return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400)   return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 604800)  return `${Math.floor(diff / 86400)}d ago`;
  return new Date(isoString).toLocaleDateString();
}

export class ConversationManager {
  constructor(appState, chatManager) {
    this.state = appState;
    this.chatManager = chatManager;
    this.listEl = document.getElementById("conversation-list");
    this.newButton = document.getElementById("new-chat");
    this.newButtonHeader = document.getElementById("new-chat-header");

    this.state.events.addEventListener("conversationUpdated", () => {
      this.refreshList();
    });
    this.state.events.addEventListener("conversationIdUpdated", (event) => {
      const id = event.detail?.id;
      this.refreshList().then(() => {
        if (id) this.setActive(id);
      });
    });
  }

  bindNewChat() {
    const handler = () => {
      this.clearActive();
      this.chatManager.startNewConversation();
    };
    if (this.newButton) this.newButton.addEventListener("click", handler);
    if (this.newButtonHeader) this.newButtonHeader.addEventListener("click", handler);
  }

  async refreshList() {
    try {
      const res = await fetch("/api/conversations");
      if (!res.ok) throw new Error("failed");
      const list = await res.json();
      this.renderList(list);
    } catch {
      this.listEl.innerHTML = "";
    }
  }

  renderList(items) {
    this.listEl.innerHTML = "";
    items.forEach((item) => {
      const el = document.createElement("div");
      el.className = "conversation-item";
      el.dataset.id = item.id;

      const label = item.title || item.id.slice(0, 8).toUpperCase();
      const relTime = formatRelativeTime(item.updatedAt);

      const titleEl = document.createElement("span");
      titleEl.className = "conv-title";
      titleEl.textContent = label;

      const timeEl = document.createElement("span");
      timeEl.className = "conv-time";
      timeEl.textContent = relTime;

      const renameBtn = document.createElement("button");
      renameBtn.className = "conv-rename";
      renameBtn.title = "Rename conversation";
      renameBtn.textContent = "✎";
      renameBtn.addEventListener("click", (e) => {
        e.stopPropagation();
        this._startInlineRename(el, titleEl, item.id, label);
      });

      const deleteBtn = document.createElement("button");
      deleteBtn.className = "conv-delete";
      deleteBtn.title = "Delete conversation";
      deleteBtn.textContent = "×";
      deleteBtn.addEventListener("click", async (e) => {
        e.stopPropagation();
        await this.deleteConversation(item.id);
      });

      el.appendChild(titleEl);
      el.appendChild(timeEl);
      el.appendChild(renameBtn);
      el.appendChild(deleteBtn);

      el.addEventListener("click", () => {
        this.setActive(item.id);
        this.chatManager.loadConversation(item.id);
      });

      this.listEl.appendChild(el);
    });
  }

  _startInlineRename(itemEl, titleEl, id, currentLabel) {
    if (itemEl.querySelector(".conv-rename-input")) return; // already editing

    const input = document.createElement("input");
    input.type = "text";
    input.className = "conv-rename-input";
    input.value = currentLabel;

    titleEl.replaceWith(input);
    input.focus();
    input.select();

    const commit = async () => {
      const newTitle = input.value.trim();
      if (newTitle && newTitle !== currentLabel) {
        try {
          await fetch(`/api/conversations/${id}/title`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ title: newTitle }),
          });
        } catch {
          // Silently ignore — list will resync
        }
      }
      await this.refreshList();
    };

    const cancel = async () => {
      await this.refreshList();
    };

    input.addEventListener("keydown", (e) => {
      if (e.key === "Enter") { e.preventDefault(); commit(); }
      if (e.key === "Escape") { e.preventDefault(); cancel(); }
    });
    input.addEventListener("blur", commit);
  }

  async deleteConversation(id) {
    try {
      const res = await fetch(`/api/conversations/${id}`, { method: "DELETE" });
      if (!res.ok && res.status !== 404) throw new Error("failed");
      // If we deleted the active conversation, start fresh
      if (this.state.conversationId === id) {
        this.clearActive();
        this.chatManager.startNewConversation();
      }
      await this.refreshList();
    } catch {
      // Silently ignore — conversation list will resync on next event
    }
  }

  setActive(id) {
    this.clearActive();
    const target = this.listEl.querySelector(`[data-id="${id}"]`);
    if (target) target.classList.add("active");
  }

  clearActive() {
    const active = this.listEl.querySelector(".conversation-item.active");
    if (active) active.classList.remove("active");
  }
}
