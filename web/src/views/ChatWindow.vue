<template>
  <div class="chat-layout">
    <!-- Left: Conversation list -->
    <div class="chat-sidebar">
      <div class="chat-sidebar-header">
        <el-input v-model="searchText" placeholder="搜索会话" size="small" clearable />
      </div>
      <el-tabs v-model="activeTab" class="chat-tabs" stretch>
        <el-tab-pane label="群聊" name="group">
          <div class="chat-sidebar-list">
            <div
              v-for="group in filteredGroups"
              :key="group.chat_id"
              :class="['chat-item', { active: group.chat_id === activeChatId }]"
              @click="switchChat(group)"
            >
              <div class="chat-item-top">
                <span class="chat-item-name">{{ group.name || group.chat_id }}</span>
                <el-badge
                  v-if="unreadMap[group.chat_id]"
                  :value="unreadMap[group.chat_id]"
                  :max="99"
                  class="item-badge"
                />
              </div>
              <div class="chat-item-desc">{{ group.description || group.chat_id }}</div>
            </div>
            <div v-if="filteredGroups.length === 0" class="chat-empty">
              <el-text type="info" size="small">暂无群组</el-text>
            </div>
          </div>
        </el-tab-pane>
        <el-tab-pane label="私聊" name="private">
          <div class="chat-sidebar-list">
            <div
              v-for="conv in filteredPrivateChats"
              :key="conv.chat_id"
              :class="['chat-item', { active: conv.chat_id === activeChatId }]"
              @click="switchChat(conv)"
            >
              <div class="chat-item-top">
                <span class="chat-item-name">{{ conv.name || conv.chat_id }}</span>
                <el-badge
                  v-if="unreadMap[conv.chat_id]"
                  :value="unreadMap[conv.chat_id]"
                  :max="99"
                  class="item-badge"
                />
              </div>
              <div class="chat-item-desc">{{ conv.description }}</div>
            </div>
            <div v-if="filteredPrivateChats.length === 0" class="chat-empty">
              <el-text type="info" size="small">暂无私聊</el-text>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- Right: Chat area -->
    <div class="chat-main">
      <template v-if="activeChatId">
        <!-- Chat header -->
        <div class="chat-header">
          <span class="chat-header-name">{{ activeChatName }}</span>
          <el-tag v-if="connected" type="success" size="small">已连接</el-tag>
          <el-tag v-else type="info" size="small">等待消息</el-tag>
        </div>

        <!-- Messages -->
        <div ref="messageContainer" class="message-container">
          <div v-if="loading" style="text-align: center; padding: 20px">
            <el-text type="info">加载中...</el-text>
          </div>
          <div
            v-for="(msg, idx) in messages"
            :key="idx"
            :class="['message-row', msg.direction === 'out' ? 'message-out' : 'message-in']"
          >
            <div class="message-bubble">
              <div class="message-meta">
                <span class="sender">{{ msg.direction === 'in' ? (msg.sender_name || msg.sender_id || '用户') : '机器人' }}</span>
                <span class="time">{{ formatTime(msg.created_at) }}</span>
              </div>
              <div class="message-text">{{ parseContent(msg.content) }}</div>
            </div>
          </div>
          <div v-if="messages.length === 0 && !loading" style="text-align: center; padding: 40px">
            <el-text type="info">暂无消息</el-text>
          </div>
        </div>

        <!-- Input -->
        <div class="input-area">
          <el-input
            v-model="inputText"
            placeholder="输入消息，按 Enter 发送..."
            @keyup.enter="handleSend"
            :disabled="sending"
          />
          <el-button type="primary" @click="handleSend" :loading="sending" style="margin-left: 8px">
            发送
          </el-button>
        </div>
      </template>

      <!-- No chat selected -->
      <div v-else class="chat-placeholder">
        <el-icon :size="48" color="#c0c4cc"><ChatDotRound /></el-icon>
        <p style="color: #909399; margin-top: 12px">选择一个会话开始聊天</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, inject, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChatDotRound } from '@element-plus/icons-vue'
