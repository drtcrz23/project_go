package subpub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSubPub(t *testing.T) {
	sp := NewSubPub()
	ctx := context.Background()

	t.Run("Subscribe and Publish", func(t *testing.T) {
		var received []interface{}
		var mu sync.Mutex

		handler := func(msg interface{}) {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
		}

		sub, err := sp.Subscribe("test", handler)
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}

		// Publish messages
		messages := []interface{}{"msg1", 42, "msg3"}
		for _, msg := range messages {
			if err := sp.Publish("test", msg); err != nil {
				t.Fatalf("Publish failed: %v", err)
			}
		}

		// Wait for messages to be processed
		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		if len(received) != len(messages) {
			t.Errorf("Expected %d messages, got %d", len(messages), len(received))
		}
		for i, msg := range messages {
			if received[i] != msg {
				t.Errorf("Expected message %v, got %v", msg, received[i])
			}
		}
		mu.Unlock()

		// Unsubscribe
		sub.Unsubscribe()
	})

	t.Run("Close with Context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()

		if err := sp.Close(ctx); err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		// Try subscribing after close
		_, err := sp.Subscribe("test", func(msg interface{}) {})
		if err == nil {
			t.Error("Expected error when subscribing after close")
		}
	})

	t.Run("Close with Cancelled Context", func(t *testing.T) {
		sp := NewSubPub()

		var received []interface{}
		var mu sync.Mutex

		handler := func(msg interface{}) {
			time.Sleep(50 * time.Millisecond) // Simulation
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
		}

		_, err := sp.Subscribe("test", handler)
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}

		messages := []interface{}{"msg1", "msg2", "msg3"}
		for _, msg := range messages {
			if err := sp.Publish("test", msg); err != nil {
				t.Fatalf("Publish failed: %v", err)
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		start := time.Now()
		if err := sp.Close(ctx); err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
		if duration := time.Since(start); duration > 10*time.Millisecond {
			t.Errorf("Close took too long (%v), expected immediate exit", duration)
		}

		time.Sleep(200 * time.Millisecond)
		mu.Lock()
		if len(received) != len(messages) {
			t.Errorf("Expected %d messages to be processed, got %d", len(messages), len(received))
		}
		mu.Unlock()
	})
}
