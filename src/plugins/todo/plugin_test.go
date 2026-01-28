package todo

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// TestTodoPlugin_Name tests the Name() method
func TestTodoPlugin_Name(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	if plugin.Name() != PLUGIN_NAME {
		t.Errorf("Expected plugin name '%s', got '%s'", PLUGIN_NAME, plugin.Name())
	}
}

// TestTodoPlugin_SystemPrompt tests the SystemPrompt() method
func TestTodoPlugin_SystemPrompt(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	prompt := plugin.SystemPrompt()

	if prompt == "" {
		t.Error("SystemPrompt() should return a non-empty string")
	}

	if !strings.Contains(prompt, "todo_handler") {
		t.Error("SystemPrompt() should mention todo_handler tool")
	}

	if !strings.Contains(prompt, "reasoning agent") {
		t.Error("SystemPrompt() should mention reasoning agent")
	}
}

// TestTodoPlugin_Tools tests the Tools() method
func TestTodoPlugin_Tools(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	if tools[0].GetName() != TODO_HANDLER_TOOL {
		t.Errorf("Expected tool name '%s', got '%s'", TODO_HANDLER_TOOL, tools[0].GetName())
	}
}

// TestTodoPlugin_On tests the On() method for hook registration
func TestTodoPlugin_On(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	// Test EventToolExecution hook
	hook := plugin.On(core.EventToolExecution)
	if hook == nil {
		t.Error("Expected non-nil hook for EventToolExecution")
	}

	// Test other events should return nil
	hook = plugin.On(core.EventNewUserMessage)
	if hook != nil {
		t.Error("Expected nil hook for EventNewUserMessage")
	}

	hook = plugin.On(core.EventAgentInitialization)
	if hook != nil {
		t.Error("Expected nil hook for EventAgentInitialization")
	}
}

// TestTodoPlugin_AddTodoItem tests adding a single todo item
func TestTodoPlugin_AddTodoItem(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	err := plugin.addTodoItem("Test Todo", "Test Description")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	items := plugin.getTodoItems()
	if len(items) != 1 {
		t.Errorf("Expected 1 todo item, got %d", len(items))
	}

	if items[0].Title != "Test Todo" {
		t.Errorf("Expected title 'Test Todo', got '%s'", items[0].Title)
	}

	if items[0].Description != "Test Description" {
		t.Errorf("Expected description 'Test Description', got '%s'", items[0].Description)
	}

	if items[0].Completed {
		t.Error("New todo item should not be completed")
	}

	if items[0].ID == "" {
		t.Error("Todo item should have an ID")
	}

	if items[0].CreatedAt.IsZero() {
		t.Error("Todo item should have CreatedAt timestamp")
	}

	if items[0].UpdatedAt.IsZero() {
		t.Error("Todo item should have UpdatedAt timestamp")
	}
}

// TestTodoPlugin_GetTodoItems tests retrieving all todo items
func TestTodoPlugin_GetTodoItems(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	// Initially should be empty
	items := plugin.getTodoItems()
	if len(items) != 0 {
		t.Errorf("Expected 0 todo items initially, got %d", len(items))
	}

	// Add some items
	if err := plugin.addTodoItem("Todo 1", "Description 1"); err != nil {
		t.Fatalf("Failed to add todo 1: %v", err)
	}
	if err := plugin.addTodoItem("Todo 2", "Description 2"); err != nil {
		t.Fatalf("Failed to add todo 2: %v", err)
	}
	if err := plugin.addTodoItem("Todo 3", "Description 3"); err != nil {
		t.Fatalf("Failed to add todo 3: %v", err)
	}

	items = plugin.getTodoItems()
	if len(items) != 3 {
		t.Errorf("Expected 3 todo items, got %d", len(items))
	}
}

// TestTodoPlugin_GetTodoItem tests retrieving a todo by ID
func TestTodoPlugin_GetTodoItem(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	if err := plugin.addTodoItem("Test Todo", "Test Description"); err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}
	items := plugin.getTodoItems()
	expectedID := items[0].ID

	// Test finding by ID
	found := plugin.getTodoItem(expectedID)
	if found == nil {
		t.Fatal("Expected to find todo item by ID")
	}

	if found != nil && found.Title != "Test Todo" {
		t.Errorf("Expected title 'Test Todo', got '%s'", found.Title)
	}

	// Test non-existent ID
	notFound := plugin.getTodoItem("non-existent-id")
	if notFound != nil {
		t.Error("Expected nil for non-existent ID")
	}
}

