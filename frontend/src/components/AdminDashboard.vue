<script setup>
import { ref, onMounted, computed } from 'vue'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const bookings = ref([])
const auditLogs = ref([])
const stats = ref(null)
const loading = ref(true)
const activeTab = ref('bookings')

const filters = ref({
  status: '',
  date: '',
  userId: ''
})

const pagination = ref({
  page: 1,
  limit: 10,
  total: 0,
  totalPages: 0
})

async function fetchStats() {
  try {
    const response = await fetch(`${API_URL}/api/admin/bookings/stats`)
    const data = await response.json()
    if (data.success) {
      stats.value = data.data
    }
  } catch (err) {
    console.error('Failed to fetch stats:', err)
  }
}

async function fetchBookings() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: pagination.value.page.toString(),
      limit: pagination.value.limit.toString()
    })
    
    if (filters.value.status) params.append('status', filters.value.status)
    if (filters.value.date) params.append('date', filters.value.date)
    if (filters.value.userId) params.append('userId', filters.value.userId)
    
    const response = await fetch(`${API_URL}/api/admin/bookings?${params}`)
    const data = await response.json()
    
    if (data.success) {
      bookings.value = data.data.bookings || []
      pagination.value.total = data.data.total
      pagination.value.totalPages = data.data.totalPages
    }
  } catch (err) {
    console.error('Failed to fetch bookings:', err)
  } finally {
    loading.value = false
  }
}

