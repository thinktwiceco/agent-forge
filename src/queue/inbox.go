package queue

// Inbox accepts autonomous messages for the agent turn queue.
// Plugins enqueue formatted turns; the agent dispatches them through a single scheduler.
type Inbox interface {
	Enqueue(body, chatId string, headers map[string]string)
	Len() int
}
