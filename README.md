# 🎬 Cinema Booking System

Online Cinema Booking System with Real-Time Seat Locking

---

## 1. 🏗️ System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              FRONTEND (Vue.js)                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   SeatMap    │  │ PaymentPage  │  │   Admin      │  │  GoogleAuth  │     │
│  │  Component   │  │   Timer 5m   │  │  Dashboard   │  │   Login      │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘     │
│         │                   │                │                 │            │
│         └───────────────────┴────────────────┴─────────────────┘            │
│                                    │                                        │
│                             WebSocket + REST API                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              BACKEND (Go + Gin)                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   Handlers   │  │ WebSocket    │  │   Services   │  │    Auth      │     │
│  │  (REST API)  │  │     Hub      │  │ (Lock, Email)│  │  (Google)    │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
           │                 │                 │                 │
           ▼                 ▼                 ▼                 ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│   MongoDB    │   │    Redis     │   │    Kafka     │   │    SMTP      │
│  (Sessions,  │   │  (Seat Lock  │   │ (Audit Logs) │   │   (Email)    │
│   Bookings,  │   │   5min TTL)  │   │              │   │              │
│   Users)     │   │              │   │              │   │              │
└──────────────┘   └──────────────┘   └──────────────┘   └──────────────┘
```

---

## 2. 🛠️ Tech Stack Overview

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Frontend** | Vue.js 3 + Vite | SPA with Composition API |
| **Styling** | TailwindCSS | Responsive UI with glassmorphism |
| **Backend** | Go + Gin | REST API + WebSocket server |
| **Database** | MongoDB | Store sessions, bookings, users |
| **Cache/Lock** | Redis | Distributed seat locking (5min TTL) |
| **Message Queue** | Kafka | Audit log streaming |
| **Auth** | Google OAuth | User authentication |
| **Email** | SMTP (Gmail) | Booking confirmation emails |
| **Container** | Docker Compose | Infrastructure orchestration |

---

## 3. 📋 Booking Flow (Step-by-Step)

### Phase 1: Initial Seat Locking

1. User selects specific seats on the interface.
2. Frontend sends a `POST /api/seats/lock` request to the Backend.
3. Backend attempts to set a temporary lock in Redis with a 5-minute Time-To-Live (TTL).
4. Redis confirms the lock is acquired.
5. Backend produces a `SEAT_LOCKED` event to Kafka for downstream services.
6. Backend sends a success response to the Frontend.
7. Frontend redirects the user to the Payment Page and starts a visible 5-minute countdown timer.

---

### Scenario A: Successful Payment (Happy Path)

1. User submits payment details.
2. Frontend sends `POST /api/bookings` to the Backend.
3. Backend checks Redis to verify that the seat locks haven't expired.
4. Backend creates a new booking record in MongoDB.
5. Backend updates the seat status to `BOOKED` in MongoDB.
6. Backend deletes the temporary locks in Redis.
7. Backend produces a `BOOKING_SUCCESS` event to Kafka.
8. Backend triggers the SMTP server to send a confirmation email.
9. Backend notifies the Frontend that the booking is confirmed.

---

### Scenario B: Payment Timeout (Expiration)

1. Redis TTL expires, and the lock key is automatically deleted.
2. Redis sends a Keyspace Notification (expired event) to the Backend.
3. Backend sends a WebSocket message to the Frontend to signal the seats are available again.
4. Backend produces a `LOCK_EXPIRED` event to Kafka.

---

### Scenario C: User Manual Cancellation

1. Frontend sends a `POST /api/seats/unlock` request if the user cancels or leaves.
2. Backend manually deletes the seat locks from Redis.
3. Backend produces a `SEAT_UNLOCKED` event to Kafka.

---

## 4. 🔒 Redis Lock Strategy

### Key Design
```
seat_lock:{sessionId}:{seatId} = {userId}
TTL = 5 minutes
```

### Lock Acquisition (Atomic)
```go
// Use SETNX (SET if Not eXists) for atomicity
result := redis.SetNX(ctx, key, userID, 5*time.Minute)
if result == false {
    // Lock already held by another user
    return error
}
```

### Lock Verification
```go
// Check who owns the lock
lockedBy := redis.Get(ctx, key)
if lockedBy != currentUser {
    return "Seat locked by someone else"
}
```

### Auto-Expiry
- Redis automatically deletes key after 5 minutes
- Keyspace notifications (`__keyevent@0__:expired`) trigger WebSocket updates
- No manual cleanup needed!

### Why Redis?
| Feature | Benefit |
|---------|---------|
| **Atomic operations** | No race conditions |
| **TTL support** | Auto-unlock after timeout |
| **Keyspace notifications** | Real-time expiry events |
| **In-memory** | Extremely fast (<1ms) |
| **Distributed** | Works across multiple backend instances |

---

## 5. 📨 Message Queue (Kafka)

### Purpose
Kafka is used for **audit logging** - tracking all booking events for:
- Analytics & reporting
- Debugging issues
- Compliance & audit trail

### Event Types
| Event | Trigger | Data |
|-------|---------|------|
| `SEAT_LOCKED` | User locks seats | sessionId, userId, seatIds |
| `SEAT_UNLOCKED` | User cancels/leaves | sessionId, userId, seatIds |
| `LOCK_EXPIRED` | 5min timeout | sessionId, seatIds |
| `BOOKING_SUCCESS` | Payment completed | bookingId, userId, seatIds |

### Architecture
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Backend   │────►│    Kafka    │────►│  Consumer   │
│  (Producer) │     │  audit-logs │     │  (MongoDB)  │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
                                        ┌─────────────┐
                                        │   Admin     │
                                        │  Dashboard  │
                                        └─────────────┘
```

