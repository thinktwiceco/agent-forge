export class ConversationManager {
  constructor(appState, chatManager) {
    this.state = appState;
    this.chatManager = chatManager;
    this.listEl = document.getElementById("conversation-list");
    this.newButton = document.getElementById("new-chat");

    this.state.events.addEventListener("conversationUpdated", () => {
      this.refreshList();
    });
    this.state.events.addEventListener("conversationIdUpdated", (event) => {
      const id = event.detail?.id;
      this.refreshList().then(() => {
        if (id) {
          this.setActive(id);
        }
      });
    });
  }

  bindNewChat() {
    if (!this.newButton) {
      return;
    }
    this.newButton.addEventListener("click", () => {
      this.clearActive();
      this.chatManager.startNewConversation();
    });
  }

  async refreshList() {
    try {
      const res = await fetch("/api/conversations");
      if (!res.ok) {
        throw new Error("failed");
      }
      const list = await res.json();
      this.renderList(list);
    } catch (err) {
      this.listEl.innerHTML = "";
    }
  }

  renderList(items) {
    this.listEl.innerHTML = "";
    items.forEach((item) => {
      const el = document.createElement("div");
      el.className = "conversation-item";
      el.dataset.id = item.id;
      const date = item.updatedAt
        ? new Date(item.updatedAt).toLocaleString()
        : "Unknown";
      el.textContent = `${item.id.slice(0, 8)} • ${date}`;
      el.addEventListener("click", () => {
        this.setActive(item.id);
        this.chatManager.loadConversation(item.id);
      });
      this.listEl.appendChild(el);
    });
  }

  setActive(id) {
    this.clearActive();
    const target = this.listEl.querySelector(`[data-id="${id}"]`);
    if (target) {
      target.classList.add("active");
    }
  }

  clearActive() {
    const active = this.listEl.querySelector(".conversation-item.active");
    if (active) {
      active.classList.remove("active");
    }
  }
}
