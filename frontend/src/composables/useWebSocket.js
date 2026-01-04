import { ref, onMounted, onUnmounted } from 'vue'
import { useSeatStore } from '../stores/seatStore'

export function useWebSocket(sessionId) {
  const ws = ref(null)
  const isConnected = ref(false)
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5
  const reconnectDelay = 3000

  const seatStore = useSeatStore()

  const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'

  function connect() {
    if (ws.value?.readyState === WebSocket.OPEN) {
      return
    }

    const url = `${WS_URL}?sessionId=${sessionId}&userId=${seatStore.userId}`
    
    try {
      ws.value = new WebSocket(url)

      ws.value.onopen = () => {
        console.log('🔌 WebSocket connected')
        isConnected.value = true
        reconnectAttempts.value = 0
        seatStore.setConnectionStatus(true)
      }

      ws.value.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data)
          handleMessage(message)
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }

      ws.value.onerror = (error) => {
        console.error('WebSocket error:', error)
      }

      ws.value.onclose = (event) => {
        console.log('🔌 WebSocket disconnected:', event.code, event.reason)
        isConnected.value = false
        seatStore.setConnectionStatus(false)
        
        if (reconnectAttempts.value < maxReconnectAttempts) {
          reconnectAttempts.value++
          console.log(`Reconnecting... attempt ${reconnectAttempts.value}/${maxReconnectAttempts}`)
          setTimeout(connect, reconnectDelay)
        }
      }
    } catch (err) {
      console.error('Failed to create WebSocket:', err)
    }
  }

  function disconnect() {
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
  }

  function handleMessage(message) {
    switch (message.type) {
      case 'SEAT_UPDATE':
        seatStore.updateSeat(message.data.seatId, {
          status: message.data.status,
          lockedBy: message.data.lockedBy
        })
        break

      case 'SEATS_UPDATE':
        seatStore.updateMultipleSeats(message.data)
        break

      case 'PONG':
        break

      default:
        console.log('Unknown message type:', message.type)
    }
  }

  function send(type, data) {
    if (ws.value?.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify({ type, ...data }))
    }
  }

  let pingInterval = null

  onMounted(() => {
    connect()
    
    pingInterval = setInterval(() => {
      send('PING')
    }, 30000)
  })

  onUnmounted(() => {
    disconnect()
    if (pingInterval) {
      clearInterval(pingInterval)
    }
  })

  return {
    ws,
    isConnected,
    reconnectAttempts,
    connect,
    disconnect,
    send
  }
}
