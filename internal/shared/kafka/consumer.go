package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type MessageHandler func(ctx context.Context, key string, value []byte) error

type Consumer struct {
	reader *kafka.Reader
	tracer trace.Tracer
}

func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	return &Consumer{
		reader: reader,
		tracer: otel.Tracer("kafka-consumer"),
	}
}

func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch message: %w", err)
			}

			ctx, span := c.tracer.Start(ctx, "kafka.Consume")
			span.SetAttributes(
				trace.StringAttribute("topic", msg.Topic),
				trace.StringAttribute("partition", fmt.Sprintf("%d", msg.Partition)),
			)

			if err := handler(ctx, string(msg.Key), msg.Value); err != nil {
				span.RecordError(err)
				// In production, send to DLQ
				span.End()
				continue
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				span.RecordError(err)
				span.End()
				return fmt.Errorf("failed to commit message: %w", err)
			}

			span.End()
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func UnmarshalEvent(data []byte, event interface{}) error {
	return json.Unmarshal(data, event)
}

