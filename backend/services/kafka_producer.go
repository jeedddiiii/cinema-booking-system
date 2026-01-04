package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cinema-booking-system/config"
	"cinema-booking-system/models"

	"github.com/segmentio/kafka-go"
)

type KafkaProducerService struct {
	writer *kafka.Writer
}

func NewKafkaProducerService() *KafkaProducerService {
	return &KafkaProducerService{
		writer: config.KafkaWriter,
	}
}

func (s *KafkaProducerService) SendAuditLog(ctx context.Context, auditLog models.AuditLog) error {
	if s.writer == nil {
		log.Println("⚠️ Kafka writer not initialized, skipping audit log")
		return nil
	}

	if auditLog.Timestamp.IsZero() {
		auditLog.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(auditLog)
	if err != nil {
		return fmt.Errorf("failed to marshal audit log: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(auditLog.SessionID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(auditLog.EventType)},
			{Key: "timestamp", Value: []byte(auditLog.Timestamp.Format(time.RFC3339))},
		},
	}

	if err := s.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	log.Printf("📝 Audit log sent: %s - %s", auditLog.EventType, auditLog.Description)
	return nil
}

func (s *KafkaProducerService) LogSeatLocked(ctx context.Context, sessionID, userID string, seatIDs []string) error {
	return s.SendAuditLog(ctx, models.AuditLog{
		EventType:   "SEAT_LOCKED",
		SessionID:   sessionID,
		UserID:      userID,
		SeatIDs:     seatIDs,
		Description: fmt.Sprintf("User %s locked seats: %v", userID, seatIDs),
	})
}

func (s *KafkaProducerService) LogSeatUnlocked(ctx context.Context, sessionID, userID string, seatIDs []string, reason string) error {
	return s.SendAuditLog(ctx, models.AuditLog{
		EventType:   "SEAT_UNLOCKED",
		SessionID:   sessionID,
		UserID:      userID,
		SeatIDs:     seatIDs,
		Description: fmt.Sprintf("Seats unlocked (%s): %v", reason, seatIDs),
	})
}

func (s *KafkaProducerService) LogBookingSuccess(ctx context.Context, sessionID, userID string, seatIDs []string, bookingID string) error {
	return s.SendAuditLog(ctx, models.AuditLog{
		EventType:   "BOOKING_SUCCESS",
		SessionID:   sessionID,
		UserID:      userID,
		SeatIDs:     seatIDs,
		Description: fmt.Sprintf("Booking %s confirmed for user %s, seats: %v", bookingID, userID, seatIDs),
	})
}

func (s *KafkaProducerService) LogBookingTimeout(ctx context.Context, sessionID, userID string, seatIDs []string) error {
	return s.SendAuditLog(ctx, models.AuditLog{
		EventType:   "BOOKING_TIMEOUT",
		SessionID:   sessionID,
		UserID:      userID,
		SeatIDs:     seatIDs,
		Description: fmt.Sprintf("Booking timed out for user %s, seats released: %v", userID, seatIDs),
	})
}

func (s *KafkaProducerService) LogBookingCancelled(ctx context.Context, sessionID, userID string, seatIDs []string, reason string) error {
	return s.SendAuditLog(ctx, models.AuditLog{
		EventType:   "BOOKING_CANCELLED",
		SessionID:   sessionID,
		UserID:      userID,
		SeatIDs:     seatIDs,
		Description: fmt.Sprintf("Booking cancelled for user %s (%s), seats: %v", userID, reason, seatIDs),
	})
}
