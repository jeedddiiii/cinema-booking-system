<script setup>
import { ref, onMounted, watch } from 'vue'
import SeatMap from '../components/SeatMap.vue'
import { useSeatStore } from '../stores/seatStore'
import { useAuth } from '../composables/useAuth'

const seatStore = useSeatStore()
const { user, isAuthenticated, signInWithGoogle, isLoading: authLoading } = useAuth()

const sessionId = ref(null)
const loading = ref(true)
const error = ref(null)

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

watch(() => user.value, (newUser) => {
  if (newUser?.id) {
    seatStore.setUserId(newUser.id)
  }
}, { immediate: true })

onMounted(async () => {
  if (user.value?.id) {
    seatStore.setUserId(user.value.id)
  }
  
  try {
    const response = await fetch(`${API_URL}/api/sessions/demo`, {
      method: 'POST'
    })
    const data = await response.json()
    
    if (data.success) {
      sessionId.value = data.data.id
      seatStore.setSession(data.data)
    }
  } catch (err) {
    console.error('Failed to create demo session:', err)
    error.value = 'Failed to connect to server. Please ensure the backend is running.'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <div v-if="loading" class="flex flex-col items-center justify-center py-20">
      <div class="w-16 h-16 border-4 border-rose-500/30 border-t-rose-500 rounded-full animate-spin"></div>
      <p class="mt-4 text-gray-400">Loading cinema...</p>
    </div>

    <div v-else-if="error" class="glass p-8 text-center">
      <div class="w-16 h-16 mx-auto mb-4 rounded-full bg-red-500/20 flex items-center justify-center">
        <svg xmlns="http://www.w3.org/2000/svg" class="w-8 h-8 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
      <h2 class="text-xl font-semibold text-white mb-2">Connection Error</h2>
      <p class="text-gray-400 mb-4">{{ error }}</p>
      <button @click="() => window.location.reload()" class="btn-primary">
        Retry
      </button>
    </div>

    <SeatMap 
      v-else-if="sessionId" 
      :session-id="sessionId" 
      :is-authenticated="isAuthenticated"
      :user="user"
    />
  </div>
</template>
