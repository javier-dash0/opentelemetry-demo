// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package kafka

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/IBM/sarama"
)

type OTelInterceptor struct {
	tracer     trace.Tracer
	fixedAttrs []attribute.KeyValue
}

// NewOTelInterceptor processes span for intercepted messages and add some
// headers with the span data.
func NewOTelInterceptor(groupID string) *OTelInterceptor {
	oi := OTelInterceptor{}
	oi.tracer = otel.Tracer("github.com/open-telemetry/opentelemetry-demo/accountingservice/sarama")

	oi.fixedAttrs = []attribute.KeyValue{
		semconv.MessagingSystemKafka,
		semconv.MessagingOperationReceive,
		semconv.MessagingKafkaConsumerGroup(groupID),
		semconv.NetworkTransportTCP,
	}
	return &oi
}

// extractTraceContext extracts the W3C trace context propagated by the producer
// from the Kafka message headers. This is called by ConsumeClaim to parent the
// process span under the original producer span.
func extractTraceContext(msg *sarama.ConsumerMessage) context.Context {
	headers := propagation.MapCarrier{}
	for _, recordHeader := range msg.Headers {
		headers[string(recordHeader.Key)] = string(recordHeader.Value)
	}
	return otel.GetTextMapPropagator().Extract(context.Background(), headers)
}

func (oi *OTelInterceptor) OnConsume(msg *sarama.ConsumerMessage) {
	ctx := extractTraceContext(msg)

	_, span := oi.tracer.Start(
		ctx,
		fmt.Sprintf("%s receive", msg.Topic),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(oi.fixedAttrs...),
		trace.WithAttributes(
			semconv.MessagingDestinationName(msg.Topic),
			semconv.MessagingKafkaMessageOffset(int(msg.Offset)),
			semconv.MessagingMessageBodySize(len(msg.Value)),
			semconv.MessagingOperationReceive,
			semconv.MessagingKafkaDestinationPartition(int(msg.Partition)),
		),
	)
	defer span.End()
}
