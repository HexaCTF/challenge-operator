package controller

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

const (
	kafkaTopic = "challenge-status"
)

// KafkaProducer Kafka producer
type KafkaProducer struct {
	producer sarama.SyncProducer
}

// StatusMessage 는 Kafka에 보낼 메세지
type StatusMessage struct {
	User      string    `json:"user"`
	ProblemID string    `json:"problemId"`
	NewStatus string    `json:"newStatus"`
	Timestamp time.Time `json:"timestamp"`
}

// NewKafkaProducer Kafka producer 객체 생성
// Producer 관련 설정 수행
func NewKafkaProducer(brokers []string) (*KafkaProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	var producer sarama.SyncProducer
	var err error
	maxRetries := 10
	retryInterval := time.Second * 10

	for i := 0; i < maxRetries; i++ {
		producer, err = sarama.NewSyncProducer(brokers, config)
		if err == nil {
			break
		}
		//log.Info("Failed to connect to Kafka, retrying...",
		//	"attempt", i+1,
		//	"maxRetries", maxRetries,
		//	"error", err)
		time.Sleep(retryInterval)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &KafkaProducer{
		producer: producer,
	}, nil
}

// SendStatusChange 상태 메세지를 보낼때 사용된다.
func (k *KafkaProducer) SendStatusChange(user, problemId, newStatus string) error {

	if k == nil {
		return fmt.Errorf("KafkaProducer instance is nil")
	}
	if k.producer == nil {
		return fmt.Errorf("internal Kafka producer is nil")
	}
	msg := StatusMessage{
		User:      user,
		ProblemID: problemId,
		NewStatus: newStatus,
		Timestamp: time.Now(),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal status message: %w", err)
	}

	_, _, err = k.producer.SendMessage(&sarama.ProducerMessage{
		Topic: kafkaTopic,
		Value: sarama.StringEncoder(payload),
		Key:   sarama.StringEncoder(fmt.Sprintf("%s-%s", user, problemId)),
	})

	if err != nil {
		return fmt.Errorf("failed to send Kafka message: %w", err)
	}

	return nil
}

// Close Kafka producer 종료
func (k *KafkaProducer) Close() error {
	if k.producer != nil {
		return k.producer.Close()
	}
	return nil
}
