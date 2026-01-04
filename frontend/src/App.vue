<script setup>
import { ref } from 'vue'
import { GoogleLogin } from 'vue3-google-login'
import { useAuth } from './composables/useAuth'
import { useRouter } from 'vue-router'

const { user, isAuthenticated, handleGoogleCallback, handleGoogleError, signOut, isLoading } = useAuth()
const router = useRouter()

const showDropdown = ref(false)

function toggleDropdown() {
  showDropdown.value = !showDropdown.value
}

function handleSignOut() {
  showDropdown.value = false
  signOut()
  router.push('/')
}

function closeDropdown() {
  showDropdown.value = false
}
function onGoogleSuccess(response) {
  handleGoogleCallback(response)
}

function onGoogleError() {
  handleGoogleError('Login failed')
}
</script>

<template>
  <div class="min-h-screen flex flex-col" @click="closeDropdown">
    <header class="glass border-b border-white/10 sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
        <router-link to="/" class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-rose-500 to-rose-700 flex items-center justify-center">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
            </svg>
          </div>
          <div>
            <h1 class="text-xl font-bold bg-gradient-to-r from-white to-gray-400 bg-clip-text text-transparent">
              Cinema Booking
            </h1>
            <p class="text-xs text-gray-400">Select your perfect seats</p>
          </div>
        </router-link>
        
        <div class="flex items-center gap-3">
          <router-link 
            v-if="isAuthenticated && user?.role === 'admin'" 
            to="/admin" 
            class="btn-secondary flex items-center gap-2 text-sm"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2" />
            </svg>
            Admin
          </router-link>
          
          <div v-if="isAuthenticated" class="relative" @click.stop>
            <button 
              @click="toggleDropdown"
              class="flex items-center gap-2 bg-white/10 hover:bg-white/20 px-3 py-2 rounded-lg transition-colors"
            >
              <img 
                v-if="user?.picture" 
                :src="user.picture" 
                :alt="user.name"
                class="w-8 h-8 rounded-full"
              />
              <div v-else class="w-8 h-8 rounded-full bg-gradient-to-br from-rose-500 to-rose-700 flex items-center justify-center text-sm font-bold">
                {{ (user?.name || user?.email || 'U')[0].toUpperCase() }}
              </div>
              
              <span class="text-sm text-white hidden sm:block">{{ user?.name || user?.email }}</span>
              
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-gray-400 transition-transform" :class="{ 'rotate-180': showDropdown }" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            
            <div 
              v-if="showDropdown"
              class="absolute right-0 mt-2 w-56 bg-cinema-dark border border-white/20 rounded-lg shadow-xl overflow-hidden z-50"
            >
              <div class="px-4 py-3 border-b border-white/10">
                <p class="text-sm font-medium text-white">{{ user?.name }}</p>
                <p class="text-xs text-gray-400 truncate">{{ user?.email }}</p>
                <p class="text-xs text-rose-400 mt-1 capitalize">{{ user?.role || 'user' }}</p>
              </div>
              
              <div class="py-1">
                <button 
                  @click="handleSignOut"
                  class="w-full px-4 py-2 text-left text-sm text-gray-300 hover:bg-white/10 flex items-center gap-2"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                  </svg>
                  Sign Out
                </button>
              </div>
            </div>
          </div>
          
          <div v-else>
            <GoogleLogin
              :callback="onGoogleSuccess"
              :error="onGoogleError"
              prompt
              auto-login
            />
          </div>
        </div>
      </div>
    </header>

    <main class="flex-1 max-w-7xl w-full mx-auto px-4 py-8">
      <router-view />
    </main>
  </div>
</template>
