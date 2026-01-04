package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"cinema-booking-system/config"
	"cinema-booking-system/models"
	"cinema-booking-system/services"
	"cinema-booking-system/websocket"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Handler struct {
	lockService  *services.RedisLockService
	kafkaService *services.KafkaProducerService
	emailService *services.EmailService
	wsHub        *websocket.Hub
}

func NewHandler(wsHub *websocket.Hub) *Handler {
	return &Handler{
		lockService:  services.NewRedisLockService(),
		kafkaService: services.NewKafkaProducerService(),
		emailService: services.NewEmailService(),
		wsHub:        wsHub,
	}
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Server is healthy",
		Data: gin.H{
			"status":      "ok",
			"timestamp":   time.Now().UTC(),
			"connections": h.wsHub.GetTotalClientCount(),
		},
	})
}

func (h *Handler) GetSessions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := config.MongoDB.Collection("sessions")

	cursor, err := collection.Find(ctx, bson.M{
		"startTime": bson.M{"$gte": time.Now()},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to fetch sessions",
		})
		return
	}
	defer cursor.Close(ctx)

	var sessions []models.MovieSession
	if err := cursor.All(ctx, &sessions); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to decode sessions",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    sessions,
	})
}

func (h *Handler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")

	objectID, err := primitive.ObjectIDFromHex(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid session ID",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := config.MongoDB.Collection("sessions")

	var session models.MovieSession
	if err := collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&session); err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Session not found",
		})
		return
	}

	for i, seat := range session.Seats {
		isLocked, lockedBy, _ := h.lockService.IsLocked(ctx, sessionID, seat.ID)
		if isLocked && seat.Status != models.SeatBooked {
			session.Seats[i].Status = models.SeatLocked
			session.Seats[i].LockedBy = lockedBy
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    session,
	})
}

func (h *Handler) LockSeats(c *gin.Context) {
	var req models.LockSeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	lockedSeats, failedSeats, err := h.lockService.LockMultipleSeats(ctx, req.SessionID, req.SeatIDs, req.UserID)
	if err != nil {
		c.JSON(http.StatusConflict, models.APIResponse{
			Success: false,
			Error:   err.Error(),
			Data: gin.H{
				"failedSeats": failedSeats,
			},
		})
		return
	}

	var seatUpdates []models.SeatUpdate
	for _, seatID := range lockedSeats {
		seatUpdates = append(seatUpdates, models.SeatUpdate{
			SeatID:   seatID,
			Status:   models.SeatLocked,
			LockedBy: req.UserID,
		})
	}
	h.wsHub.BroadcastMultipleSeatUpdates(req.SessionID, seatUpdates)

	go h.kafkaService.LogSeatLocked(context.Background(), req.SessionID, req.UserID, lockedSeats)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Seats locked successfully",
		Data: gin.H{
			"lockedSeats": lockedSeats,
			"expiresIn":   services.LockDuration.Seconds(),
		},
	})
}

func (h *Handler) UnlockSeats(c *gin.Context) {
	var req models.UnlockSeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.lockService.UnlockMultipleSeats(ctx, req.SessionID, req.SeatIDs, req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	var seatUpdates []models.SeatUpdate
	for _, seatID := range req.SeatIDs {
		seatUpdates = append(seatUpdates, models.SeatUpdate{
			SeatID: seatID,
			Status: models.SeatAvailable,
		})
	}
	h.wsHub.BroadcastMultipleSeatUpdates(req.SessionID, seatUpdates)

	go h.kafkaService.LogSeatUnlocked(context.Background(), req.SessionID, req.UserID, req.SeatIDs, "manual")

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Seats unlocked successfully",
	})
}