import { getChats, getConversations, getMessageLogs, sendMessage, getToken } from '../api/client'
import { ElMessage } from 'element-plus'

const unreadMap = inject<Record<string, number>>('unreadMap', {})
const clearChatUnread = inject<(chatId: string) => void>('clearChatUnread', () => {})

interface ChatItem {
  chat_id: string
  name: string
  description: string
}

interface Message {
  id?: number
  chat_id: string
  sender_id: string
  sender_name?: string
  direction: string
  msg_type: string
  content: string
  created_at: string
}

const route = useRoute()
const router = useRouter()

const activeTab = ref('group')
const groups = ref<ChatItem[]>([])
const privateChats = ref<ChatItem[]>([])
const searchText = ref('')
const activeChatId = ref('')
const activeChatName = ref('')
const messages = ref<Message[]>([])
const inputText = ref('')
const loading = ref(false)
const sending = ref(false)
const connected = ref(false)
const messageContainer = ref<HTMLElement>()

let eventSource: EventSource | null = null

const filteredGroups = computed(() => {
  if (!searchText.value) return groups.value
  const keyword = searchText.value.toLowerCase()
  return groups.value.filter(
    (g) => g.name?.toLowerCase().includes(keyword) || g.chat_id.includes(keyword)
  )
})

const filteredPrivateChats = computed(() => {
  if (!searchText.value) return privateChats.value
  const keyword = searchText.value.toLowerCase()
  return privateChats.value.filter(
    (c) => c.name?.toLowerCase().includes(keyword) || c.chat_id.includes(keyword)
  )
})

const scrollToBottom = () => {
  nextTick(() => {
    if (messageContainer.value) {
      messageContainer.value.scrollTop = messageContainer.value.scrollHeight
    }
  })
}

const parseContent = (content: string): string => {
  try {
    const parsed = JSON.parse(content)
    if (parsed.text) return parsed.text
    return content
  } catch {
    return content
  }
}

const formatTime = (t: string) => {
  if (!t) return ''
  return new Date(t).toLocaleTimeString()
}

const loadGroups = async () => {
  try {
    const res = await getChats({ page: 1, page_size: 100 })
    groups.value = res.data.data || []
  } catch (e) {
    console.error('加载群列表失败', e)
  }
}

const loadPrivateChats = async () => {
  try {
    const res = await getConversations()
    const conversations = res.data.data || []
    // Filter by chat_type == "p2p" from the backend
    privateChats.value = conversations
      .filter((c: any) => c.chat_type === 'p2p')
      .map((c: any) => ({
        chat_id: c.chat_id,
        name: c.sender_name || c.chat_id,
        description: `${c.msg_count} 条消息`,
      }))
  } catch (e) {
    console.error('加载私聊列表失败', e)
  }
}

const loadHistory = async () => {
  if (!activeChatId.value) return
  loading.value = true
  try {
    const res = await getMessageLogs({
      chat_id: activeChatId.value,
      page: 1,
      page_size: 50,
    })
    const logs = res.data.data || []
    messages.value = logs.reverse()
    scrollToBottom()
  } catch (e) {
    console.error('加载历史消息失败', e)
  } finally {
    loading.value = false
  }
}

const connectSSE = () => {
  disconnectSSE()
  if (!activeChatId.value) return

  eventSource = new EventSource(`/api/messages/stream?chat_id=${activeChatId.value}&token=${getToken()}`)

  eventSource.onopen = () => {
    connected.value = true
  }

  eventSource.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data) as Message
      messages.value.push(msg)
      scrollToBottom()
    } catch (e) {
      console.error('解析 SSE 消息失败', e)
    }
  }

  eventSource.onerror = () => {
    connected.value = false
  }
}

const disconnectSSE = () => {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
  connected.value = false
}

const switchChat = (item: ChatItem) => {
  if (item.chat_id === activeChatId.value) return
  activeChatId.value = item.chat_id
  activeChatName.value = item.name || item.chat_id
  messages.value = []
  clearChatUnread(item.chat_id)
  router.replace({ path: `/chat/${item.chat_id}`, query: { name: item.name } })
  loadHistory()
  connectSSE()
}