async function fetchAuditLogs() {
  loading.value = true
  try {
    const response = await fetch(`${API_URL}/api/admin/audit-logs?limit=50`)
    const data = await response.json()
    
    if (data.success) {
      auditLogs.value = data.data || []
    }
  } catch (err) {
    console.error('Failed to fetch audit logs:', err)
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  pagination.value.page = 1
  fetchBookings()
}

function clearFilters() {
  filters.value = { status: '', date: '', userId: '' }
  pagination.value.page = 1
  fetchBookings()
}

function changePage(newPage) {
  if (newPage >= 1 && newPage <= pagination.value.totalPages) {
    pagination.value.page = newPage
    fetchBookings()
  }
}

function switchTab(tab) {
  activeTab.value = tab
  if (tab === 'bookings') {
    fetchBookings()
  } else if (tab === 'logs') {
    fetchAuditLogs()
  }
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString()
}

function getStatusClass(status) {
  switch (status) {
    case 'CONFIRMED': return 'bg-green-500/20 text-green-400'
    case 'CANCELLED': return 'bg-red-500/20 text-red-400'
    default: return 'bg-gray-500/20 text-gray-400'
  }
}

function getEventTypeClass(eventType) {
  switch (eventType) {
    case 'BOOKING_SUCCESS': return 'bg-green-500/20 text-green-400'
    case 'BOOKING_TIMEOUT': return 'bg-amber-500/20 text-amber-400'
    case 'SEAT_LOCKED': return 'bg-blue-500/20 text-blue-400'
    case 'SEAT_UNLOCKED': return 'bg-purple-500/20 text-purple-400'
    case 'SYSTEM_ERROR': return 'bg-red-500/20 text-red-400'
    default: return 'bg-gray-500/20 text-gray-400'
  }
}

onMounted(() => {
  fetchStats()
  fetchBookings()
})
</script>

<template>
  <div class="min-h-screen bg-cinema-darker text-white p-6">
    <div class="max-w-7xl mx-auto">
      <div class="flex items-center justify-between mb-8">
        <div>
          <h1 class="text-3xl font-bold bg-gradient-to-r from-white to-gray-400 bg-clip-text text-transparent">
            Admin Dashboard
          </h1>
          <p class="text-gray-400 mt-1">Manage bookings and view audit logs</p>
        </div>
        <a href="/" class="btn-secondary flex items-center gap-2">
          ← Back to Booking
        </a>
      </div>

      <div v-if="stats" class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <div class="glass p-4">
          <p class="text-gray-400 text-sm">Total Bookings</p>
          <p class="text-2xl font-bold text-white">{{ stats.totalBookings }}</p>
        </div>
        <div class="glass p-4">
          <p class="text-gray-400 text-sm">Confirmed</p>
          <p class="text-2xl font-bold text-green-400">{{ stats.confirmedBookings }}</p>
        </div>
        <div class="glass p-4">
          <p class="text-gray-400 text-sm">Today's Bookings</p>
          <p class="text-2xl font-bold text-blue-400">{{ stats.todayBookings }}</p>
        </div>
        <div class="glass p-4">
          <p class="text-gray-400 text-sm">Total Revenue</p>
          <p class="text-2xl font-bold text-cinema-gold">฿{{ stats.totalRevenue?.toFixed(2) || '0.00' }}</p>
        </div>
      </div>

      <div class="flex gap-4 mb-6">
        <button 
          @click="switchTab('bookings')"
          :class="['px-4 py-2 rounded-lg transition-colors', 
            activeTab === 'bookings' ? 'bg-rose-600 text-white' : 'bg-white/10 text-gray-400 hover:bg-white/20']"
        >
          Bookings
        </button>
        <button 
          @click="switchTab('logs')"
          :class="['px-4 py-2 rounded-lg transition-colors', 
            activeTab === 'logs' ? 'bg-rose-600 text-white' : 'bg-white/10 text-gray-400 hover:bg-white/20']"
        >
          Audit Logs
        </button>
      </div>

      <div v-if="activeTab === 'bookings'" class="glass p-6">
        <div class="flex flex-wrap gap-4 mb-6">
          <select v-model="filters.status" class="bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white">
            <option value="">All Status</option>
            <option value="CONFIRMED">Confirmed</option>
            <option value="CANCELLED">Cancelled</option>
          </select>
          <input 
            type="date" 
            v-model="filters.date"
            class="bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white"
          />
          <input 
            type="text" 
            v-model="filters.userId"
            placeholder="User ID"
            class="bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white placeholder-gray-500"
          />
          <button @click="applyFilters" class="btn-primary">Apply</button>
          <button @click="clearFilters" class="btn-secondary">Clear</button>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full text-left">
            <thead class="border-b border-white/20">
              <tr>
                <th class="py-3 px-4 text-gray-400 font-medium">Booking ID</th>
                <th class="py-3 px-4 text-gray-400 font-medium">User ID</th>
                <th class="py-3 px-4 text-gray-400 font-medium">Email</th>
                <th class="py-3 px-4 text-gray-400 font-medium">Seats</th>
                <th class="py-3 px-4 text-gray-400 font-medium">Amount</th>
                <th class="py-3 px-4 text-gray-400 font-medium">Status</th>
                <th class="py-3 px-4 text-gray-400 font-medium">Date</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading">
                <td colspan="7" class="py-8 text-center text-gray-400">Loading...</td>
              </tr>
              <tr v-else-if="bookings.length === 0">
                <td colspan="7" class="py-8 text-center text-gray-400">No bookings found</td>
              </tr>
              <tr v-for="booking in bookings" :key="booking.id" class="border-b border-white/10 hover:bg-white/5">
                <td class="py-3 px-4 font-mono text-sm">{{ booking.id?.slice(-8) || '-' }}</td>
                <td class="py-3 px-4 font-mono text-sm text-gray-400">{{ booking.userId?.slice(-8) || '-' }}</td>
                <td class="py-3 px-4">{{ booking.userEmail || '-' }}</td>
                <td class="py-3 px-4">{{ booking.seats?.join(', ') || '-' }}</td>
                <td class="py-3 px-4 text-cinema-gold">฿{{ booking.totalAmount?.toFixed(2) || '0.00' }}</td>
                <td class="py-3 px-4">
                  <span :class="['px-2 py-1 rounded-full text-xs', getStatusClass(booking.status)]">
                    {{ booking.status }}
                  </span>
                </td>
                <td class="py-3 px-4 text-gray-400">{{ formatDate(booking.createdAt) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="pagination.totalPages > 1" class="flex justify-center gap-2 mt-6">
          <button 
            @click="changePage(pagination.page - 1)"
            :disabled="pagination.page === 1"
            class="btn-secondary disabled:opacity-50"
          >
            Previous
          </button>
          <span class="px-4 py-2 text-gray-400">
            Page {{ pagination.page }} of {{ pagination.totalPages }}
          </span>
          <button 
            @click="changePage(pagination.page + 1)"
            :disabled="pagination.page === pagination.totalPages"
            class="btn-secondary disabled:opacity-50"
          >
            Next
          </button>
        </div>
      </div>

      <div v-if="activeTab === 'logs'" class="glass p-6">
        <div class="overflow-x-auto">
          <table class="w-full text-left">
            <thead class="border-b border-white/20">
              <tr>
                <th class="py-3 px-4 text-gray-400 font-medium">Event Type</th>
                <th class="py-3 px-4 text-gray-400 font-medium">Session</th>
                <th class="py-3 px-4 text-gray-400 font-medium">User</th>
                <th class="py-3 px-4 text-gray-400 font-medium">Seats</th>
                <th class="py-3 px-4 text-gray-400 font-medium">Description</th>
                <th class="py-3 px-4 text-gray-400 font-medium">Timestamp</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading">
                <td colspan="6" class="py-8 text-center text-gray-400">Loading...</td>
              </tr>
              <tr v-else-if="auditLogs.length === 0">
                <td colspan="6" class="py-8 text-center text-gray-400">No audit logs found</td>
              </tr>
              <tr v-for="(log, index) in auditLogs" :key="index" class="border-b border-white/10 hover:bg-white/5">
                <td class="py-3 px-4">
                  <span :class="['px-2 py-1 rounded-full text-xs', getEventTypeClass(log.eventType)]">
                    {{ log.eventType }}
                  </span>
                </td>
                <td class="py-3 px-4 font-mono text-sm">{{ log.sessionId?.slice(-8) || '-' }}</td>
                <td class="py-3 px-4">{{ log.userId || '-' }}</td>
                <td class="py-3 px-4">{{ log.seatIds?.join(', ') || '-' }}</td>
                <td class="py-3 px-4 text-gray-300 text-sm max-w-xs truncate">{{ log.description }}</td>
                <td class="py-3 px-4 text-gray-400">{{ formatDate(log.timestamp) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
select option {
  background: #1a1a2e;
  color: white;
}
</style>
