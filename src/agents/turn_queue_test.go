package agents

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/thinktwiceco/agent-forge/src/core"
)

func TestTurnQueue_SameChatID_Serializes(t *testing.T) {
	var mu sync.Mutex
	order := make([]int, 0, 2)
	done := make(chan struct{}, 2)

	tq := NewTurnQueue(8, func(_ context.Context, req turnRequest) {
		var n int
		switch req.Body {
		case "first":
			n = 1
		case "second":
			n = 2
		}
		mu.Lock()
		order = append(order, n)
		mu.Unlock()
		done <- struct{}{}
	})
	tq.Start()
	defer tq.Stop()

	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "first", ChatID: "conv-a"})
	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "second", ChatID: "conv-a"})

	<-done
	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("expected [1 2], got %v", order)
	}
}

func TestTurnQueue_DifferentChatIDs_MayOverlap(t *testing.T) {
	start := make(chan string, 2)
	release := make(chan struct{})

	tq := NewTurnQueue(8, func(_ context.Context, req turnRequest) {
		start <- req.ChatID
		<-release
	})
	tq.Start()
	defer tq.Stop()

	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "a", ChatID: "conv-1"})
	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "b", ChatID: "conv-2"})

	first := <-start
	second := <-start
	if first == second {
		t.Fatalf("expected different chats to start, got %q twice", first)
	}
	close(release)
	time.Sleep(50 * time.Millisecond)
}

func TestTurnQueue_CancelWhileQueued(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{}, 1)

	tq := NewTurnQueue(8, func(_ context.Context, req turnRequest) {
		if req.Body == "block" {
			started <- struct{}{}
			<-block
		}
	})
	tq.Start()
	defer tq.Stop()

	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "block", ChatID: "conv-b"})
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	responseCh := core.NewResponseCh("test", "", "conv-b", nil)
	_ = tq.Submit(turnRequest{Ctx: ctx, Body: "cancel-me", ChatID: "conv-b", ResponseCh: responseCh})
	cancel()

	select {
	case err := <-responseCh.Error:
		if err == nil {
			t.Fatal("expected cancel error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cancelled turn")
	}

	close(block)
}

func TestTurnQueue_EnqueueAssignsChatID(t *testing.T) {
	got := make(chan string, 1)
	tq := NewTurnQueue(4, func(_ context.Context, req turnRequest) {
		got <- req.ChatID
	})
	tq.Start()
	defer tq.Stop()

	tq.Enqueue("hello", "", map[string]string{"sender": "scheduler"})

	select {
	case chatID := <-got:
		if chatID == "" {
			t.Fatal("expected non-empty chatId from Enqueue")
		}
	case <-time.After(time.Second):
		t.Fatal("turn was not dispatched")
	}
}

func TestTurnQueue_ThreeDeepSameChatID_FIFO(t *testing.T) {
	block := make(chan struct{})
	var mu sync.Mutex
	order := make([]string, 0, 3)
	done := make(chan struct{}, 3)

	tq := NewTurnQueue(8, func(_ context.Context, req turnRequest) {
		mu.Lock()
		order = append(order, req.Body)
		mu.Unlock()
		done <- struct{}{}
		if req.Body == "first" {
			<-block
		}
	})
	tq.Start()
	defer tq.Stop()

	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "first", ChatID: "conv-a"})
	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "second", ChatID: "conv-a"})
	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "third", ChatID: "conv-a"})

	<-done
	close(block)
	<-done
	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 3 || order[0] != "first" || order[1] != "second" || order[2] != "third" {
		t.Fatalf("expected [first second third], got %v", order)
	}
}

func TestTurnQueue_CrossSourceSameChatID(t *testing.T) {
	block := make(chan struct{})
	var mu sync.Mutex
	order := make([]string, 0, 2)
	done := make(chan struct{}, 2)

	tq := NewTurnQueue(8, func(_ context.Context, req turnRequest) {
		mu.Lock()
		order = append(order, req.Source)
		mu.Unlock()
		done <- struct{}{}
		if req.Source == "direct" {
			<-block
		}
	})
	tq.Start()
	defer tq.Stop()

	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "user", ChatID: "conv-x", Source: "direct"})
	tq.Enqueue("reminder", "conv-x", map[string]string{"sender": "scheduler"})

	<-done
	close(block)
	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "direct" || order[1] != "scheduler" {
		t.Fatalf("expected [direct scheduler], got %v", order)
	}
}

func TestTurnQueue_SubmitBlocksWhenFull(t *testing.T) {
	tq := NewTurnQueue(1, func(_ context.Context, _ turnRequest) {})
	// Dispatcher not started: one slot in ingress buffer.

	if err := tq.Submit(turnRequest{Ctx: context.Background(), Body: "one", ChatID: "c1"}); err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if tq.Len() != 1 {
		t.Fatalf("expected len 1, got %d", tq.Len())
	}

	unblocked := make(chan struct{})
	go func() {
		_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "two", ChatID: "c2"})
		close(unblocked)
	}()

	select {
	case <-unblocked:
		t.Fatal("second submit should block while ingress is full and dispatcher stopped")
	case <-time.After(100 * time.Millisecond):
	}

	tq.Start()
	defer tq.Stop()
	<-unblocked
}

func TestTurnQueue_Len(t *testing.T) {
	block := make(chan struct{})
	tq := NewTurnQueue(8, func(_ context.Context, _ turnRequest) {
		<-block
	})
	// do not start dispatcher yet

	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "one", ChatID: "c"})
	_ = tq.Submit(turnRequest{Ctx: context.Background(), Body: "two", ChatID: "c"})
	if tq.Len() != 2 {
		t.Fatalf("expected len 2, got %d", tq.Len())
	}
}
