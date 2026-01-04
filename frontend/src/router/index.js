import { createRouter, createWebHistory } from 'vue-router'

import HomePage from '../views/HomePage.vue'
import PaymentPage from '../views/PaymentPage.vue'
import AdminDashboard from '../components/AdminDashboard.vue'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const routes = [
  {
    path: '/',
    name: 'home',
    component: HomePage
  },
  {
    path: '/payment',
    name: 'payment',
    component: PaymentPage,
    meta: { requiresAuth: true }
  },
  {
    path: '/admin',
    name: 'admin',
    component: AdminDashboard,
    meta: { requiresAuth: true, requiresAdmin: true }
  },
  {
    path: '/unauthorized',
    name: 'unauthorized',
    component: {
      template: `
        <div class="min-h-screen flex items-center justify-center">
          <div class="glass p-8 text-center max-w-md">
            <div class="w-16 h-16 mx-auto mb-4 rounded-full bg-red-500/20 flex items-center justify-center">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-8 h-8 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
            </div>
            <h1 class="text-2xl font-bold text-white mb-2">Access Denied</h1>
            <p class="text-gray-400 mb-6">You don't have permission to access the Admin Dashboard. Contact an administrator to get admin access.</p>
            <a href="/" class="btn-primary inline-block">Go to Home</a>
          </div>
        </div>
      `
    }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to, from, next) => {
  if (to.meta.requiresAuth) {
    const savedUser = localStorage.getItem('cinema_user')
    
    if (!savedUser) {
      alert('Please sign in to access this page')
      next({ name: 'home' })
      return
    }
    
    if (to.meta.requiresAdmin) {
      try {
        const user = JSON.parse(savedUser)
        
        const response = await fetch(`${API_URL}/api/auth/role?email=${encodeURIComponent(user.email)}`)
        const data = await response.json()
        
        if (!data.success || data.data?.role !== 'admin') {
          console.log('Access denied for:', user.email, 'Role:', data.data?.role)
          next({ name: 'unauthorized' })
          return
        }
        
        console.log('Admin access granted for:', user.email)
      } catch (e) {
        console.error('Failed to check admin role:', e)
        next({ name: 'unauthorized' })
        return
      }
    }
  }
  
  next()
})

export default router
