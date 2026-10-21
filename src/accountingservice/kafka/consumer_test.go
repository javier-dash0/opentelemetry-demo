// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package kafka

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
)

// fakeConsumerGroup records how often Consume is called. The first call returns
// immediately to simulate the end of a consumer session (a server-side
// rebalance or a session timeout). Later calls block until the context is
// cancelled, like a healthy session that stays joined.
type fakeConsumerGroup struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *fakeConsumerGroup) Consume(ctx context.Context, _ []string, _ sarama.ConsumerGroupHandler) error {
	f.mu.Lock()
	f.calls++
	n := f.calls
	err := f.err
	f.mu.Unlock()

	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	<-ctx.Done()
	return nil
}

func (f *fakeConsumerGroup) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeConsumerGroup) Errors() <-chan error      { return nil }
func (f *fakeConsumerGroup) Close() error              { return nil }
func (f *fakeConsumerGroup) Pause(map[string][]int32)  {}
func (f *fakeConsumerGroup) Resume(map[string][]int32) {}
func (f *fakeConsumerGroup) PauseAll()                 {}
func (f *fakeConsumerGroup) ResumeAll()                {}

func TestConsumeRejoinsGroupAfterSessionEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fake := &fakeConsumerGroup{}

	done := make(chan error, 1)
	go func() { done <- consume(ctx, fake, &groupHandler{}) }()

	// The loop must call Consume again after the first session ends.
	deadline := time.After(2 * time.Second)
	for fake.callCount() < 2 {
		select {
		case <-deadline:
			t.Fatalf("Consume called %d time(s); the consumer did not rejoin the group after the session ended", fake.callCount())
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("consume returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("consume did not return after the context was cancelled")
	}
}

func TestConsumeReturnsError(t *testing.T) {
	wantErr := errors.New("broker unavailable")
	fake := &fakeConsumerGroup{err: wantErr}

	if err := consume(context.Background(), fake, &groupHandler{}); !errors.Is(err, wantErr) {
		t.Fatalf("consume returned %v, want %v", err, wantErr)
	}
}