### Why Kafka (not direct MongoDB write)?
- **Async processing** - Don't slow down booking flow
- **Scalability** - Handle high throughput
- **Decoupling** - Producers don't wait for consumers
- **Replayability** - Re-process events if needed

---

## 6. 🚀 How to Run
### Prerequisites
- Docker Desktop
- Go 1.21+
- Node.js 18+

### Option A: Run Everything with Docker (Recommended)
```
cd d:\cinema-booking-system
docker compose up --build -d
```
Note: Ensure your .env files are configured in the backend and frontend folders before running this command, as Docker will inject them into the containers.


### Option B: Local Development (Docker Infra + Local App)
#### Step 1: Start Infrastructure
```bash
cd d:\cinema-booking-system
docker compose up -d mongodb redis zookeeper kafka
```

#### Step 2: Configure Backend
```bash
cd backend

# Copy and edit .env
cp .env.example .env
# Edit SMTP credentials for email notifications
```

#### Step 3: Run Backend
```bash
cd backend
go run main.go
```
> Backend runs on `http://localhost:8080`

#### Step 4: Configure Frontend
```bash
cd frontend

# Copy and edit .env
cp .env.example .env
# Set VITE_GOOGLE_CLIENT_ID for Google login
```

#### Step 5: Run Frontend
```bash
cd frontend
npm install
npm run dev
```
> Frontend runs on `http://localhost:5173`

#### Step 6: Access Application
- **Booking System**: http://localhost:5173
- **Admin Dashboard**: http://localhost:5173/admin (requires admin role)

---

## 7. ⚖️ Assumptions & Trade-offs

### Assumptions
1. **Single movie session** - Demo uses one movie for simplicity
2. **150 Baht per seat** - Fixed pricing, no dynamic pricing
3. **5-minute lock** - Reasonable time for payment
4. **Google OAuth only** - No email/password login
5. **No real payment** - Mock payment for demo purposes

### Trade-offs

| Decision | Trade-off |
|----------|-----------|
| **Redis for locks** | ✅ Fast & atomic, ❌ Requires separate service |
| **5-min TTL** | ✅ Prevents seat hoarding, ❌ May rush users |
| **WebSocket** | ✅ Real-time updates, ❌ More complex than polling |
| **Kafka for audit** | ✅ Scalable & async, ❌ Overkill for small apps |
| **MongoDB** | ✅ Flexible schema, ❌ Not ideal for transactions |

### Potential Improvements
1. **Payment gateway** - Integrate Stripe/Omise
2. **Multiple sessions** - Different movies/times
3. **Seat pricing** - VIP/regular zones
4. **Queue system** - For high-demand events
5. **Horizontal scaling** - Multiple backend instances
6. **Admin role management** - UI for role assignment

---

