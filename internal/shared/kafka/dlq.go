package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type DLQProducer struct {
	writer *kafka.Writer
}

func NewDLQProducer(brokers []string) *DLQProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        "dead-letter-queue",
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
	}

	return &DLQProducer{writer: writer}
}

type DLQMessage struct {
	OriginalTopic string          `json:"original_topic"`
	OriginalKey    string          `json:"original_key"`
	OriginalValue  json.RawMessage `json:"original_value"`
	Error          string          `json:"error"`
	Timestamp      time.Time       `json:"timestamp"`
	RetryCount     int             `json:"retry_count"`
}

func (p *DLQProducer) SendToDLQ(ctx context.Context, originalTopic, originalKey string, originalValue []byte, err error, retryCount int) error {
	dlqMsg := DLQMessage{
		OriginalTopic: originalTopic,
		OriginalKey:   originalKey,
		OriginalValue: originalValue,
		Error:         err.Error(),
		Timestamp:     time.Now(),
		RetryCount:    retryCount,
	}

	data, err := json.Marshal(dlqMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	message := kafka.Message{
		Key:   []byte(fmt.Sprintf("%s:%s", originalTopic, originalKey)),
		Value: data,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, message)
}

func (p *DLQProducer) Close() error {
	return p.writer.Close()
}

