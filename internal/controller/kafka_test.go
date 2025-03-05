package controller

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/stretchr/testify/assert"
)

func TestKafkaProducer_SendStatusChange(t *testing.T) {
	tests := []struct {
		name      string
		user      string
		problemId string
		newStatus string
		wantErr   bool
	}{
		{
			name:      "successful message send",
			user:      "testuser",
			problemId: "problem1",
			newStatus: "Created",
			wantErr:   false,
		},
		{
			name:      "empty fields",
			user:      "",
			problemId: "",
			newStatus: "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock producer
			mockProducer := mocks.NewSyncProducer(t, nil)
			defer mockProducer.Close()

			// Create Kafka producer with mock
			kafkaProducer := &KafkaProducer{
				producer: mockProducer,
			}

			// Set up expectations
			mockProducer.ExpectSendMessageAndSucceed()

			// Send message
			err := kafkaProducer.SendStatusChange(tt.user, tt.problemId, tt.newStatus)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

			}

			// Ensure all expectations were met
			assert.NoError(t, mockProducer.Close())
		})
	}
}

func checkKafkaMessage(t *testing.T, msg *sarama.ProducerMessage, userId, problemId, newStatus string) {
	// Check topic
	assert.Equal(t, kafkaTopic, msg.Topic)

	// Check key
	key, err := msg.Key.Encode()
	assert.NoError(t, err)
	assert.Equal(t, userId+"-"+problemId, string(key))

	// Check value
	value, err := msg.Value.Encode()
	assert.NoError(t, err)

	var statusMsg StatusMessage
	err = json.Unmarshal(value, &statusMsg)
	assert.NoError(t, err)

	// Verify message contents
	assert.Equal(t, userId, statusMsg.User)
	assert.Equal(t, problemId, statusMsg.ProblemID)
	assert.Equal(t, newStatus, statusMsg.NewStatus)
	assert.False(t, statusMsg.Timestamp.IsZero())
	assert.True(t, statusMsg.Timestamp.Before(time.Now()))
}

func TestKafkaProducer_Close(t *testing.T) {
	tests := []struct {
		name     string
		producer sarama.SyncProducer
		wantErr  bool
	}{
		{
			name:     "successful close",
			producer: mocks.NewSyncProducer(t, nil),
			wantErr:  false,
		},
		{
			name:     "nil producer",
			producer: nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &KafkaProducer{
				producer: tt.producer,
			}

			err := k.Close()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewKafkaProducer(t *testing.T) {
	tests := []struct {
		name    string
		brokers []string
		wantErr bool
	}{
		{
			name:    "empty brokers",
			brokers: []string{},
			wantErr: true,
		},
		{
			name:    "invalid broker address",
			brokers: []string{"invalid:9092"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			producer, err := NewKafkaProducer(tt.brokers)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, producer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, producer)
				assert.NoError(t, producer.Close())
			}
		})
	}
}

func TestKafkaProducer_SendStatusChange_ProducerError(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	defer mockProducer.Close()

	mockProducer.ExpectSendMessageAndFail(sarama.ErrBrokerNotAvailable)

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
	}

	err := kafkaProducer.SendStatusChange("user1", "problem1", "Created")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send Kafka message")

	assert.NoError(t, mockProducer.Close())
}
