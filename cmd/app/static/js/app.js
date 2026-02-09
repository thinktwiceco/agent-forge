import { ChatManager } from "./chat.js";
import { ConversationManager } from "./conversations.js";
import { ConfigPanel } from "./config.js";
import { TodoManager } from "./todos.js";

const appState = {
  agentName: null,
  conversationId: "",
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
}

bootstrap();
