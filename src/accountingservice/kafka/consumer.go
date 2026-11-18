// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package kafka

import (
	"context"
	pb "github.com/open-telemetry/opentelemetry-demo/src/accountingservice/genproto/oteldemo"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
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
		log: log,
	}

	if err := consumeSessions(ctx, consumerGroup, &handler, Topic); err != nil {
		return nil, err
	}

	return consumerGroup, nil
}

// consumeSessions joins the consumer group for as long as the context is live.
// ConsumerGroup.Consume returns every time a session ends, for example after a
// rebalance, so it must be called again to get the new claims.
func consumeSessions(ctx context.Context, group sarama.ConsumerGroup, handler sarama.ConsumerGroupHandler, topics ...string) error {
	for ctx.Err() == nil {
		if err := group.Consume(ctx, topics, handler); err != nil {
			return err
		}
	}

	return nil
}

type groupHandler struct {
	log *logrus.Logger
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
			orderResult := pb.OrderResult{}
			err := proto.Unmarshal(message.Value, &orderResult)
			if err != nil {
				return err
			}

			g.log.WithFields(logrus.Fields{
				"orderId":          orderResult.OrderId,
				"messageTimestamp": message.Timestamp,
				"messageTopic":     message.Topic,
			}).Info("Message claimed")
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}
