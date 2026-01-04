package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SeatStatus string

const (
	SeatAvailable SeatStatus = "AVAILABLE"
	SeatLocked    SeatStatus = "LOCKED"
	SeatBooked    SeatStatus = "BOOKED"
)

type Seat struct {
	ID       string     `json:"id" bson:"id"`
	Row      string     `json:"row" bson:"row"`
	Number   int        `json:"number" bson:"number"`
	Status   SeatStatus `json:"status" bson:"status"`
	LockedBy string     `json:"lockedBy,omitempty" bson:"lockedBy,omitempty"`
	LockedAt *time.Time `json:"lockedAt,omitempty" bson:"lockedAt,omitempty"`
	Price    float64    `json:"price" bson:"price"`
}

type MovieSession struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	MovieTitle  string             `json:"movieTitle" bson:"movieTitle"`
	MoviePoster string             `json:"moviePoster" bson:"moviePoster"`
	Theater     string             `json:"theater" bson:"theater"`
	StartTime   time.Time          `json:"startTime" bson:"startTime"`
	EndTime     time.Time          `json:"endTime" bson:"endTime"`
	Seats       []Seat             `json:"seats" bson:"seats"`
	TotalSeats  int                `json:"totalSeats" bson:"totalSeats"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type Booking struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SessionID   primitive.ObjectID `json:"sessionId" bson:"sessionId"`
	UserID      string             `json:"userId" bson:"userId"`
	UserEmail   string             `json:"userEmail" bson:"userEmail"`
	Seats       []string           `json:"seats" bson:"seats"`
	TotalAmount float64            `json:"totalAmount" bson:"totalAmount"`
	Status      string             `json:"status" bson:"status"`
	PaymentID   string             `json:"paymentId,omitempty" bson:"paymentId,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	ConfirmedAt *time.Time         `json:"confirmedAt,omitempty" bson:"confirmedAt,omitempty"`
}

type AuditLog struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	EventType   string             `json:"eventType" bson:"eventType"`
	SessionID   string             `json:"sessionId" bson:"sessionId"`
	UserID      string             `json:"userId" bson:"userId"`
	SeatIDs     []string           `json:"seatIds" bson:"seatIds"`
	Timestamp   time.Time          `json:"timestamp" bson:"timestamp"`
	Description string             `json:"description" bson:"description"`
}

type WSMessage struct {
	Type      string      `json:"type"`
	SessionID string      `json:"sessionId"`
	Data      interface{} `json:"data"`
}

type SeatUpdate struct {
	SeatID   string     `json:"seatId"`
	Status   SeatStatus `json:"status"`
	LockedBy string     `json:"lockedBy,omitempty"`
}

type LockSeatRequest struct {
	SessionID string   `json:"sessionId" binding:"required"`
	SeatIDs   []string `json:"seatIds" binding:"required"`
	UserID    string   `json:"userId" binding:"required"`
}

type UnlockSeatRequest struct {
	SessionID string   `json:"sessionId" binding:"required"`
	SeatIDs   []string `json:"seatIds" binding:"required"`
	UserID    string   `json:"userId" binding:"required"`
}

type BookingRequest struct {
	SessionID string   `json:"sessionId" binding:"required"`
	SeatIDs   []string `json:"seatIds" binding:"required"`
	UserID    string   `json:"userId" binding:"required"`
	UserEmail string   `json:"userEmail" binding:"required"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
