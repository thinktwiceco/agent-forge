import { ChatManager } from "./chat.js";
import { ConversationManager } from "./conversations.js";
import { ConfigPanel } from "./config.js";
import { TodoManager } from "./todos.js";

// Load conversation ID from localStorage
function loadConversationId() {
  return localStorage.getItem('currentConversationId') || '';
}

// Save conversation ID to localStorage
function saveConversationId(id) {
  if (id) {
    localStorage.setItem('currentConversationId', id);
  } else {
    localStorage.removeItem('currentConversationId');
  }
}

const appState = {
  agentName: null,
  conversationId: loadConversationId(), // Load from storage
  events: new EventTarget(),
};

const chatManager = new ChatManager(appState);
const conversationManager = new ConversationManager(appState, chatManager);
const configPanel = new ConfigPanel(appState);
const todoManager = new TodoManager();

async function bootstrap() {
  await configPanel.load();
  await conversationManager.refreshList();
  conversationManager.bindNewChat();
  todoManager.start();
  
  // Auto-load last conversation if exists
  if (appState.conversationId) {
    await chatManager.loadConversation(appState.conversationId);
    conversationManager.setActive(appState.conversationId);
  }
}

bootstrap();
