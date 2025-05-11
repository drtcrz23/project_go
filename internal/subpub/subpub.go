package subpub

import (
	"context"
	"errors"
	"sync"
)

// MessageHandler is a callback function that processes messages delivered to subscribers.
type MessageHandler func(msg interface{})

// Subscription interface for unsubscribing from a topic.
type Subscription interface {
	Unsubscribe()
}

// SubPub interface for the event bus.
type SubPub interface {
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	Publish(subject string, msg interface{}) error
	Close(ctx context.Context) error
}

// subPub is the main event bus implementation.
type subPub struct {
	mu        sync.RWMutex
	closed    bool
	subs      map[string][]*subscriber
	closeChan chan struct{}
}

// subscriber represents a single subscription.
type subscriber struct {
	cb        MessageHandler
	msgChan   chan interface{}
	closeChan chan struct{}
}

// subscription handles unsubscription.
type subscription struct {
	sp      *subPub
	subject string
	sub     *subscriber
}

// NewSubPub creates a new SubPub instance.
func NewSubPub() SubPub {
	return &subPub{
		subs:      make(map[string][]*subscriber),
		closeChan: make(chan struct{}),
	}
}

// Subscribe creates an asynchronous subscriber for the specified subject.
func (sp *subPub) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.closed {
		return nil, errors.New("subpub is closed")
	}

	sub := &subscriber{
		cb:        cb,
		msgChan:   make(chan interface{}, 100),
		closeChan: make(chan struct{}),
	}

	// Start a goroutine to handle messages for this subscriber
	go sub.processMessages()

	sp.subs[subject] = append(sp.subs[subject], sub)

	return &subscription{
		sp:      sp,
		subject: subject,
		sub:     sub,
	}, nil
}

// Publish publishes a message to the specified subject.
func (sp *subPub) Publish(subject string, msg interface{}) error {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if sp.closed {
		return errors.New("subpub is closed")
	}

	subs, exists := sp.subs[subject]
	if !exists {
		return nil
	}

	for _, sub := range subs {
		select {
		case sub.msgChan <- msg:
		case <-sub.closeChan:
			// Subscriber is closed, skip
		case <-sp.closeChan:
			return errors.New("subpub is closed")
		}
	}

	return nil
}

// Close shuts down the pub-sub system.
func (sp *subPub) Close(ctx context.Context) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.closed {
		return nil
	}

	sp.closed = true

	if ctx.Err() != nil {
		return ctx.Err()
	}

	for _, subs := range sp.subs {
		for _, sub := range subs {
			close(sub.closeChan)
		}
	}
	sp.subs = make(map[string][]*subscriber)
	return nil
}

// Unsubscribe cancels the subscription.
func (s *subscription) Unsubscribe() {
	s.sp.mu.Lock()
	defer s.sp.mu.Unlock()

	if s.sp.closed {
		return
	}

	// Find and remove the subscriber
	for i, sub := range s.sp.subs[s.subject] {
		if sub == s.sub {
			s.sp.subs[s.subject] = append(s.sp.subs[s.subject][:i], s.sp.subs[s.subject][i+1:]...)
			close(s.sub.closeChan)
			break
		}
	}

	// Clean up empty subject
	if len(s.sp.subs[s.subject]) == 0 {
		delete(s.sp.subs, s.subject)
	}
}

// processMessages handles incoming messages for a subscriber.
func (s *subscriber) processMessages() {
	for {
		select {
		case msg, ok := <-s.msgChan:
			if !ok {
				return
			}
			s.cb(msg)
		case <-s.closeChan:
			for len(s.msgChan) > 0 {
				msg := <-s.msgChan
				s.cb(msg)
			}
			return
		}
	}
}
