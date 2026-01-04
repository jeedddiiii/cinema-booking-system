<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useSeatStore } from '../stores/seatStore'
import { useWebSocket } from '../composables/useWebSocket'

const props = defineProps({
  sessionId: {
    type: String,
    required: true
  },
  isAuthenticated: {
    type: Boolean,
    default: false
  },
  user: {
    type: Object,
    default: null
  }
})

const router = useRouter()
const seatStore = useSeatStore()
const { isConnected } = useWebSocket(props.sessionId)

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const rows = computed(() => seatStore.seatsByRow)
const rowLabels = computed(() => Object.keys(rows.value).sort())

const canBook = computed(() => props.isAuthenticated)

function getSeatClass(seat) {
  const status = seatStore.getSeatStatus(seat.id)
  
  switch (status) {
    case 'SELECTED':
      return 'seat seat-selected'
    case 'LOCKED':
      return 'seat seat-locked'
    case 'BOOKED':
      return 'seat seat-booked'
    default:
      return 'seat seat-available'
  }
}

function handleSeatClick(seat) {
  if (!props.isAuthenticated) {
    alert('Please sign in to select seats')
    return
  }
  
  const status = seatStore.getSeatStatus(seat.id)
  
  if (status === 'BOOKED') return
  if (status === 'LOCKED') return
  
  seatStore.toggleSeatSelection(seat.id)
}

async function proceedToPayment() {
  if (!props.isAuthenticated) {
    alert('Please sign in to proceed')
    return
  }
  
  if (seatStore.selectedSeats.length === 0) return
  
  seatStore.setLoading(true)
  
  try {
    const response = await fetch(`${API_URL}/api/seats/lock`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        sessionId: props.sessionId,
        seatIds: seatStore.selectedSeats,
        userId: props.user?.id || seatStore.userId
      })
    })
    
    const data = await response.json()
    
    if (data.success) {
      const seatsToLock = [...seatStore.selectedSeats]
      seatsToLock.forEach(seatId => {
        seatStore.updateSeat(seatId, {
          status: 'LOCKED',
          lockedBy: props.user?.id || seatStore.userId
        })
      })
      
      router.push({
        name: 'payment',
        query: {
          sessionId: props.sessionId,
          seats: seatsToLock.join(','),
          total: seatStore.totalSelectedPrice.toFixed(2)
        }
      })
    } else {
      alert(data.error || 'Failed to lock seats. Someone may have taken them.')
    }
  } catch (err) {
    console.error('Lock seats error:', err)
    alert('Failed to connect to server')
  } finally {
    seatStore.setLoading(false)
  }
}

function cancelSelection() {
  seatStore.clearSelection()
}
</script>

<template>
  <div class="space-y-8">
    <div v-if="seatStore.session" class="glass p-6 flex flex-col md:flex-row gap-6">
      <img 
        :src="seatStore.session.moviePoster" 
        :alt="seatStore.session.movieTitle"
        class="w-32 h-48 object-cover rounded-lg shadow-xl"
      />
      <div class="flex-1">
        <h2 class="text-2xl font-bold text-white mb-2">{{ seatStore.session.movieTitle }}</h2>
        <div class="space-y-2 text-gray-400">
          <p class="flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
            </svg>
            {{ seatStore.session.theater }}
          </p>
          <p class="flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            {{ new Date(seatStore.session.startTime).toLocaleString() }}
          </p>
        </div>
        
        <div class="mt-4 flex items-center gap-2">
          <div :class="['w-2 h-2 rounded-full', isConnected ? 'bg-green-500' : 'bg-red-500']"></div>
          <span class="text-sm" :class="isConnected ? 'text-green-400' : 'text-red-400'">
            {{ isConnected ? 'Live updates connected' : 'Reconnecting...' }}
          </span>
        </div>
      </div>
    </div>

    <div class="relative">
      <div class="screen h-8 w-3/4 mx-auto mb-12 flex items-center justify-center">
        <span class="text-slate-800 font-semibold text-sm tracking-widest">SCREEN</span>
      </div>
    </div>

    <div class="glass p-8 overflow-x-auto">
      <div class="flex flex-col items-center gap-2 min-w-fit">
        <div v-for="row in rowLabels" :key="row" class="flex items-center gap-2">
          <span class="w-8 text-center text-gray-400 font-medium">{{ row }}</span>
          
          <div class="flex gap-2">
            <button
              v-for="seat in rows[row]"
              :key="seat.id"
              :class="getSeatClass(seat)"
              :disabled="seat.status === 'BOOKED' || (seat.status === 'LOCKED' && seat.lockedBy !== seatStore.userId)"
              @click="handleSeatClick(seat)"
              :title="`Seat ${seat.id} - $${seat.price}`"
            >
              {{ seat.number }}
            </button>
          </div>
          
          <span class="w-8 text-center text-gray-400 font-medium">{{ row }}</span>
        </div>
      </div>
    </div>

    <div class="flex flex-wrap justify-center gap-6 text-sm">
      <div class="flex items-center gap-2">
        <div class="seat seat-available w-6 h-6 text-xs">1</div>
        <span class="text-gray-400">Available</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="seat seat-selected w-6 h-6 text-xs">1</div>
        <span class="text-gray-400">Selected</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="seat seat-locked w-6 h-6 text-xs">1</div>
        <span class="text-gray-400">Locked</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="seat seat-booked w-6 h-6 text-xs">1</div>
        <span class="text-gray-400">Booked</span>
      </div>
    </div>

    <div class="glass p-6">
      <div class="flex flex-col md:flex-row justify-between items-center gap-4">
        <div class="text-center md:text-left">
          <p class="text-gray-400">Selected Seats</p>
          <p class="text-xl font-bold text-white">
            {{ seatStore.selectedSeats.length > 0 ? seatStore.selectedSeats.join(', ') : 'None' }}
          </p>
          <p v-if="seatStore.myLockedSeats.length > 0" class="text-blue-400 text-sm mt-1">
            Locked: {{ seatStore.myLockedSeats.map(s => s.id).join(', ') }}
          </p>
        </div>
        
        <div class="text-center">
          <p class="text-gray-400">Total Price</p>
          <p class="text-3xl font-bold text-cinema-gold">
            ฿{{ seatStore.totalSelectedPrice.toFixed(2) }}
          </p>
        </div>
        
        <div class="flex gap-3">
          <button 
            v-if="seatStore.selectedSeats.length > 0"
            class="btn-secondary"
            @click="cancelSelection"
          >
            Cancel
          </button>
          
          <button 
            v-if="seatStore.selectedSeats.length > 0"
            class="btn-primary flex items-center gap-2"
            :disabled="seatStore.isLoading"
            @click="proceedToPayment"
          >
            <span v-if="seatStore.isLoading">Processing...</span>
            <template v-else>
              <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
              </svg>
              Proceed to Payment
            </template>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
