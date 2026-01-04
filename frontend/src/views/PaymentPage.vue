<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute, onBeforeRouteLeave } from 'vue-router'
import { useSeatStore } from '../stores/seatStore'
import { useAuth } from '../composables/useAuth'

const router = useRouter()
const route = useRoute()
const seatStore = useSeatStore()
const { user, isAuthenticated } = useAuth()

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const isProcessing = ref(false)
const paymentSuccess = ref(false)
const paymentError = ref(null)

const LOCK_DURATION = 300
const timeRemaining = ref(LOCK_DURATION)
const timerInterval = ref(null)

const sessionId = computed(() => route.query.sessionId || seatStore.session?.id)
const lockedSeats = computed(() => {
  const seats = route.query.seats
  if (typeof seats === 'string') return seats.split(',')
  if (Array.isArray(seats)) return seats
  return seatStore.myLockedSeats.map(s => s.id)
})
const totalPrice = computed(() => {
  const price = parseFloat(route.query.total) || seatStore.totalSelectedPrice
  return price.toFixed(2)
})

const formattedTime = computed(() => {
  const minutes = Math.floor(timeRemaining.value / 60)
  const seconds = timeRemaining.value % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
})

const timerProgress = computed(() => {
  return (timeRemaining.value / LOCK_DURATION) * 100
})

const timerColor = computed(() => {
  if (timeRemaining.value <= 60) return 'text-red-500'
  if (timeRemaining.value <= 120) return 'text-yellow-500'
  return 'text-green-500'
})

function startTimer() {
  timerInterval.value = setInterval(() => {
    timeRemaining.value--
    
    if (timeRemaining.value <= 0) {
      handleTimeout()
    }
  }, 1000)
}

function stopTimer() {
  if (timerInterval.value) {
    clearInterval(timerInterval.value)
    timerInterval.value = null
  }
}

const hasUnlocked = ref(false)

async function unlockSeats() {
  if (paymentSuccess.value) return
  if (hasUnlocked.value) return
  
  hasUnlocked.value = true
  
  try {
    await fetch(`${API_URL}/api/seats/unlock`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        sessionId: sessionId.value,
        seatIds: lockedSeats.value,
        userId: user.value?.id || 'anonymous'
      })
    })
    console.log('🔓 Seats unlocked')
  } catch (err) {
    console.error('Failed to unlock seats:', err)
    hasUnlocked.value = false
  }
}

async function handleTimeout() {
  stopTimer()
  await unlockSeats()
  alert('⏰ Payment time expired! Your seats have been released.')
  router.push('/')
}

async function handleCancel() {
  stopTimer()
  await unlockSeats()
  router.push('/')
}

