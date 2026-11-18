// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
)

// fakeConsumerGroup counts the sessions it joins and ends a session on every
// call, the same way sarama returns from Consume when a session ends.
type fakeConsumerGroup struct {
	calls      int
	err        error
	cancelOn   int
	cancel     context.CancelFunc
	lastTopics []string
}

func (f *fakeConsumerGroup) Consume(_ context.Context, topics []string, _ sarama.ConsumerGroupHandler) error {
	f.calls++
	f.lastTopics = topics
	if f.err != nil {
		return f.err
	}
	if f.calls == f.cancelOn {
		f.cancel()
	}
	return nil
}

func (f *fakeConsumerGroup) Errors() <-chan error      { return nil }
func (f *fakeConsumerGroup) Close() error              { return nil }
func (f *fakeConsumerGroup) Pause(map[string][]int32)  {}
func (f *fakeConsumerGroup) Resume(map[string][]int32) {}
func (f *fakeConsumerGroup) PauseAll()                 {}
func (f *fakeConsumerGroup) ResumeAll()                {}

func TestConsumeSessionsRejoinsAfterEverySession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group := &fakeConsumerGroup{cancel: cancel, cancelOn: 3}

	if err := consumeSessions(ctx, group, &groupHandler{log: logrus.New()}, Topic); err != nil {
		t.Fatalf("consumeSessions returned an error: %v", err)
	}

	if group.calls != 3 {
		t.Fatalf("Consume was called %d times, want 3 sessions", group.calls)
	}
	if len(group.lastTopics) != 1 || group.lastTopics[0] != Topic {
		t.Fatalf("Consume subscribed to %v, want [%q]", group.lastTopics, Topic)
	}
}

func TestConsumeSessionsReturnsConsumerError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumeErr := errors.New("broker is gone")
	group := &fakeConsumerGroup{err: consumeErr}

	err := consumeSessions(ctx, group, &groupHandler{log: logrus.New()}, Topic)
	if !errors.Is(err, consumeErr) {
		t.Fatalf("consumeSessions returned %v, want %v", err, consumeErr)
	}
	if group.calls != 1 {
		t.Fatalf("Consume was called %d times, want 1", group.calls)
	}
}

func TestConsumeSessionsDoesNotConsumeAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	group := &fakeConsumerGroup{}

	if err := consumeSessions(ctx, group, &groupHandler{log: logrus.New()}, Topic); err != nil {
		t.Fatalf("consumeSessions returned an error: %v", err)
	}
	if group.calls != 0 {
		t.Fatalf("Consume was called %d times, want 0", group.calls)
	}
}
