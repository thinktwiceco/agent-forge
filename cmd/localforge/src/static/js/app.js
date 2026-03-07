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
const todoManager = new TodoManager();
const fsManager = new FileSystemManager();

async function loadAgentName() {
  try {
    const res = await fetch("/api/config");
    if (!res.ok) return;
    const cfg = await res.json();
    appState.agentName = cfg.name || null;
    const nameLabel = document.getElementById("agent-name-label");
    if (nameLabel && cfg.name) nameLabel.textContent = cfg.name;
    if (cfg.name) document.title = cfg.name;
  } catch {
    // non-critical
  }
}

async function bootstrap() {
  await setupAuthUI();
  await loadAgentName();
  await conversationManager.refreshList();
  conversationManager.bindNewChat();
  todoManager.start();

  if (appState.conversationId) {
    await chatManager.loadConversation(appState.conversationId);
    conversationManager.setActive(appState.conversationId);
  }
}

bootstrap();
