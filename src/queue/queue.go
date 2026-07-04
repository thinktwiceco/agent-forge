package queue

// Queue is a buffered FIFO channel of Messages. It is safe for concurrent use by multiple
// producers (Enqueue) and is intended to be consumed by a single agent drain loop.
//
// Example:
//
//	q := queue.New(32)
//	q.Enqueue("Ciao!", "conv-abc", map[string]string{"sender": "user"})
//	q.Enqueue("Hello!", "conv-xyz", map[string]string{"sender": "agent-b"})
//	// agent.Drain(ctx, q) processes them sequentially
type Queue struct {
	ch chan Message
}

// New creates a Queue with the given buffer size.
// bufSize controls how many messages can be pending before Enqueue blocks.
func New(bufSize int) *Queue {
	return &Queue{ch: make(chan Message, bufSize)}
}

// Enqueue adds a message to the queue.
//
// chatId routes the message to a specific conversation on the agent;
// pass an empty string to start a new conversation.
//
// headers is a map of arbitrary metadata. A "timestamp" header (RFC3339 UTC)
// is auto-injected unless already present in the provided map.
//
// Enqueue blocks if the queue buffer is full.
func (q *Queue) Enqueue(body, chatId string, headers map[string]string) {
	// Copy headers so the caller's map is not mutated.
	// Timestamp injection happens in FormatHeaders at drain time.
	h := make(map[string]string, len(headers))
	for k, v := range headers {
		h[k] = v
	}
	q.ch <- Message{Headers: h, Body: body, ChatId: chatId}
}

// C returns the receive-only channel for reading messages.
// Intended for use by the agent drain loop.
func (q *Queue) C() <-chan Message {
	return q.ch
}

// Len returns the number of messages currently buffered in the queue.
// It is safe to call concurrently and does not block.
func (q *Queue) Len() int {
	return len(q.ch)
}

// Close closes the underlying channel, signalling the drain loop to stop
// after processing remaining messages.
// Close must be called exactly once.
func (q *Queue) Close() {
	close(q.ch)
}

var _ Inbox = (*Queue)(nil)