// TestTodoPlugin_GetTodoItemByTitle tests retrieving a todo by title
func TestTodoPlugin_GetTodoItemByTitle(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	if err := plugin.addTodoItem("Unique Title", "Description"); err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}

	// Test finding by title
	found := plugin.getTodoItemByTitle("Unique Title")
	if found == nil {
		t.Fatal("Expected to find todo item by title")
	}

	if found != nil && found.Description != "Description" {
		t.Errorf("Expected description 'Description', got '%s'", found.Description)
	}

	// Test non-existent title
	notFound := plugin.getTodoItemByTitle("Non-existent Title")
	if notFound != nil {
		t.Error("Expected nil for non-existent title")
	}
}

// TestTodoPlugin_UpdateTodoStatus tests updating todo status by ID
func TestTodoPlugin_UpdateTodoStatus(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	if err := plugin.addTodoItem("Test Todo", "Description"); err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}
	items := plugin.getTodoItems()
	id := items[0].ID

	// Verify initial state
	if items[0].Completed {
		t.Error("Todo should not be completed initially")
	}

	// Update to completed
	err := plugin.updateTodoStatus(id, true)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	updated := plugin.getTodoItem(id)
	if !updated.Completed {
		t.Error("Todo should be completed after update")
	}

	if updated.UpdatedAt.Before(items[0].UpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}

	// Update back to incomplete
	err = plugin.updateTodoStatus(id, false)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	updated = plugin.getTodoItem(id)
	if updated.Completed {
		t.Error("Todo should not be completed after second update")
	}

	// Test updating non-existent ID
	err = plugin.updateTodoStatus("non-existent-id", true)
	if err == nil {
		t.Error("Expected error for non-existent ID")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

// TestTodoPlugin_UpdateTodoStatusByTitle tests updating todo status by title
func TestTodoPlugin_UpdateTodoStatusByTitle(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	if err := plugin.addTodoItem("Test Todo", "Description"); err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}

	// Update to completed
	err := plugin.updateTodoStatusByTitle("Test Todo", true)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	updated := plugin.getTodoItemByTitle("Test Todo")
	if !updated.Completed {
		t.Error("Todo should be completed after update")
	}

	// Test updating non-existent title
	err = plugin.updateTodoStatusByTitle("Non-existent", true)
	if err == nil {
		t.Error("Expected error for non-existent title")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

// TestTodoPlugin_AddBulkTodos tests adding multiple todos at once
func TestTodoPlugin_AddBulkTodos(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	todos := []map[string]string{
		{"title": "Todo 1", "description": "Description 1"},
		{"title": "Todo 2", "description": "Description 2"},
		{"title": "Todo 3", "description": "Description 3"},
	}

	err := plugin.addBulkTodos(todos)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	items := plugin.getTodoItems()
	if len(items) != 3 {
		t.Errorf("Expected 3 todo items, got %d", len(items))
	}

	// Verify all items were added correctly
	for i, item := range items {
		expectedTitle := todos[i]["title"]
		if item.Title != expectedTitle {
			t.Errorf("Expected title '%s', got '%s'", expectedTitle, item.Title)
		}
	}
}

// TestTodoPlugin_HandleToolExecution tests the tool execution hook
func TestTodoPlugin_HandleToolExecution(t *testing.T) {
	var callbackInvoked bool
	var callbackTodos []*TodoItem

	callback := func(todos []*TodoItem) {
		callbackInvoked = true
		callbackTodos = todos
	}

	plugin := NewTodoPlugin(callback)

	// Add a todo item
	if err := plugin.addTodoItem("Test Todo", "Description"); err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}

	// Create a mock agent (we only need it for the hook signature)
	var mockAgent *agents.Agent

	// Test with todo_handler tool
	toolResult := &llms.ToolResult{
		ToolName: TODO_HANDLER_TOOL,
		Success:  true,
	}

	err := plugin.handleToolExecution(mockAgent, toolResult)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !callbackInvoked {
		t.Error("Expected callback to be invoked")
	}

	if len(callbackTodos) != 1 {
		t.Errorf("Expected callback to receive 1 todo, got %d", len(callbackTodos))
	}

	// Test with different tool (should not invoke callback)
	callbackInvoked = false
	toolResult.ToolName = "other_tool"

	err = plugin.handleToolExecution(mockAgent, toolResult)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if callbackInvoked {
		t.Error("Callback should not be invoked for other tools")
	}
}

// TestTodoPlugin_HandleToolExecution_NoCallback tests hook without callback
func TestTodoPlugin_HandleToolExecution_NoCallback(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	toolResult := &llms.ToolResult{
		ToolName: TODO_HANDLER_TOOL,
		Success:  true,
	}

	var mockAgent *agents.Agent
	err := plugin.handleToolExecution(mockAgent, toolResult)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	// Should not panic even without callback
}

// TestTodoHandlerTool_AddTodo tests the addTodo action
func TestTodoHandlerTool_AddTodo(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action":      "addTodo",
		"title":       "Test Todo",
		"description": "Test Description",
	}

	result := tool.Call(agentContext, args)
	if !result.Success() {
		t.Errorf("Expected success, got error: %s", result.Error())
	}

	if result.Data() != "Todo item added successfully" {
		t.Errorf("Expected success message, got: %s", result.Data())
	}

	// Verify todo was added
	items := plugin.getTodoItems()
	if len(items) != 1 {
		t.Errorf("Expected 1 todo item, got %d", len(items))
	}
}

