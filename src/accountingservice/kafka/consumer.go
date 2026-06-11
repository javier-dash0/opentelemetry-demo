// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package kafka

import (
	"context"
	"fmt"

	pb "github.com/open-telemetry/opentelemetry-demo/src/accountingservice/genproto/oteldemo"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

var (
	Topic           = "orders"
	ProtocolVersion = sarama.V3_0_0_0
	GroupID         = "accountingservice"
)

// ordersProcessedCounter counts orders successfully processed by the accounting service.
var ordersProcessedCounter metric.Int64Counter

// ordersFailedCounter counts orders that failed to process (unmarshal or other errors).
var ordersFailedCounter metric.Int64Counter

func init() {
	meter := otel.Meter("accountingservice")
	var err error
	ordersProcessedCounter, err = meter.Int64Counter(
		"app.accounting.orders_processed",
		metric.WithDescription("Total number of orders successfully processed by the accounting service"),
		metric.WithUnit("{order}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create orders_processed counter: %v", err))
	}
	ordersFailedCounter, err = meter.Int64Counter(
		"app.accounting.orders_failed",
		metric.WithDescription("Total number of orders that failed to process in the accounting service"),
		metric.WithUnit("{order}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create orders_failed counter: %v", err))
	}
}

func StartConsumerGroup(ctx context.Context, brokers []string, log *logrus.Logger) (sarama.ConsumerGroup, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = ProtocolVersion
	// So we can know the partition and offset of messages.
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Consumer.Interceptors = []sarama.ConsumerInterceptor{NewOTelInterceptor(GroupID)}

	consumerGroup, err := sarama.NewConsumerGroup(brokers, GroupID, saramaConfig)
	if err != nil {
		return nil, err
	}

	handler := groupHandler{
		log:    log,
		tracer: otel.Tracer("accountingservice"),
	}

	err = consumerGroup.Consume(ctx, []string{Topic}, &handler)
	if err != nil {
		return nil, err
	}

	return consumerGroup, nil
}

type groupHandler struct {
	log    *logrus.Logger
	tracer trace.Tracer
}

func (g *groupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (g *groupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (g *groupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			// Start a child span to cover the actual message-processing work.
			// The parent context is extracted from the Kafka headers by the OTelInterceptor
			// in OnConsume; we re-extract here so the process span is a child of the
			// producer span, maintaining full trace continuity.
			ctx := extractTraceContext(message)
			ctx, span := g.tracer.Start(
				ctx,
				fmt.Sprintf("%s process", message.Topic),
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
					semconv.MessagingSystemKafka,
					semconv.MessagingOperationProcess,
					semconv.MessagingDestinationName(message.Topic),
					semconv.MessagingKafkaConsumerGroup(GroupID),
					semconv.MessagingKafkaMessageOffset(int(message.Offset)),
					semconv.MessagingKafkaDestinationPartition(int(message.Partition)),
					semconv.MessagingMessageBodySize(len(message.Value)),
				),
			)

			orderResult := pb.OrderResult{}
			if err := proto.Unmarshal(message.Value, &orderResult); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to unmarshal order protobuf")
				ordersFailedCounter.Add(ctx, 1,
					metric.WithAttributes(attribute.String("app.accounting.failure_reason", "unmarshal_error")),
				)
				g.log.WithContext(ctx).WithError(err).Error("Failed to unmarshal order message")
				span.End()
				return err
			}

			span.SetAttributes(
				attribute.String("app.order.id", orderResult.OrderId),
				attribute.Int("app.order.items.count", len(orderResult.Items)),
			)

			g.log.WithContext(ctx).WithFields(logrus.Fields{
				"orderId":          orderResult.OrderId,
				"itemCount":        len(orderResult.Items),
				"messageTimestamp": message.Timestamp,
				"messageTopic":     message.Topic,
				"partition":        message.Partition,
				"offset":           message.Offset,
			}).Info("Order message claimed and processed")

			ordersProcessedCounter.Add(ctx, 1)
			session.MarkMessage(message, "")
			span.End()

		case <-session.Context().Done():
			return nil
		}
	}
}
