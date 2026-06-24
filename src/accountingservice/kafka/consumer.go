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
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

var (
	Topic           = "orders"
	ProtocolVersion = sarama.V3_0_0_0
	GroupID         = "accountingservice"
)

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
		tracer: otel.Tracer("github.com/open-telemetry/opentelemetry-demo/accountingservice/consumer"),
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
			// Extract propagated trace context from Kafka message headers.
			headers := propagation.MapCarrier{}
			for _, h := range message.Headers {
				headers[string(h.Key)] = string(h.Value)
			}
			parentCtx := otel.GetTextMapPropagator().Extract(context.Background(), headers)

			ctx, span := g.tracer.Start(
				parentCtx,
				fmt.Sprintf("%s process", message.Topic),
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
					semconv.MessagingSystemKafka,
					semconv.MessagingOperationProcess,
					semconv.MessagingKafkaConsumerGroup(GroupID),
					semconv.MessagingDestinationName(message.Topic),
					semconv.MessagingKafkaMessageOffset(int(message.Offset)),
					semconv.MessagingMessageBodySize(len(message.Value)),
					semconv.MessagingKafkaDestinationPartition(int(message.Partition)),
				),
			)

			orderResult := pb.OrderResult{}
			if err := proto.Unmarshal(message.Value, &orderResult); err != nil {
				span.RecordError(err)
				span.SetStatus(otelcodes.Error, err.Error())
				span.End()
				return err
			}

			span.SetAttributes(attribute.String("app.order.id", orderResult.OrderId))

			g.log.WithContext(ctx).WithFields(logrus.Fields{
				"orderId":          orderResult.OrderId,
				"messageTimestamp": message.Timestamp,
				"messageTopic":     message.Topic,
			}).Info("Message claimed")

			session.MarkMessage(message, "")
			span.End()

		case <-session.Context().Done():
			return nil
		}
	}
}
