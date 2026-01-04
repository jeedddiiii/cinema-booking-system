package handlers

import (
	"context"
	"net/http"
	"time"

	"cinema-booking-system/config"
	"cinema-booking-system/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := config.MongoDB.Collection("users")

	var existingUser models.User
	err := collection.FindOne(ctx, bson.M{
		"$or": []bson.M{
			{"googleId": req.GoogleID},
			{"email": req.Email},
		},
	}).Decode(&existingUser)

	now := time.Now().UTC()

	if err == mongo.ErrNoDocuments {
		newUser := models.User{
			GoogleID:  req.GoogleID,
			Email:     req.Email,
			Name:      req.Name,
			Picture:   req.Picture,
			Role:      models.RoleUser,
			CreatedAt: now,
			UpdatedAt: now,
			LastLogin: now,
		}

		result, err := collection.InsertOne(ctx, newUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Error:   "Failed to create user",
			})
			return
		}

		newUser.ID = result.InsertedID.(primitive.ObjectID)

		c.JSON(http.StatusCreated, models.APIResponse{
			Success: true,
			Message: "User registered successfully",
			Data: models.UserResponse{
				ID:      newUser.ID.Hex(),
				Email:   newUser.Email,
				Name:    newUser.Name,
				Picture: newUser.Picture,
				Role:    newUser.Role,
			},
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Database error",
		})
		return
	}

	collection.UpdateOne(ctx,
		bson.M{"_id": existingUser.ID},
		bson.M{
			"$set": bson.M{
				"lastLogin": now,
				"name":      req.Name,
				"picture":   req.Picture,
			},
		},
	)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Login successful",
		Data: models.UserResponse{
			ID:      existingUser.ID.Hex(),
			Email:   existingUser.Email,
			Name:    existingUser.Name,
			Picture: existingUser.Picture,
			Role:    existingUser.Role,
		},
	})
}

func (h *AuthHandler) GetUserRole(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Email is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := config.MongoDB.Collection("users")

	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusOK, models.APIResponse{
			Success: true,
			Data: gin.H{
				"role":   "user",
				"exists": false,
			},
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Database error",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: gin.H{
			"role":   user.Role,
			"exists": true,
		},
	})
}

func (h *AuthHandler) SetUserRole(c *gin.Context) {
	var req struct {
		Email string          `json:"email" binding:"required"`
		Role  models.UserRole `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	if req.Role != models.RoleUser && req.Role != models.RoleAdmin {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid role. Must be 'user' or 'admin'",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := config.MongoDB.Collection("users")

	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx,
		bson.M{"email": req.Email},
		bson.M{
			"$set": bson.M{
				"role":      req.Role,
				"updatedAt": time.Now().UTC(),
			},
			"$setOnInsert": bson.M{
				"email":     req.Email,
				"createdAt": time.Now().UTC(),
			},
		},
		opts,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to update user role",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "User role updated",
		Data: gin.H{
			"email":    req.Email,
			"role":     req.Role,
			"modified": result.ModifiedCount,
			"upserted": result.UpsertedCount,
		},
	})
}
