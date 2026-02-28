<template>
  <!-- Login page: no sidebar -->
  <router-view v-if="isLoginPage" />

  <!-- Main layout with sidebar -->
  <el-container v-else style="height: 100vh">
    <el-aside width="200px" style="background-color: #304156">
      <Sidebar />
    </el-aside>
    <el-main style="padding: 20px; background-color: #f0f2f5; overflow: hidden">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { h, reactive, computed, provide, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElNotification } from 'element-plus'
import Sidebar from './components/Sidebar.vue'
import { getToken, getChats, getConversations, getBotInfo } from './api/client'

const router = useRouter()

// Chat name cache for notifications
const chatNameCache = reactive<Record<string, string>>({})

const route = useRoute()
const isLoginPage = computed(() => route.path === '/login')

// Per-chat unread counts: { chatId: count }
const unreadMap = reactive<Record<string, number>>({})

// Total unread count for sidebar badge
const totalUnread = computed(() => Object.values(unreadMap).reduce((sum, n) => sum + n, 0))

// Bot info
const botInfo = reactive({ name: '', open_id: '', avatar_url: '' })
const loadBotInfo = async () => {
  try {
    const res = await getBotInfo()
    Object.assign(botInfo, res.data)
  } catch {
    // ignore
  }
}

// Provide to child components
provide('unreadMap', unreadMap)
provide('totalUnread', totalUnread)
provide('clearChatUnread', (chatId: string) => {
  delete unreadMap[chatId]
})
provide('botInfo', botInfo)

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

      const senderName = msg.sender_name || ''
      const msgText = parseContent(msg.content)
      const isGroup = msg.chat_type === 'group'

      // Auto-refresh cache for unknown chats
      if (!chatNameCache[chatId]) {
        if (isGroup) {
          // Async fetch group name (will be available for next notification)
          loadChatNames()
        } else if (senderName) {
          chatNameCache[chatId] = senderName
        }
      }

      const chatName = chatNameCache[chatId] || (isGroup ? '群聊' : senderName || '私聊')

      // Build notification title: "[群名] 用户名" or "用户名 (私聊)"
      const notifyTitle = isGroup
        ? `[${chatName}] ${senderName || '新消息'}`
        : `${senderName || chatName}`

      // Browser notification (click to focus window)
      if (Notification.permission === 'granted') {
        const n = new Notification(notifyTitle, {
          body: msgText,
          icon: '/favicon.ico',
          tag: 'lark-msg-' + msg.id,
        })
        n.onclick = () => {
          window.focus()
          router.push(`/chat/${chatId}`)
        }
      }

      // In-app notification (click to jump to chat)
      let notifyInstance: any = null
      notifyInstance = ElNotification({
        title: notifyTitle,
        message: h('div', {
          style: 'cursor: pointer',
          onClick: () => {
            router.push(`/chat/${chatId}`)
            notifyInstance?.close()
          },
        }, msgText),
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
watch(isLoginPage, async (isLogin) => {
  if (!isLogin) {
    await Promise.all([loadChatNames(), loadBotInfo()])
    connectGlobalSSE()
  }
})

const loadChatNames = async () => {
  try {
    const groupRes = await getChats({ page: 1, page_size: 100 })
    for (const g of (groupRes.data.data || [])) {
      if (g.name) chatNameCache[g.chat_id] = g.name
    }
    const convRes = await getConversations()
    for (const c of (convRes.data.data || [])) {
      if (c.chat_type === 'p2p' && c.sender_name) {
        chatNameCache[c.chat_id] = c.sender_name
      }
    }
  } catch {
    // ignore
  }
}

onMounted(async () => {
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission()
  }
  if (!isLoginPage.value) {
    await Promise.all([loadChatNames(), loadBotInfo()])
    connectGlobalSSE()
  }
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
