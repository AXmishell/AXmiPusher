package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaQueue 基于 Kafka 的队列(生产模式)。
// topic 结构: <topic>        正常消息
//           <topic>-retry   重试消息(延迟消费)
//           <topic>-dead    死信消息
type KafkaQueue struct {
	name        string
	topic       string
	groupID     string
	brokers     []string
	concurrency int

	writer *kafka.Writer
	reader *kafka.Reader
}

// NewKafkaQueue 创建 Kafka 队列。
func NewKafkaQueue(brokers []string, topic, groupID string, concurrency int) (*KafkaQueue, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("Kafka brokers 为空")
	}
	if concurrency <= 0 {
		concurrency = 4
	}
	q := &KafkaQueue{
		name:        "kafka",
		topic:       topic,
		groupID:     groupID,
		brokers:     brokers,
		concurrency: concurrency,
	}
	q.writer = &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		MaxAttempts:  3,
	}
	return q, nil
}

// Publish 发布消息到 Kafka。
func (q *KafkaQueue) Publish(ctx context.Context, msg *TaskMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	key := []byte(fmt.Sprintf("%d-%d", msg.TenantID, msg.MessageID))
	return q.writer.WriteMessages(ctx, kafka.Message{Key: key, Value: data})
}

// Subscribe 消费消息。失败的消息进入 retry topic, 重试仍失败进 dead topic。
func (q *KafkaQueue) Subscribe(ctx context.Context, handler Handler) error {
	for i := 0; i < q.concurrency; i++ {
		go q.consumer(ctx, handler, i)
	}
	<-ctx.Done()
	q.Close()
	return nil
}

func (q *KafkaQueue) consumer(ctx context.Context, handler Handler, id int) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     q.brokers,
		GroupID:     q.groupID,
		Topic:       q.topic,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     500 * time.Millisecond,
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	for {
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		var msg TaskMessage
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			reader.CommitMessages(ctx, m)
			continue
		}
		if err := handler(ctx, &msg); err != nil {
			// 重试策略: 重试次数 < 3 进 retry topic, 否则进 dead topic。
			if msg.RetryCount < 3 {
				msg.RetryCount++
				if werr := q.writeRetry(ctx, &msg); werr == nil {
					reader.CommitMessages(ctx, m)
					continue
				}
			}
			if werr := q.writeDead(ctx, &msg); werr != nil {
				continue
			}
		}
		reader.CommitMessages(ctx, m)
	}
}

// writeRetry 写入重试 topic。
func (q *KafkaQueue) writeRetry(ctx context.Context, msg *TaskMessage) error {
	data, _ := json.Marshal(msg)
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: q.brokers,
		Topic:   q.topic + "-retry",
	})
	defer w.Close()
	return w.WriteMessages(ctx, kafka.Message{Key: []byte(fmt.Sprintf("%d", msg.MessageID)), Value: data})
}

// writeDead 写入死信 topic。
func (q *KafkaQueue) writeDead(ctx context.Context, msg *TaskMessage) error {
	data, _ := json.Marshal(msg)
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: q.brokers,
		Topic:   q.topic + "-dead",
	})
	defer w.Close()
	return w.WriteMessages(ctx, kafka.Message{Key: []byte(fmt.Sprintf("%d", msg.MessageID)), Value: data})
}

// Close 关闭队列。
func (q *KafkaQueue) Close() error {
	if q.writer != nil {
		return q.writer.Close()
	}
	return nil
}

// Type 返回队列类型。
func (q *KafkaQueue) Type() string { return q.name }
