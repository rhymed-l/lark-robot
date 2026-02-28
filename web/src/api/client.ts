import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

// Request interceptor: attach token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: redirect to login on 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

// Auth helper
export const getToken = () => localStorage.getItem('token') || ''
export const logout = () => {
  localStorage.removeItem('token')
  window.location.href = '/login'
}

// Bot info
export const getBotInfo = () => api.get('/bot/info')

// Dashboard
export const getDashboardStats = () => api.get('/dashboard/stats')

// Messages
export const sendMessage = (data: {
  receive_id: string
  receive_id_type: string
  msg_type: string
  content: string
}) => api.post('/messages/send', data)

export const replyMessage = (data: {
  message_id: string
  msg_type: string
  content: string
}) => api.post('/messages/reply', data)

export const getMessageLogs = (params: {
  page?: number
  page_size?: number
  chat_id?: string
  chat_type?: string
  direction?: string
  source?: string
}) => api.get('/messages/logs', { params })

export const deleteMessage = (messageId: string) => api.delete(`/messages/${messageId}`)

export const getConversations = () => api.get('/messages/conversations')

// Chats
export const getChats = (params?: { page?: number; page_size?: number }) => api.get('/chats', { params })
export const syncChats = () => api.post('/chats/sync')
export const leaveChat = (chatId: string) => api.post(`/chats/${chatId}/leave`)
export const getChatMembers = (chatId: string, params?: { page_token?: string; page_size?: number }) =>
  api.get(`/chats/${chatId}/members`, { params })

// Auto-reply rules
export const getAutoReplyRules = (params?: { page?: number; page_size?: number }) => api.get('/auto-reply-rules', { params })
export const createAutoReplyRule = (data: {
  keyword: string
  reply_text: string
  match_mode?: string
  trigger_mode?: string
  chat_id?: string
  enabled?: boolean
}) => api.post('/auto-reply-rules', data)
export const updateAutoReplyRule = (id: number, data: {
  keyword: string
  reply_text: string
  match_mode?: string
  trigger_mode?: string
  chat_id?: string
  enabled?: boolean
}) => api.put(`/auto-reply-rules/${id}`, data)
export const deleteAutoReplyRule = (id: number) => api.delete(`/auto-reply-rules/${id}`)
export const toggleAutoReplyRule = (id: number) => api.post(`/auto-reply-rules/${id}/toggle`)

// Users
export const getUsers = (params?: { page?: number; page_size?: number; keyword?: string; sort_by?: string; sort_dir?: string }) =>
  api.get('/users', { params })
export const syncUsers = (openIds?: string[]) =>
  api.post('/users/sync', openIds ? { open_ids: openIds } : {})
export const getUserByOpenID = (openId: string) => api.get(`/users/${openId}`)

// Scheduled tasks
export const getScheduledTasks = (params?: { page?: number; page_size?: number }) => api.get('/scheduled-tasks', { params })
export const createScheduledTask = (data: {
  name: string
  cron_expr: string
  chat_id: string
  msg_type?: string
  content: string
  enabled?: boolean
}) => api.post('/scheduled-tasks', data)
export const updateScheduledTask = (id: number, data: {
  name: string
  cron_expr: string
  chat_id: string
  msg_type?: string
  content: string
  enabled?: boolean
}) => api.put(`/scheduled-tasks/${id}`, data)
export const deleteScheduledTask = (id: number) => api.delete(`/scheduled-tasks/${id}`)
export const toggleScheduledTask = (id: number) => api.post(`/scheduled-tasks/${id}/toggle`)
export const runScheduledTask = (id: number) => api.post(`/scheduled-tasks/${id}/run`)

// Upload
export const uploadImage = (file: File) => {
  const formData = new FormData()
  formData.append('file', file)
  return api.post('/upload/image', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 60000,
  })
}

export const uploadFile = (file: File, fileType?: string) => {
  const formData = new FormData()
  formData.append('file', file)
  if (fileType) {
    formData.append('file_type', fileType)
  }
  return api.post('/upload/file', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 60000,
  })
}

export default api
