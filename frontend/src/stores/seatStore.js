import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useSeatStore = defineStore('seats', () => {
  const session = ref(null)
  const seats = ref([])
  const selectedSeats = ref([])
  const userId = ref(`user_${Date.now()}`)
  const isConnected = ref(false)
  const isLoading = ref(false)

  const availableSeats = computed(() => 
    seats.value.filter(seat => seat.status === 'AVAILABLE')
  )

  const lockedSeats = computed(() => 
    seats.value.filter(seat => seat.status === 'LOCKED')
  )

  const bookedSeats = computed(() => 
    seats.value.filter(seat => seat.status === 'BOOKED')
  )

  const myLockedSeats = computed(() => 
    seats.value.filter(seat => seat.status === 'LOCKED' && seat.lockedBy === userId.value)
  )

  const totalSelectedPrice = computed(() => 
    selectedSeats.value.reduce((total, seatId) => {
      const seat = seats.value.find(s => s.id === seatId)
      return total + (seat?.price || 0)
    }, 0)
  )

  const seatsByRow = computed(() => {
    const grouped = {}
    seats.value.forEach(seat => {
      if (!grouped[seat.row]) {
        grouped[seat.row] = []
      }
      grouped[seat.row].push(seat)
    })
    Object.keys(grouped).forEach(row => {
      grouped[row].sort((a, b) => a.number - b.number)
    })
    return grouped
  })

  function setSession(sessionData) {
    session.value = sessionData
    seats.value = sessionData.seats || []
    selectedSeats.value = []
  }

  function setUserId(id) {
    if (id) {
      userId.value = id
      console.log('📍 SeatStore userId set to:', id)
    }
  }

  function updateSeat(seatId, updates) {
    const seatIndex = seats.value.findIndex(s => s.id === seatId)
    if (seatIndex !== -1) {
      seats.value[seatIndex] = { ...seats.value[seatIndex], ...updates }
    }
  }

  function updateMultipleSeats(seatUpdates) {
    seatUpdates.forEach(update => {
      updateSeat(update.seatId, { 
        status: update.status, 
        lockedBy: update.lockedBy 
      })
    })
  }

  function toggleSeatSelection(seatId) {
    const seat = seats.value.find(s => s.id === seatId)
    if (!seat) return

    if (seat.status === 'BOOKED') return
    if (seat.status === 'LOCKED' && seat.lockedBy !== userId.value) return

    const index = selectedSeats.value.indexOf(seatId)
    if (index === -1) {
      selectedSeats.value.push(seatId)
    } else {
      selectedSeats.value.splice(index, 1)
    }
  }

  function clearSelection() {
    selectedSeats.value = []
  }

  function setConnectionStatus(status) {
    isConnected.value = status
  }

  function setLoading(status) {
    isLoading.value = status
  }

  function isSeatSelected(seatId) {
    return selectedSeats.value.includes(seatId)
  }

  function getSeatStatus(seatId) {
    const seat = seats.value.find(s => s.id === seatId)
    if (!seat) return 'UNKNOWN'
    
    if (isSeatSelected(seatId)) return 'SELECTED'
    return seat.status
  }

  return {
    session,
    seats,
    selectedSeats,
    userId,
    isConnected,
    isLoading,
    availableSeats,
    lockedSeats,
    bookedSeats,
    myLockedSeats,
    totalSelectedPrice,
    seatsByRow,
    setSession,
    setUserId,
    updateSeat,
    updateMultipleSeats,
    toggleSeatSelection,
    clearSelection,
    setConnectionStatus,
    setLoading,
    isSeatSelected,
    getSeatStatus
  }
})
