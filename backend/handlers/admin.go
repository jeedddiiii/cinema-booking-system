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
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) GetBookings(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{}

	if sessionID := c.Query("sessionId"); sessionID != "" {
		if oid, err := primitive.ObjectIDFromHex(sessionID); err == nil {
			filter["sessionId"] = oid
		}
	}

	if userID := c.Query("userId"); userID != "" {
		filter["userId"] = bson.M{"$regex": userID, "$options": "i"}
	}

	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}

	if dateStr := c.Query("date"); dateStr != "" {
		if date, err := time.Parse("2006-01-02", dateStr); err == nil {
			startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
			endOfDay := startOfDay.Add(24 * time.Hour)
			filter["createdAt"] = bson.M{
				"$gte": startOfDay,
				"$lt":  endOfDay,
			}
		}
	}

	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if parsed, err := parseInt(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	skip := (page - 1) * limit

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	collection := config.MongoDB.Collection("bookings")

	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to count bookings",
		})
		return
	}

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to fetch bookings",
		})
		return
	}
	defer cursor.Close(ctx)

	var bookings []models.Booking
	if err := cursor.All(ctx, &bookings); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to decode bookings",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: gin.H{
			"bookings":   bookings,
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

func (h *AdminHandler) GetBookingStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := config.MongoDB.Collection("bookings")

	totalBookings, _ := collection.CountDocuments(ctx, bson.M{})
	confirmedBookings, _ := collection.CountDocuments(ctx, bson.M{"status": "CONFIRMED"})
	cancelledBookings, _ := collection.CountDocuments(ctx, bson.M{"status": "CANCELLED"})

	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	todayBookings, _ := collection.CountDocuments(ctx, bson.M{
		"createdAt": bson.M{"$gte": startOfDay},
	})

	pipeline := []bson.M{
		{"$match": bson.M{"status": "CONFIRMED"}},
		{"$group": bson.M{
			"_id":          nil,
			"totalRevenue": bson.M{"$sum": "$totalAmount"},
		}},
	}
	cursor, _ := collection.Aggregate(ctx, pipeline)
	var revenueResult []bson.M
	cursor.All(ctx, &revenueResult)

	totalRevenue := 0.0
	if len(revenueResult) > 0 {
		if rev, ok := revenueResult[0]["totalRevenue"].(float64); ok {
			totalRevenue = rev
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: gin.H{
			"totalBookings":     totalBookings,
			"confirmedBookings": confirmedBookings,
			"cancelledBookings": cancelledBookings,
			"todayBookings":     todayBookings,
			"totalRevenue":      totalRevenue,
		},
	})
}

func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{}

	if eventType := c.Query("eventType"); eventType != "" {
		filter["eventType"] = eventType
	}

	if sessionID := c.Query("sessionId"); sessionID != "" {
		filter["sessionId"] = sessionID
	}

	page := 1
	limit := 50
	if p := c.Query("page"); p != "" {
		if parsed, err := parseInt(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	skip := (page - 1) * limit

	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	collection := config.MongoDB.Collection("audit_logs")

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to fetch audit logs",
		})
		return
	}
	defer cursor.Close(ctx)

	var logs []models.AuditLog
	if err := cursor.All(ctx, &logs); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to decode audit logs",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    logs,
	})
}

func parseInt(s string) (int, error) {
	var result int
	_, err := parseIntHelper(s, &result)
	return result, err
}

func parseIntHelper(s string, result *int) (bool, error) {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		*result = *result*10 + int(c-'0')
	}
	return true, nil
}
