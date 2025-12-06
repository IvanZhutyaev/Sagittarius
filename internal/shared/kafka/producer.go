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

type Producer struct {
	writer *kafka.Writer
	tracer trace.Tracer
}

func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
	}

	return &Producer{
		writer: writer,
		tracer: otel.Tracer("kafka-producer"),
	}
}

func (p *Producer) Publish(ctx context.Context, key string, event interface{}) error {
	ctx, span := p.tracer.Start(ctx, "kafka.Publish")
	defer span.End()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	message := kafka.Message{
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{Key: "Content-Type", Value: []byte("application/json")},
		},
	}

	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

