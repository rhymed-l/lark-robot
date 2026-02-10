<template>
  <!-- Login page: no sidebar -->
  <router-view v-if="isLoginPage" />

  <!-- Main layout with sidebar -->
  <el-container v-else style="height: 100vh">
    <el-aside width="200px" style="background-color: #304156">
      <Sidebar />
    </el-aside>
    <el-main style="padding: 20px; background-color: #f0f2f5">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { reactive, computed, provide, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElNotification } from 'element-plus'
import Sidebar from './components/Sidebar.vue'
import { getToken } from './api/client'

const route = useRoute()
const isLoginPage = computed(() => route.path === '/login')

// Per-chat unread counts: { chatId: count }
const unreadMap = reactive<Record<string, number>>({})

// Total unread count for sidebar badge
const totalUnread = computed(() => Object.values(unreadMap).reduce((sum, n) => sum + n, 0))

// Provide to child components
provide('unreadMap', unreadMap)
provide('totalUnread', totalUnread)
provide('clearChatUnread', (chatId: string) => {
  delete unreadMap[chatId]
})

let eventSource: EventSource | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null

const parseContent = (content: string): string => {
  try {
    const parsed = JSON.parse(content)
    if (parsed.text) return parsed.text
    return content
  } catch {
    return content
  }
}

const connectGlobalSSE = () => {
  if (eventSource) {
    eventSource.close()
  }

  const token = getToken()
  if (!token) return
  eventSource = new EventSource(`/api/messages/stream?token=${token}`)

  eventSource.onopen = () => {
    console.log('[SSE] 全局连接已建立')
  }

  eventSource.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.direction !== 'in') return

      const chatId = msg.chat_id
      // Don't increment if user is currently viewing this chat
      const currentChatId = route.params.chatId as string
      if (route.path.startsWith('/chat') && currentChatId === chatId) return

      // Increment per-chat unread
      unreadMap[chatId] = (unreadMap[chatId] || 0) + 1

      // Browser notification
      if (Notification.permission === 'granted') {
        const senderName = msg.sender_name || '新消息'
        new Notification(`飞书机器人 - ${senderName}`, {
          body: parseContent(msg.content),
          icon: '/favicon.ico',
          tag: 'lark-msg-' + msg.id,
        })
      }

      // In-app notification
      const senderName = msg.sender_name || ''
      ElNotification({
        title: senderName ? `${senderName} 发来消息` : '新消息',
        message: parseContent(msg.content),
        type: 'info',
        duration: 3000,
        position: 'bottom-right',
      })
    } catch (e) {
      // ignore parse errors
    }
  }

  eventSource.onerror = () => {
    console.warn('[SSE] 连接断开，3 秒后重连...')
    eventSource?.close()
    eventSource = null
    // Auto-reconnect after 3 seconds
    if (reconnectTimer) clearTimeout(reconnectTimer)
    reconnectTimer = setTimeout(connectGlobalSSE, 3000)
  }
}

// Reconnect SSE when navigating away from login (i.e., after login)
watch(isLoginPage, (isLogin) => {
  if (!isLogin) connectGlobalSSE()
})

onMounted(() => {
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission()
  }
  if (!isLoginPage.value) connectGlobalSSE()
})

onUnmounted(() => {
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
})
</script>

<style>
body {
  margin: 0;
  padding: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}
</style>
