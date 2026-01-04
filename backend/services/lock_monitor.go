package services

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"cinema-booking-system/config"
	"cinema-booking-system/models"
	"cinema-booking-system/websocket"

	"github.com/redis/go-redis/v9"
)

type LockExpiryMonitor struct {
	redisClient  *redis.Client
	kafkaService *KafkaProducerService
	wsHub        *websocket.Hub
}

func NewLockExpiryMonitor(wsHub *websocket.Hub) *LockExpiryMonitor {
	return &LockExpiryMonitor{
		redisClient:  config.RedisClient,
		kafkaService: NewKafkaProducerService(),
		wsHub:        wsHub,
	}
}

func (m *LockExpiryMonitor) Start(ctx context.Context) {
	if m.redisClient == nil {
		log.Println("⚠️ Redis client not available, lock expiry monitor disabled")
		return
	}

	m.redisClient.ConfigSet(ctx, "notify-keyspace-events", "Ex")
	pubsub := m.redisClient.PSubscribe(ctx, "__keyevent@0__:expired")
	defer pubsub.Close()

	log.Println("🔔 Lock expiry monitor started")

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			log.Println("🔔 Lock expiry monitor stopped")
			return
		case msg := <-ch:
			m.handleExpiredKey(ctx, msg.Payload)
		}
	}
}

func (m *LockExpiryMonitor) handleExpiredKey(ctx context.Context, key string) {
	if len(key) < 11 || key[:10] != "seat_lock:" {
		return
	}

	keyParts := parseKeyParts(key[10:])
	if len(keyParts) != 2 {
		return
	}

	sessionID := keyParts[0]
	seatID := keyParts[1]

	log.Printf("⏰ Lock expired: session=%s, seat=%s", sessionID, seatID)

	m.wsHub.BroadcastSeatUpdate(sessionID, models.SeatUpdate{
		SeatID: seatID,
		Status: models.SeatAvailable,
	})

	go m.kafkaService.SendAuditLog(ctx, models.AuditLog{
		EventType:   "LOCK_EXPIRED",
		SessionID:   sessionID,
		SeatIDs:     []string{seatID},
		Description: "Seat lock expired (5 min timeout)",
		Timestamp:   time.Now().UTC(),
	})
}

func parseKeyParts(s string) []string {
	var parts []string
	start := 0
	for i, c := range s {
		if c == ':' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

type AuditLogPersistence struct{}

func NewAuditLogPersistence() *AuditLogPersistence {
	return &AuditLogPersistence{}
}

func (p *AuditLogPersistence) SaveAuditLog(ctx context.Context, auditLog models.AuditLog) error {
	if config.MongoDB == nil {
		return nil
	}

	if auditLog.Timestamp.IsZero() {
		auditLog.Timestamp = time.Now().UTC()
	}

	_, err := config.MongoDB.Collection("audit_logs").InsertOne(ctx, auditLog)
	return err
}

type EnhancedKafkaProducerService struct {
	*KafkaProducerService
	persistence *AuditLogPersistence
}

func NewEnhancedKafkaProducerService() *EnhancedKafkaProducerService {
	return &EnhancedKafkaProducerService{
		KafkaProducerService: NewKafkaProducerService(),
		persistence:          NewAuditLogPersistence(),
	}
}

func (s *EnhancedKafkaProducerService) SendAuditLog(ctx context.Context, auditLog models.AuditLog) error {
	if err := s.persistence.SaveAuditLog(ctx, auditLog); err != nil {
		log.Printf("⚠️ Failed to save audit log to MongoDB: %v", err)
	}

	return s.KafkaProducerService.SendAuditLog(ctx, auditLog)
}

func (s *EnhancedKafkaProducerService) LogSystemError(ctx context.Context, errorType, description string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)

	return s.SendAuditLog(ctx, models.AuditLog{
		EventType:   "SYSTEM_ERROR",
		SessionID:   "",
		UserID:      "system",
		SeatIDs:     nil,
		Description: errorType + ": " + description + " | " + string(detailsJSON),
		Timestamp:   time.Now().UTC(),
	})
}