// TestTodoHandlerTool_AddTodo_EmptyTitle tests addTodo with empty title
func TestTodoHandlerTool_AddTodo_EmptyTitle(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action":      "addTodo",
		"title":       "",
		"description": "Description",
	}

	result := tool.Call(agentContext, args)
	if !result.Success() {
		t.Errorf("Expected success even with empty title, got error: %s", result.Error())
	}
}

// TestTodoHandlerTool_AddBulkTodos tests the addBulkTodos action
func TestTodoHandlerTool_AddBulkTodos(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action": "addBulkTodos",
		"todos": []any{
			map[string]any{"title": "Todo 1", "description": "Desc 1"},
			map[string]any{"title": "Todo 2", "description": "Desc 2"},
		},
	}

	result := tool.Call(agentContext, args)
	if !result.Success() {
		t.Errorf("Expected success, got error: %s", result.Error())
	}

	if !strings.Contains(result.Data(), "Successfully added 2 todo items") {
		t.Errorf("Expected success message with count, got: %s", result.Data())
	}

	// Verify todos were added
	items := plugin.getTodoItems()
	if len(items) != 2 {
		t.Errorf("Expected 2 todo items, got %d", len(items))
	}
}

// TestTodoHandlerTool_AddBulkTodos_EmptyArray tests addBulkTodos with empty array
func TestTodoHandlerTool_AddBulkTodos_EmptyArray(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action": "addBulkTodos",
		"todos":  []any{},
	}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for empty array")
	}

	if !strings.Contains(result.Error(), "cannot be empty") {
		t.Errorf("Error should mention empty array, got: %s", result.Error())
	}
}

// TestTodoHandlerTool_AddBulkTodos_InvalidType tests addBulkTodos with invalid type
func TestTodoHandlerTool_AddBulkTodos_InvalidType(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action": "addBulkTodos",
		"todos":  "not an array",
	}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for invalid type")
	}

	if !strings.Contains(result.Error(), "array") {
		t.Errorf("Error should mention array type, got: %s", result.Error())
	}
}

// TestTodoHandlerTool_AddBulkTodos_InvalidItemType tests addBulkTodos with invalid item type
func TestTodoHandlerTool_AddBulkTodos_InvalidItemType(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action": "addBulkTodos",
		"todos": []any{
			"not an object",
		},
	}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for invalid item type")
	}

	if !strings.Contains(result.Error(), "must be an object") {
		t.Errorf("Error should mention object type, got: %s", result.Error())
	}
}

