export class TodoManager {
  constructor(appState) {
    this.state = appState;
    this.container = document.getElementById("todos-panel");
    this.pollInterval = null;
    this.todos = [];
    this.activePollMs = 3000;
    this.idlePollMs = 10000;
    this.agentActive = false;
    this.visible = !document.hidden;
  }

  start() {
    this.load();

    document.addEventListener("visibilitychange", () => {
      this.visible = !document.hidden;
      if (this.visible) {
        this.load();
        this._restartPolling();
      } else {
        this._stopPolling();
      }
    });

    this.state.events.addEventListener("agentStatusChanged", (event) => {
      const status = (event.detail?.status || "").toLowerCase();
      const wasActive = this.agentActive;
      this.agentActive = status === "streaming";
      if (this.agentActive !== wasActive) {
        this._restartPolling();
      }
      if (this.agentActive) {
        this.load();
      }
    });

    this.state.events.addEventListener("conversationUpdated", () => {
      if (this.visible) this.load();
    });

    this._restartPolling();
  }

  stop() {
    this._stopPolling();
  }

  _stopPolling() {
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
      this.pollInterval = null;
    }
  }

  _restartPolling() {
    this._stopPolling();
    if (!this.visible) return;

    const interval = this.agentActive ? this.activePollMs : this.idlePollMs;
    this.pollInterval = setInterval(() => this.load(), interval);
  }

  async load() {
    if (!this.visible) return;

    try {
      const res = await fetch("/api/todos");
      if (!res.ok) {
        throw new Error("failed to fetch todos");
      }
      const data = await res.json();
      this.todos = data.todos || [];
      this.render();
    } catch (err) {
      console.error("Failed to load todos:", err);
    }
  }

  render() {
    if (!this.container) return;

    const pendingTodos = this.todos.filter((t) => !t.completed);
    const completedTodos = this.todos.filter((t) => t.completed);

    let html = "";

    html += `
      <div class="todos-header">
        <h3>Active Tasks</h3>
        <span class="todos-count">${pendingTodos.length}</span>
      </div>
    `;

    if (pendingTodos.length > 0) {
      html += '<div class="todos-section">';
      html += '<h4>Pending</h4>';
      pendingTodos.forEach((todo) => {
        html += this.renderTodoItem(todo, false);
      });
      html += "</div>";
    }

    if (completedTodos.length > 0) {
      html += '<div class="todos-section">';
      html += '<h4>Completed</h4>';
      completedTodos.forEach((todo) => {
        html += this.renderTodoItem(todo, true);
      });
      html += "</div>";
    }

    if (this.todos.length === 0) {
      html += '<div class="todos-empty">No tasks yet.<br />They appear here when the agent uses the todo tool.</div>';
    }

    this.container.innerHTML = html;
  }

  renderTodoItem(todo, completed) {
    const statusClass = completed ? "completed" : "pending";
    const statusIcon = completed ? "✓" : "○";

    return `
      <div class="todo-item ${statusClass}">
        <div class="todo-status">${statusIcon}</div>
        <div class="todo-content">
          <div class="todo-title">${this.escapeHtml(todo.title)}</div>
          ${todo.description ? `<div class="todo-description">${this.escapeHtml(todo.description)}</div>` : ""}
        </div>
      </div>
    `;
  }

  escapeHtml(text) {
    const div = document.createElement("div");
    div.textContent = text;
    return div.innerHTML;
  }
}
