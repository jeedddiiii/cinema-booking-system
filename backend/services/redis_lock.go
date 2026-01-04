package services

import (
	"context"
	"fmt"
	"time"

	"cinema-booking-system/config"

	"github.com/redis/go-redis/v9"
)

const (
	LockDuration  = 5 * time.Minute
	LockKeyPrefix = "seat_lock:"
)

type RedisLockService struct {
	client *redis.Client
}

func NewRedisLockService() *RedisLockService {
	return &RedisLockService{
		client: config.RedisClient,
	}
}

func (s *RedisLockService) LockSeat(ctx context.Context, sessionID, seatID, userID string) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("redis client not initialized")
	}

	key := s.getLockKey(sessionID, seatID)
	fmt.Printf("🔒 Attempting to lock: key=%s, user=%s\n", key, userID)

	result, err := s.client.SetNX(ctx, key, userID, LockDuration).Result()
	if err != nil {
		fmt.Printf("❌ Lock failed: %v\n", err)
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if result {
		fmt.Printf("✅ Lock acquired: key=%s\n", key)
	} else {
		fmt.Printf("⚠️ Lock already exists: key=%s\n", key)
	}

	return result, nil
}

func (s *RedisLockService) UnlockSeat(ctx context.Context, sessionID, seatID, userID string) (bool, error) {
	key := s.getLockKey(sessionID, seatID)

	currentOwner, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check lock owner: %w", err)
	}

	if currentOwner != userID {
		return false, fmt.Errorf("cannot unlock: seat is locked by another user")
	}

	deleted, err := s.client.Del(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to release lock: %w", err)
	}

	return deleted > 0, nil
}

func (s *RedisLockService) IsLocked(ctx context.Context, sessionID, seatID string) (bool, string, error) {
	key := s.getLockKey(sessionID, seatID)

	owner, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("failed to check lock status: %w", err)
	}

	return true, owner, nil
}

func (s *RedisLockService) GetLockTTL(ctx context.Context, sessionID, seatID string) (time.Duration, error) {
	key := s.getLockKey(sessionID, seatID)

	ttl, err := s.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get lock TTL: %w", err)
	}

	return ttl, nil
}

func (s *RedisLockService) ExtendLock(ctx context.Context, sessionID, seatID, userID string) (bool, error) {
	key := s.getLockKey(sessionID, seatID)

	currentOwner, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, fmt.Errorf("lock does not exist")
	}
	if err != nil {
		return false, fmt.Errorf("failed to check lock owner: %w", err)
	}

	if currentOwner != userID {
		return false, fmt.Errorf("cannot extend: seat is locked by another user")
	}

	result, err := s.client.Expire(ctx, key, LockDuration).Result()
	if err != nil {
		return false, fmt.Errorf("failed to extend lock: %w", err)
	}

	return result, nil
}

func (s *RedisLockService) LockMultipleSeats(ctx context.Context, sessionID string, seatIDs []string, userID string) ([]string, []string, error) {
	var lockedSeats []string
	var failedSeats []string

	pipe := s.client.Pipeline()

	for _, seatID := range seatIDs {
		key := s.getLockKey(sessionID, seatID)
		pipe.SetNX(ctx, key, userID, LockDuration)
	}

	results, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, seatIDs, fmt.Errorf("failed to execute lock pipeline: %w", err)
	}

	for i, result := range results {
		cmd := result.(*redis.BoolCmd)
		success, _ := cmd.Result()

		if success {
			lockedSeats = append(lockedSeats, seatIDs[i])
		} else {
			failedSeats = append(failedSeats, seatIDs[i])
		}
	}

	if len(failedSeats) > 0 && len(lockedSeats) > 0 {
		for _, seatID := range lockedSeats {
			s.UnlockSeat(ctx, sessionID, seatID, userID)
		}
		return nil, seatIDs, fmt.Errorf("could not lock all seats, some are already locked")
	}

	return lockedSeats, failedSeats, nil
}

func (s *RedisLockService) UnlockMultipleSeats(ctx context.Context, sessionID string, seatIDs []string, userID string) error {
	for _, seatID := range seatIDs {
		if _, err := s.UnlockSeat(ctx, sessionID, seatID, userID); err != nil {
			fmt.Printf("Warning: failed to unlock seat %s: %v\n", seatID, err)
		}
	}
	return nil
}

func (s *RedisLockService) getLockKey(sessionID, seatID string) string {
	return fmt.Sprintf("%s%s:%s", LockKeyPrefix, sessionID, seatID)
}
