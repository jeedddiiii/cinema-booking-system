import { ref, computed } from 'vue'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const user = ref(null)
const isLoading = ref(false)
const error = ref(null)

const savedUser = localStorage.getItem('cinema_user')
if (savedUser) {
  try {
    user.value = JSON.parse(savedUser)
  } catch (e) {
    localStorage.removeItem('cinema_user')
  }
}

export function useAuth() {
  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  async function handleGoogleCallback(response) {
    isLoading.value = true
    error.value = null

    try {
      if (response.credential) {
        const payload = decodeJWT(response.credential)
        
        console.log('🔑 Google auth successful, calling backend...')
        
        try {
          const backendResponse = await fetch(`${API_URL}/api/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              googleId: payload.sub,
              email: payload.email,
              name: payload.name,
              picture: payload.picture
            })
          })
          
          const data = await backendResponse.json()
          
          if (data.success) {
            user.value = {
              id: data.data.id,
              googleId: payload.sub,
              email: data.data.email,
              name: data.data.name,
              picture: data.data.picture,
              role: data.data.role,
              token: response.credential
            }
            console.log('✅ Backend login successful:', user.value.email, 'Role:', user.value.role)
          } else {
            throw new Error(data.error || 'Backend login failed')
          }
        } catch (backendErr) {
          console.warn('⚠️ Backend unavailable, using Google data directly:', backendErr.message)
          user.value = {
            id: payload.sub,
            googleId: payload.sub,
            email: payload.email,
            name: payload.name,
            picture: payload.picture,
            role: 'user',
            token: response.credential
          }
          console.log('✅ Login (offline mode):', user.value.email)
        }
        
        localStorage.setItem('cinema_user', JSON.stringify(user.value))
      }
    } catch (err) {
      error.value = err.message || 'Sign in failed'
      console.error('❌ Google Sign-In error:', err)
    } finally {
      isLoading.value = false
    }
  }

  function handleGoogleError(err) {
    console.error('❌ Google Sign-In error:', err)
    error.value = 'Google Sign-In failed'
    isLoading.value = false
  }

  async function refreshRole() {
    if (!user.value?.email) return
    
    try {
      const response = await fetch(`${API_URL}/api/auth/role?email=${encodeURIComponent(user.value.email)}`)
      const data = await response.json()
      
      if (data.success && data.data) {
        user.value.role = data.data.role
        localStorage.setItem('cinema_user', JSON.stringify(user.value))
      }
    } catch (err) {
      console.error('Failed to refresh role:', err)
    }
  }

  async function checkAdminAccess() {
    if (!user.value?.email) return false
    
    try {
      const response = await fetch(`${API_URL}/api/auth/role?email=${encodeURIComponent(user.value.email)}`)
      const data = await response.json()
      
      return data.success && data.data?.role === 'admin'
    } catch (err) {
      console.error('Failed to check admin access:', err)
      return false
    }
  }

  async function signOut() {
    isLoading.value = true
    
    try {
      if (window.google?.accounts?.id) {
        window.google.accounts.id.disableAutoSelect()
      }
      user.value = null
      localStorage.removeItem('cinema_user')
      console.log('✅ Signed out')
    } catch (err) {
      error.value = err.message || 'Sign out failed'
    } finally {
      isLoading.value = false
    }
  }

  function decodeJWT(token) {
    try {
      const base64Url = token.split('.')[1]
      const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
      const jsonPayload = decodeURIComponent(
        atob(base64)
          .split('')
          .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
          .join('')
      )
      return JSON.parse(jsonPayload)
    } catch (e) {
      console.error('Failed to decode JWT:', e)
      return {}
    }
  }

  return {
    user,
    isAuthenticated,
    isAdmin,
    isLoading,
    error,
    handleGoogleCallback,
    handleGoogleError,
    signOut,
    refreshRole,
    checkAdminAccess
  }
}
