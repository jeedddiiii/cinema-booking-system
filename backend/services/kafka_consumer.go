package services

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"cinema-booking-system/config"
	"cinema-booking-system/models"

	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type KafkaConsumerService struct {
	reader *kafka.Reader
}

func NewKafkaConsumerService(broker, topic, groupID string) *KafkaConsumerService {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{broker},
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})

	log.Printf("✅ Kafka consumer initialized for topic: %s (group: %s)", topic, groupID)
	return &KafkaConsumerService{
		reader: reader,
	}
}

func (s *KafkaConsumerService) Start(ctx context.Context) {
	log.Println("🎧 Kafka consumer started, listening for audit logs...")

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Kafka consumer stopped")
			s.reader.Close()
			return
		default:
			msg, err := s.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("⚠️ Error reading Kafka message: %v", err)
				continue
			}

			if err := s.processMessage(ctx, msg); err != nil {
				log.Printf("⚠️ Error processing message: %v", err)
			}
		}
	}
}

func (s *KafkaConsumerService) processMessage(ctx context.Context, msg kafka.Message) error {
	var auditLog models.AuditLog
	if err := json.Unmarshal(msg.Value, &auditLog); err != nil {
		return err
	}

	if auditLog.ID.IsZero() {
		auditLog.ID = primitive.NewObjectID()
	}

	if auditLog.Timestamp.IsZero() {
		auditLog.Timestamp = time.Now().UTC()
	}

	collection := config.MongoDB.Collection("audit_logs")
	_, err := collection.InsertOne(ctx, auditLog)
	if err != nil {
		return err
	}

	log.Printf("💾 Audit log saved: %s - %s", auditLog.EventType, auditLog.Description)
	return nil
}

func (s *KafkaConsumerService) Close() error {
	if s.reader != nil {
		return s.reader.Close()
	}
	return nil
}