// TestTodoHandlerTool_AddBulkTodos_MissingTitle tests addBulkTodos with missing title
func TestTodoHandlerTool_AddBulkTodos_MissingTitle(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action": "addBulkTodos",
		"todos": []any{
			map[string]any{"description": "Desc without title"},
		},
	}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for missing title")
	}

	if !strings.Contains(result.Error(), "title") {
		t.Errorf("Error should mention title, got: %s", result.Error())
	}
}

// TestTodoHandlerTool_GetTodos tests the getTodos action
func TestTodoHandlerTool_GetTodos(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	// Add some todos
	if err := plugin.addTodoItem("Todo 1", "Description 1"); err != nil {
		t.Fatalf("Failed to add todo 1: %v", err)
	}
	if err := plugin.addTodoItem("Todo 2", "Description 2"); err != nil {
		t.Fatalf("Failed to add todo 2: %v", err)
	}

	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action": "getTodos",
	}

	result := tool.Call(agentContext, args)
	if !result.Success() {
		t.Errorf("Expected success, got error: %s", result.Error())
	}

	// Parse JSON response
	var items []TodoItem
	err := json.Unmarshal([]byte(result.Data()), &items)
	if err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 todo items in JSON, got %d", len(items))
	}

	// Verify ephemeral response
	if !result.Ephemeral() {
		t.Error("getTodos should return ephemeral response")
	}
}

// TestTodoHandlerTool_GetTodos_Empty tests getTodos with no todos
func TestTodoHandlerTool_GetTodos_Empty(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action": "getTodos",
	}

	result := tool.Call(agentContext, args)
	if !result.Success() {
		t.Errorf("Expected success even with empty list, got error: %s", result.Error())
	}

	var items []TodoItem
	err := json.Unmarshal([]byte(result.Data()), &items)
	if err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected empty array, got %d items", len(items))
	}
}

// TestTodoHandlerTool_UpdateTodo_ByID tests updateTodo action with ID
func TestTodoHandlerTool_UpdateTodo_ByID(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	// Add a todo
	if err := plugin.addTodoItem("Test Todo", "Description"); err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}
	items := plugin.getTodoItems()
	id := items[0].ID

	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action":    "updateTodo",
		"id":        id,
		"completed": true,
	}

	result := tool.Call(agentContext, args)
	if !result.Success() {
		t.Errorf("Expected success, got error: %s", result.Error())
	}

	// Verify update
	updated := plugin.getTodoItem(id)
	if !updated.Completed {
		t.Error("Todo should be completed after update")
	}
}

// TestTodoHandlerTool_UpdateTodo_ByTitle tests updateTodo action with title
func TestTodoHandlerTool_UpdateTodo_ByTitle(t *testing.T) {
	plugin := NewTodoPlugin(nil)

	// Add a todo
	if err := plugin.addTodoItem("Test Todo", "Description"); err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}

	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action":    "updateTodo",
		"title":     "Test Todo",
		"completed": true,
	}

	result := tool.Call(agentContext, args)
	if !result.Success() {
		t.Errorf("Expected success, got error: %s", result.Error())
	}

	// Verify update
	updated := plugin.getTodoItemByTitle("Test Todo")
	if !updated.Completed {
		t.Error("Todo should be completed after update")
	}
}

// TestTodoHandlerTool_UpdateTodo_MissingCompleted tests updateTodo without completed parameter
func TestTodoHandlerTool_UpdateTodo_MissingCompleted(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action": "updateTodo",
		"id":     "some-id",
	}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for missing completed parameter")
	}

	if !strings.Contains(result.Error(), "completed") {
		t.Errorf("Error should mention completed parameter, got: %s", result.Error())
	}
}

// TestTodoHandlerTool_UpdateTodo_MissingIDAndTitle tests updateTodo without ID or title
func TestTodoHandlerTool_UpdateTodo_MissingIDAndTitle(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action":    "updateTodo",
		"completed": true,
	}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for missing ID and title")
	}

	if !strings.Contains(result.Error(), "id") || !strings.Contains(result.Error(), "title") {
		t.Errorf("Error should mention id or title, got: %s", result.Error())
	}
}