const handleSend = async () => {
  const text = inputText.value.trim()
  if (!text || !activeChatId.value) return

  sending.value = true
  try {
    await sendMessage({
      receive_id: activeChatId.value,
      receive_id_type: 'chat_id',
      msg_type: 'text',
      content: JSON.stringify({ text }),
    })
    inputText.value = ''
    messages.value.push({
      chat_id: activeChatId.value,
      sender_id: '',
      direction: 'out',
      msg_type: 'text',
      content: JSON.stringify({ text }),
      created_at: new Date().toISOString(),
    })
    scrollToBottom()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '发送失败')
  } finally {
    sending.value = false
  }
}

onMounted(async () => {
  await loadGroups()
  await loadPrivateChats()
  const chatId = route.params.chatId as string
  if (chatId) {
    activeChatId.value = chatId
    activeChatName.value = (route.query.name as string) || chatId
    const match = groups.value.find((g) => g.chat_id === chatId)
      || privateChats.value.find((c) => c.chat_id === chatId)
    if (match) activeChatName.value = match.name || chatId
    // Auto-select correct tab
    if (!groups.value.find((g) => g.chat_id === chatId)) {
      activeTab.value = 'private'
    }
    clearChatUnread(chatId)
    loadHistory()
    connectSSE()
  }
})

// Watch for route param changes (e.g., clicking notification while on chat page)
watch(() => route.params.chatId, (newChatId) => {
  if (newChatId && newChatId !== activeChatId.value) {
    const chatId = newChatId as string
    activeChatId.value = chatId
    activeChatName.value = (route.query.name as string) || chatId
    const match = groups.value.find((g) => g.chat_id === chatId)
      || privateChats.value.find((c) => c.chat_id === chatId)
    if (match) activeChatName.value = match.name || chatId
    if (!groups.value.find((g) => g.chat_id === chatId)) {
      activeTab.value = 'private'
    } else {
      activeTab.value = 'group'
    }
    messages.value = []
    clearChatUnread(chatId)
    loadHistory()
    connectSSE()
  }
})

onUnmounted(() => {
  disconnectSSE()
})
</script>

<style scoped>
.chat-layout {
  display: flex;
  height: calc(100vh - 40px);
}

.chat-sidebar {
  width: 260px;
  min-width: 260px;
  background: #fff;
  border-right: 1px solid #e4e7ed;
  border-radius: 8px 0 0 8px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.chat-sidebar-header {
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.chat-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.chat-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
}

.chat-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow: hidden;
}

.chat-sidebar-list {
  height: 100%;
  overflow-y: auto;
}

.chat-item {
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid #fafafa;
  transition: background 0.15s;
}

.chat-item:hover {
  background: #f5f7fa;
}

.chat-item.active {
  background: #ecf5ff;
  border-left: 3px solid #409eff;
}

.chat-item-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.chat-item-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.item-badge {
  flex-shrink: 0;
  margin-left: 6px;
}

.chat-item-desc {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-empty {
  text-align: center;
  padding: 20px;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 0 8px 8px 0;
  overflow: hidden;
}

.chat-header {
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  gap: 8px;
}

.chat-header-name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.message-container {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.message-row {
  display: flex;
  margin-bottom: 12px;
}

.message-in {
  justify-content: flex-start;
}

.message-out {
  justify-content: flex-end;
}

.message-bubble {
  max-width: 65%;
  padding: 8px 12px;
  border-radius: 8px;
  word-break: break-word;
}

.message-in .message-bubble {
  background: #f0f2f5;
}

.message-out .message-bubble {
  background: #409eff;
  color: #fff;
}

.message-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  margin-bottom: 4px;
  opacity: 0.7;
  gap: 12px;
}

.message-text {
  font-size: 14px;
  line-height: 1.5;
}

.input-area {
  display: flex;
  padding: 12px 16px;
  border-top: 1px solid #e4e7ed;
}

.chat-placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
</style>
