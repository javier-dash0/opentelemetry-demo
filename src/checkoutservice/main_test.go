// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama"

	pb "github.com/open-telemetry/opentelemetry-demo/src/checkoutservice/genproto/oteldemo"
)

type fakeAsyncProducer struct {
	sarama.AsyncProducer
	input     chan *sarama.ProducerMessage
	successes chan *sarama.ProducerMessage
	errors    chan *sarama.ProducerError
}

func newFakeAsyncProducer() *fakeAsyncProducer {
	return &fakeAsyncProducer{
		input:     make(chan *sarama.ProducerMessage, 1),
		successes: make(chan *sarama.ProducerMessage, 1),
		errors:    make(chan *sarama.ProducerError, 1),
	}
}

func (f *fakeAsyncProducer) Input() chan<- *sarama.ProducerMessage     { return f.input }
func (f *fakeAsyncProducer) Successes() <-chan *sarama.ProducerMessage { return f.successes }
func (f *fakeAsyncProducer) Errors() <-chan *sarama.ProducerError      { return f.errors }
func (f *fakeAsyncProducer) AsyncClose()                               {}
func (f *fakeAsyncProducer) Close() error                              { return nil }

// A failed produce is delivered on Errors(), not Successes(). sendToPostProcessor
// must return in that case so PlaceOrder can answer.
func TestSendToPostProcessorReturnsOnProduceFailure(t *testing.T) {
	f := newFakeAsyncProducer()
	cs := &checkoutService{KafkaProducerClient: f}

	go func() {
		<-f.input
		f.errors <- &sarama.ProducerError{Err: errors.New("broker unavailable")}
	}()

	done := make(chan struct{})
	go func() {
		cs.sendToPostProcessor(context.Background(), &pb.OrderResult{})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("sendToPostProcessor blocked forever after a failed produce")
	}
}

func TestSendToPostProcessorReturnsOnProduceSuccess(t *testing.T) {
	f := newFakeAsyncProducer()
	cs := &checkoutService{KafkaProducerClient: f}

	go func() {
		<-f.input
		f.successes <- &sarama.ProducerMessage{Offset: 7}
	}()

	done := make(chan struct{})
	go func() {
		cs.sendToPostProcessor(context.Background(), &pb.OrderResult{})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("sendToPostProcessor blocked forever after a successful produce")
	}
}