func (h *Handler) CreateBooking(c *gin.Context) {
	var req models.BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, seatID := range req.SeatIDs {
		isLocked, lockedBy, err := h.lockService.IsLocked(ctx, req.SessionID, seatID)
		if err != nil || !isLocked || lockedBy != req.UserID {
			c.JSON(http.StatusConflict, models.APIResponse{
				Success: false,
				Error:   "Seat " + seatID + " is not locked by you",
			})
			return
		}
	}

	sessionObjectID, _ := primitive.ObjectIDFromHex(req.SessionID)

	sessCollection := config.MongoDB.Collection("sessions")
	var session models.MovieSession
	if err := sessCollection.FindOne(ctx, bson.M{"_id": sessionObjectID}).Decode(&session); err != nil {
		session.MovieTitle = "Cinema Booking"
		session.Theater = "Theater 1"
		session.StartTime = time.Now()
	}

	booking := models.Booking{
		SessionID:   sessionObjectID,
		UserID:      req.UserID,
		UserEmail:   req.UserEmail,
		Seats:       req.SeatIDs,
		TotalAmount: float64(len(req.SeatIDs)) * 150.0,
		Status:      "CONFIRMED",
		CreatedAt:   time.Now().UTC(),
	}
	confirmedAt := time.Now().UTC()
	booking.ConfirmedAt = &confirmedAt

	collection := config.MongoDB.Collection("bookings")
	result, err := collection.InsertOne(ctx, booking)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to create booking",
		})
		return
	}

	bookingID := result.InsertedID.(primitive.ObjectID).Hex()

	for _, seatID := range req.SeatIDs {
		sessCollection.UpdateOne(ctx,
			bson.M{"_id": sessionObjectID, "seats.id": seatID},
			bson.M{"$set": bson.M{"seats.$.status": models.SeatBooked}},
		)
	}

	h.lockService.UnlockMultipleSeats(ctx, req.SessionID, req.SeatIDs, req.UserID)

	var seatUpdates []models.SeatUpdate
	for _, seatID := range req.SeatIDs {
		seatUpdates = append(seatUpdates, models.SeatUpdate{
			SeatID: seatID,
			Status: models.SeatBooked,
		})
	}
	h.wsHub.BroadcastMultipleSeatUpdates(req.SessionID, seatUpdates)

	go h.kafkaService.LogBookingSuccess(context.Background(), req.SessionID, req.UserID, req.SeatIDs, bookingID)

	go func() {
		log.Printf("📧 Attempting to send email to: %s", req.UserEmail)
		err := h.emailService.SendBookingConfirmation(req.UserEmail, services.BookingConfirmationData{
			UserName:    req.UserEmail,
			BookingID:   bookingID,
			MovieTitle:  session.MovieTitle,
			Theater:     session.Theater,
			Seats:       req.SeatIDs,
			TotalAmount: booking.TotalAmount,
			BookingDate: session.StartTime.Format("January 2, 2006 at 3:04 PM"),
		})
		if err != nil {
			log.Printf("❌ Failed to send email to %s: %v", req.UserEmail, err)
		}
	}()

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Booking confirmed",
		Data: gin.H{
			"bookingId":   bookingID,
			"seats":       req.SeatIDs,
			"totalAmount": booking.TotalAmount,
		},
	})
}

func (h *Handler) CreateDemoSession(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := config.MongoDB.Collection("sessions")

	var existingSession models.MovieSession
	err := collection.FindOne(ctx, bson.M{
		"movieTitle": "Inception",
		"theater":    "Theater 1",
	}).Decode(&existingSession)

	if err == nil {
		sessionID := existingSession.ID.Hex()
		for i, seat := range existingSession.Seats {
			isLocked, lockedBy, _ := h.lockService.IsLocked(ctx, sessionID, seat.ID)
			if isLocked && seat.Status != models.SeatBooked {
				existingSession.Seats[i].Status = models.SeatLocked
				existingSession.Seats[i].LockedBy = lockedBy
			}
		}

		c.JSON(http.StatusOK, models.APIResponse{
			Success: true,
			Message: "Using existing demo session",
			Data:    existingSession,
		})
		return
	}

	var seats []models.Seat
	rows := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	for _, row := range rows {
		for num := 1; num <= 10; num++ {
			seatID := fmt.Sprintf("%s%d", row, num)
			seats = append(seats, models.Seat{
				ID:     seatID,
				Row:    row,
				Number: num,
				Status: models.SeatAvailable,
				Price:  150.0,
			})
		}
	}

	session := models.MovieSession{
		MovieTitle:  "Inception",
		MoviePoster: "https://m.media-amazon.com/images/M/MV5BMjAxMzY3NjcxNF5BMl5BanBnXkFtZTcwNTI5OTM0Mw@@._V1_.jpg",
		Theater:     "Theater 1",
		StartTime:   time.Now().Add(2 * time.Hour),
		EndTime:     time.Now().Add(4 * time.Hour),
		Seats:       seats,
		TotalSeats:  len(seats),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	result, err := collection.InsertOne(ctx, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to create demo session",
		})
		return
	}

	session.ID = result.InsertedID.(primitive.ObjectID)

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Demo session created",
		Data:    session,
	})
}
