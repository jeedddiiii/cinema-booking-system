package websocket

import (
	"log"
	"sync"

	"cinema-booking-system/models"
)

type Hub struct {
	clients    map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
}

type BroadcastMessage struct {
	SessionID string
	Message   []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToSession(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.SessionID] == nil {
		h.clients[client.SessionID] = make(map[*Client]bool)
	}
	h.clients[client.SessionID][client] = true

	log.Printf("🔌 Client connected: session=%s, user=%s", client.SessionID, client.UserID)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.SessionID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)

			if len(clients) == 0 {
				delete(h.clients, client.SessionID)
			}

			log.Printf("🔌 Client disconnected: session=%s, user=%s", client.SessionID, client.UserID)
		}
	}
}

func (h *Hub) broadcastToSession(message *BroadcastMessage) {
	h.mu.RLock()
	clients, ok := h.clients[message.SessionID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	for client := range clients {
		select {
		case client.send <- message.Message:
		default:
			h.unregister <- client
		}
	}
}

func (h *Hub) BroadcastSeatUpdate(sessionID string, seatUpdate models.SeatUpdate) {
	msg := models.WSMessage{
		Type:      "SEAT_UPDATE",
		SessionID: sessionID,
		Data:      seatUpdate,
	}

	data, err := encodeJSON(msg)
	if err != nil {
		log.Printf("Error encoding seat update: %v", err)
		return
	}

	h.broadcast <- &BroadcastMessage{
		SessionID: sessionID,
		Message:   data,
	}

	log.Printf("📡 Broadcast seat update: session=%s, seat=%s, status=%s",
		sessionID, seatUpdate.SeatID, seatUpdate.Status)
}

func (h *Hub) BroadcastMultipleSeatUpdates(sessionID string, seatUpdates []models.SeatUpdate) {
	msg := models.WSMessage{
		Type:      "SEATS_UPDATE",
		SessionID: sessionID,
		Data:      seatUpdates,
	}

	data, err := encodeJSON(msg)
	if err != nil {
		log.Printf("Error encoding seats update: %v", err)
		return
	}

	h.broadcast <- &BroadcastMessage{
		SessionID: sessionID,
		Message:   data,
	}

	log.Printf("📡 Broadcast %d seat updates for session=%s", len(seatUpdates), sessionID)
}

func (h *Hub) GetClientCount(sessionID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[sessionID]; ok {
		return len(clients)
	}
	return 0
}

func (h *Hub) GetTotalClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}
	return total
}
