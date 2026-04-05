export class TodoManager {
  constructor() {
    this.container = document.getElementById("todos-panel");
    this.pollInterval = null;
    this.todos = [];
  }

  start() {
    // Initial load
    this.load();
    
    // Poll every 3 seconds
    this.pollInterval = setInterval(() => this.load(), 3000);
  }

  stop() {
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
      this.pollInterval = null;
    }
  }

  async load() {
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

    // Header with count
    html += `
      <div class="todos-header">
        <h3>Active Tasks</h3>
        <span class="todos-count">${pendingTodos.length}</span>
      </div>
    `;

    // Pending todos
    if (pendingTodos.length > 0) {
      html += '<div class="todos-section">';
      html += '<h4>Pending</h4>';
      pendingTodos.forEach((todo) => {
        html += this.renderTodoItem(todo, false);
      });
      html += "</div>";
    }

    // Completed todos
    if (completedTodos.length > 0) {
      html += '<div class="todos-section">';
      html += '<h4>Completed</h4>';
      completedTodos.forEach((todo) => {
        html += this.renderTodoItem(todo, true);
      });
      html += "</div>";
    }

    // Empty state
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
