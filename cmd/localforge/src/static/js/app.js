import { ChatManager } from "./chat.js";
import { ConversationManager } from "./conversations.js";
import { TodoManager } from "./todos.js";
import { FileSystemManager } from "./fs.js";
import { setupAuthUI } from "./auth.js";

function loadConversationId() {
  return localStorage.getItem('currentConversationId') || '';
}

function saveConversationId(id) {
  if (id) {
    localStorage.setItem('currentConversationId', id);
  } else {
    localStorage.removeItem('currentConversationId');
  }
}

const appState = {
  agentName: null,
  conversationId: loadConversationId(),
  events: new EventTarget(),
};

const chatManager = new ChatManager(appState);
const conversationManager = new ConversationManager(appState, chatManager);
const todoManager = new TodoManager(appState);
const fsManager = new FileSystemManager();

function setupSidebarNavActive() {
  const path = (window.location.pathname || "/").replace(/\/$/, "") || "/";
  document.querySelectorAll(".sidebar-nav-item").forEach((el) => {
    const href = el.getAttribute("href");
    if (!href) return;
    const hrefPath = href.replace(/\/$/, "") || "/";
    if (path === hrefPath) el.classList.add("active");
  });
}

async function loadAgentName() {
  try {
    const res = await fetch("/api/config");
    if (!res.ok) return;
    const cfg = await res.json();
    appState.agentName = cfg.name || null;
    const nameLabel = document.getElementById("agent-name-label");
    if (nameLabel && cfg.name) nameLabel.textContent = cfg.name;
    const welcomeTitle = document.getElementById("welcome-agent-name");
    if (welcomeTitle && cfg.name) welcomeTitle.textContent = cfg.name;
    if (cfg.name) document.title = cfg.name;
    return cfg;
  } catch {
    // non-critical
  }
  return null;
}

async function bootstrap() {
  await setupAuthUI();
  setupSidebarNavActive();
  const cfg = await loadAgentName();
  await conversationManager.refreshList();
  conversationManager.bindNewChat();

  // Only poll todos when the todo plugin is active.
  const plugins = cfg?.plugins || [];
  if (plugins.includes("todo")) {
    todoManager.start();
  }

  // Start permanent heartbeat listener so autonomous agent turns are always visible.
  chatManager.startHeartbeatListener();

  if (appState.conversationId) {
    await chatManager.loadConversation(appState.conversationId);
    conversationManager.setActive(appState.conversationId);
  }
}

bootstrap();