async function handlePayment() {
  isProcessing.value = true
  paymentError.value = null
  
  try {
    await new Promise(resolve => setTimeout(resolve, 1500))
    
    const response = await fetch(`${API_URL}/api/bookings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        sessionId: sessionId.value,
        seatIds: lockedSeats.value,
        userId: user.value?.id || 'anonymous',
        userEmail: user.value?.email || 'user@example.com'
      })
    })
    
    const data = await response.json()
    
    if (data.success) {
      stopTimer()
      paymentSuccess.value = true
      
      lockedSeats.value.forEach(seatId => {
        seatStore.updateSeat(seatId, { status: 'BOOKED', lockedBy: null })
      })
      seatStore.clearSelection()
      
      setTimeout(() => {
        router.push('/')
      }, 3000)
    } else {
      throw new Error(data.error || 'Booking failed')
    }
  } catch (err) {
    paymentError.value = err.message || 'Payment failed. Please try again.'
    console.error('Payment error:', err)
  } finally {
    isProcessing.value = false
  }
}

function handleBeforeUnload(event) {
  if (!paymentSuccess.value && lockedSeats.value.length > 0) {
    const data = JSON.stringify({
      sessionId: sessionId.value,
      seatIds: lockedSeats.value,
      userId: user.value?.id || 'anonymous'
    })
    navigator.sendBeacon(`${API_URL}/api/seats/unlock`, data)
  }
}

onBeforeRouteLeave(async (to, from) => {
  if (!paymentSuccess.value && lockedSeats.value.length > 0) {
    stopTimer()
    await unlockSeats()
  }
  return true
})

onMounted(() => {
  if (!lockedSeats.value || lockedSeats.value.length === 0) {
    alert('No seats selected. Redirecting to seat selection.')
    router.push('/')
    return
  }
  
  if (!isAuthenticated.value) {
    alert('Please sign in to complete payment.')
    router.push('/')
    return
  }
  
  window.addEventListener('beforeunload', handleBeforeUnload)
  
  startTimer()
})

onUnmounted(() => {
  stopTimer()
  window.removeEventListener('beforeunload', handleBeforeUnload)
})
</script>

<template>
  <div class="max-w-2xl mx-auto">
    <div v-if="paymentSuccess" class="glass p-8 text-center">
      <div class="w-20 h-20 mx-auto mb-6 rounded-full bg-green-500/20 flex items-center justify-center">
        <svg xmlns="http://www.w3.org/2000/svg" class="w-10 h-10 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
      </div>
      <h1 class="text-3xl font-bold text-white mb-2">Payment Successful!</h1>
      <p class="text-gray-400 mb-4">Your booking has been confirmed.</p>
      <p class="text-cinema-gold text-lg font-semibold">Seats: {{ lockedSeats.join(', ') }}</p>
      <p class="text-gray-500 mt-4">Redirecting to home page...</p>
    </div>
    
    <div v-else class="space-y-6">
      <div class="glass p-6 text-center">
        <p class="text-gray-400 mb-2">Time remaining to complete payment</p>
        <div class="text-5xl font-bold font-mono" :class="timerColor">
          {{ formattedTime }}
        </div>
        
        <div class="mt-4 h-2 bg-gray-700 rounded-full overflow-hidden">
          <div 
            class="h-full transition-all duration-1000 ease-linear"
            :class="{
              'bg-green-500': timeRemaining > 120,
              'bg-yellow-500': timeRemaining <= 120 && timeRemaining > 60,
              'bg-red-500': timeRemaining <= 60
            }"
            :style="{ width: timerProgress + '%' }"
          ></div>
        </div>
        
        <p v-if="timeRemaining <= 60" class="text-red-400 text-sm mt-2 animate-pulse">
          ⚠️ Hurry! Your seats will be released soon.
        </p>
      </div>
      
      <div class="glass p-6">
        <h2 class="text-xl font-bold text-white mb-4">Order Summary</h2>
        
        <div class="space-y-3">
          <div class="flex justify-between">
            <span class="text-gray-400">Movie</span>
            <span class="text-white">{{ seatStore.session?.movieTitle || 'Demo Movie' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-400">Seats</span>
            <span class="text-white font-medium">{{ lockedSeats.join(', ') }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-400">Quantity</span>
            <span class="text-white">{{ lockedSeats.length }} seat(s)</span>
          </div>
          
          <div class="border-t border-white/10 pt-3 mt-3">
            <div class="flex justify-between text-lg">
              <span class="text-white font-semibold">Total</span>
              <span class="text-cinema-gold font-bold">฿{{ totalPrice }}</span>
            </div>
          </div>
        </div>
      </div>
      
      <div class="glass p-6">
        <h2 class="text-xl font-bold text-white mb-4">Payment Details</h2>
        
        <form @submit.prevent="handlePayment" class="space-y-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Card Number</label>
            <input 
              type="text" 
              value="4242 4242 4242 4242"
              readonly
              class="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-3 text-white"
            />
          </div>
          
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm text-gray-400 mb-1">Expiry</label>
              <input 
                type="text" 
                value="12/28"
                readonly
                class="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-3 text-white"
              />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">CVV</label>
              <input 
                type="text" 
                value="123"
                readonly
                class="w-full bg-white/10 border border-white/20 rounded-lg px-4 py-3 text-white"
              />
            </div>
          </div>
          
          <p class="text-xs text-gray-500">
            * This is a demo payment form. No real payment will be processed.
          </p>
          
          <div v-if="paymentError" class="bg-red-500/20 border border-red-500/50 rounded-lg p-3 text-red-400 text-sm">
            {{ paymentError }}
          </div>
          
          <div class="flex gap-3 pt-4">
            <button 
              type="button"
              @click="handleCancel"
              class="flex-1 btn-secondary"
              :disabled="isProcessing"
            >
              Cancel
            </button>
            <button 
              type="submit"
              class="flex-1 btn-primary"
              :disabled="isProcessing"
            >
              <span v-if="isProcessing" class="flex items-center justify-center gap-2">
                <svg class="animate-spin w-5 h-5" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Processing...
              </span>
              <span v-else>Pay ${{ totalPrice }}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
