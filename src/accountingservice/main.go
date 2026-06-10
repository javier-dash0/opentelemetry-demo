// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package main

//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go
//go:generate go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go:generate protoc --go_out=./ --go-grpc_out=./ --proto_path=../../pb ../../pb/demo.proto

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/open-telemetry/opentelemetry-demo/src/accountingservice/kafka"
)

var log *logrus.Logger
var resource *sdkresource.Resource
var initResourcesOnce sync.Once

func init() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout
}

func initResource() *sdkresource.Resource {
	initResourcesOnce.Do(func() {
		extraResources, _ := sdkresource.New(
			context.Background(),
			sdkresource.WithOS(),
			sdkresource.WithProcess(),
			sdkresource.WithContainer(),
			sdkresource.WithHost(),
		)
		resource, _ = sdkresource.Merge(
			sdkresource.Default(),
			extraResources,
		)
	})
	return resource
}

func initLogProvider() *sdklog.LoggerProvider {
	ctx := context.Background()

	logExporter, err := otlploggrpc.New(ctx)
	if err != nil {
		log.Fatalf("new otlp log grpc exporter failed: %v", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(initResource()),
	)
	return lp
}

func initTracerProvider() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(initResource()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

func main() {
	lp := initLogProvider()
	defer func() {
		if err := lp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Error shutting down log provider: %v", err)
		}
		log.Println("Shutdown log provider")
	}()

	// Bridge logrus → OTel LoggerProvider so all log calls reach the collector.
	log.AddHook(otellogrus.NewHook(
		"accountingservice",
		otellogrus.WithLevels([]logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
			logrus.InfoLevel,
		}),
		otellogrus.WithLoggerProvider(lp),
	))

	tp, err := initTracerProvider()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
		log.Println("Shutdown trace provider")
	}()

	var brokers string
	mustMapEnv(&brokers, "KAFKA_SERVICE_ADDR")

	brokerList := strings.Split(brokers, ",")
	log.WithFields(logrus.Fields{
		"brokers": strings.Join(brokerList, ", "),
	}).Info("Connecting to Kafka brokers")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()
	var consumerGroup sarama.ConsumerGroup
	if consumerGroup, err = kafka.StartConsumerGroup(ctx, brokerList, log); err != nil {
		log.WithError(err).Fatal("Failed to start Kafka consumer group")
	}
	log.WithFields(logrus.Fields{
		"topic":   kafka.Topic,
		"groupID": kafka.GroupID,
	}).Info("Kafka consumer group started")

	defer func() {
		if err := consumerGroup.Close(); err != nil {
			log.WithError(err).Error("Error closing consumer group")
		} else {
			log.Info("Kafka consumer group closed")
		}
	}()

	<-ctx.Done()

	log.Info("Accounting service shutting down")
}

func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		log.WithField("envKey", envKey).Fatal("Required environment variable not set")
	}
	*target = v
}
