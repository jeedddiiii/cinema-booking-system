package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

type User struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	GoogleID  string             `json:"googleId" bson:"googleId"`
	Email     string             `json:"email" bson:"email"`
	Name      string             `json:"name" bson:"name"`
	Picture   string             `json:"picture,omitempty" bson:"picture,omitempty"`
	Role      UserRole           `json:"role" bson:"role"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
	LastLogin time.Time          `json:"lastLogin" bson:"lastLogin"`
}

type UserResponse struct {
	ID      string   `json:"id"`
	Email   string   `json:"email"`
	Name    string   `json:"name"`
	Picture string   `json:"picture,omitempty"`
	Role    UserRole `json:"role"`
}

type LoginRequest struct {
	GoogleID string `json:"googleId" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Name     string `json:"name"`
	Picture  string `json:"picture,omitempty"`
}