// TestTodoHandlerTool_UpdateTodo_NonExistentID tests updateTodo with non-existent ID
func TestTodoHandlerTool_UpdateTodo_NonExistentID(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action":    "updateTodo",
		"id":        "non-existent-id",
		"completed": true,
	}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for non-existent ID")
	}

	if !strings.Contains(result.Error(), "failed to update") {
		t.Errorf("Error should mention update failure, got: %s", result.Error())
	}
}

// TestTodoHandlerTool_UpdateTodo_NonExistentTitle tests updateTodo with non-existent title
func TestTodoHandlerTool_UpdateTodo_NonExistentTitle(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action":    "updateTodo",
		"title":     "Non-existent Title",
		"completed": true,
	}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for non-existent title")
	}

	if !strings.Contains(result.Error(), "failed to update") {
		t.Errorf("Error should mention update failure, got: %s", result.Error())
	}
}

// TestTodoHandlerTool_InvalidAction tests invalid action
func TestTodoHandlerTool_InvalidAction(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action": "invalidAction",
	}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for invalid action")
	}

	if !strings.Contains(result.Error(), "unknown action") {
		t.Errorf("Error should mention unknown action, got: %s", result.Error())
	}
}

// TestTodoHandlerTool_MissingAction tests missing action parameter
func TestTodoHandlerTool_MissingAction(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for missing action")
	}

	if !strings.Contains(result.Error(), "action") {
		t.Errorf("Error should mention action parameter, got: %s", result.Error())
	}
}

// TestTodoHandlerTool_InvalidActionType tests invalid action type
func TestTodoHandlerTool_InvalidActionType(t *testing.T) {
	plugin := NewTodoPlugin(nil)
	tools := plugin.Tools()
	tool := tools[0]

	agentContext := map[string]any{}
	args := map[string]any{
		"action": 123, // Not a string
	}

	result := tool.Call(agentContext, args)
	if result.Success() {
		t.Error("Expected error for invalid action type")
	}

	if !strings.Contains(result.Error(), "string") {
		t.Errorf("Error should mention string type, got: %s", result.Error())
	}
}

// TestTodoItem_JSONSerialization tests JSON serialization of TodoItem
func TestTodoItem_JSONSerialization(t *testing.T) {
	item := TodoItem{
		ID:          "test-id",
		Title:       "Test Title",
		Description: "Test Description",
		Completed:   true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	jsonData, err := json.Marshal(item)
	if err != nil {
		t.Errorf("Failed to marshal TodoItem: %v", err)
	}

	var unmarshaled TodoItem
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal TodoItem: %v", err)
	}

	if unmarshaled.ID != item.ID {
		t.Errorf("ID mismatch: expected %s, got %s", item.ID, unmarshaled.ID)
	}

	if unmarshaled.Title != item.Title {
		t.Errorf("Title mismatch: expected %s, got %s", item.Title, unmarshaled.Title)
	}

	if unmarshaled.Description != item.Description {
		t.Errorf("Description mismatch: expected %s, got %s", item.Description, unmarshaled.Description)
	}

	if unmarshaled.Completed != item.Completed {
		t.Errorf("Completed mismatch: expected %v, got %v", item.Completed, unmarshaled.Completed)
	}
}

// TestNewTodoPlugin tests the constructor
func TestNewTodoPlugin(t *testing.T) {
	// Test with callback
	callback := func(todos []*TodoItem) {
		// Callback function for testing
	}

	plugin := NewTodoPlugin(callback)
	if plugin == nil {
		t.Fatal("Expected non-nil plugin")
	}

	if plugin != nil && plugin.onTodoUpdate == nil {
		t.Error("Expected callback to be set")
	}

	// Test with nil callback
	plugin2 := NewTodoPlugin(nil)
	if plugin2 == nil {
		t.Fatal("Expected non-nil plugin")
	}

	if plugin2 != nil && plugin2.onTodoUpdate != nil {
		t.Error("Expected nil callback when passed nil")
	}

	// Verify initial state
	items := plugin2.getTodoItems()
	if len(items) != 0 {
		t.Errorf("Expected empty todo list initially, got %d items", len(items))
	}
}
