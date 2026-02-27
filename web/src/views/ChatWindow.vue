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
              <div class="chat-item-body">
                <el-avatar :size="36" :src="group.avatar || undefined" class="chat-item-avatar">
                  {{ (group.name || '?').charAt(0) }}
                </el-avatar>
                <div class="chat-item-info">
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
              </div>
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
              <div class="chat-item-body">
                <el-avatar
                  :size="36"
                  :src="avatarMap[conv.sender_id || ''] || avatarMap[conv.chat_id] || undefined"
                  class="chat-item-avatar"
                >
                  {{ (conv.name || '?').charAt(0) }}
                </el-avatar>
                <div class="chat-item-info">
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
              </div>
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
        <div ref="messageContainer" class="message-container" @click="handleImageClick">
          <div v-if="loading" style="text-align: center; padding: 20px">
            <el-text type="info">加载中...</el-text>
          </div>
          <div
            v-for="(msg, idx) in messages"
            :key="idx"
            :class="[msg.recalled ? 'message-row message-recall' : 'message-row ' + (msg.direction === 'out' ? 'message-out' : 'message-in')]"
            @contextmenu.prevent="!msg.recalled && showContextMenu($event, msg)"
          >
            <template v-if="msg.recalled">
              <div class="recall-notice" @click="handleRecallClick($event)">
                {{ msg.direction === 'out' ? '你' : (msg.sender_name || msg.sender_id || '对方') }}撤回了一条消息：<span class="recall-body" v-html="renderRecallContent(msg)"></span>
                <span v-if="msg.direction === 'out' && !isResourceMsg(msg)" class="recall-reedit" @click.stop="handleReedit(msg)">重新编辑</span>
              </div>
            </template>
            <template v-else>
              <el-avatar
                v-if="msg.direction === 'in'"
                :size="32"
                :src="avatarMap[msg.sender_id] || undefined"
                class="message-avatar"
              >
                {{ (msg.sender_name || msg.sender_id || '?').charAt(0) }}
              </el-avatar>
              <div class="message-bubble">
                <div class="message-meta">
                  <span v-if="msg.direction === 'in'" class="sender sender-clickable" @click="insertMention(msg)">{{ msg.sender_name || msg.sender_id || '用户' }}</span>
                  <span v-else class="sender">机器人</span>
                  <span class="time">{{ formatTime(msg.created_at) }}</span>
                </div>
                <div class="message-text" v-html="renderContent(msg.content, msg.msg_type, msg.message_id)"></div>
              </div>
            </template>
          </div>
          <div v-if="messages.length === 0 && !loading" style="text-align: center; padding: 40px">
            <el-text type="info">暂无消息</el-text>
          </div>
        </div>

        <!-- Reply bar -->
        <div v-if="replyTo" class="reply-bar">
          <div class="reply-info">
            <span class="reply-label">回复</span>
            <span class="reply-name">{{ replyTo.direction === 'in' ? (replyTo.sender_name || replyTo.sender_id || '用户') : '机器人' }}</span>
            <span class="reply-text">{{ parseContent(replyTo.content) }}</span>
          </div>
          <el-button text size="small" @click="replyTo = null" style="color: #909399">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>

        <!-- Input -->
        <div class="input-area">
          <div
            ref="editorRef"
            class="mention-editor"
            :contenteditable="sending ? 'false' : 'true'"
            :class="{ disabled: sending }"
            :data-placeholder="replyTo ? '回复消息...' : '输入消息，按 Enter 发送...'"
            @keydown="handleEditorKeydown"
          ></div>
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

    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        @click.stop
      >
        <div class="context-menu-item" @click="handleReply">回复</div>
        <div
          v-if="contextMenu.msg?.direction === 'in' && contextMenu.msg?.sender_id"
          class="context-menu-item"
          @click="handleStartPrivateChat"
        >私聊</div>
        <div
          v-if="contextMenu.msg?.direction === 'out' && contextMenu.msg?.message_id"
          class="context-menu-item context-menu-danger"
          @click="handleRecall"
        >撤回</div>
      </div>
    </Teleport>

    <!-- Image preview -->
    <el-image-viewer
      v-if="previewVisible"
      :url-list="previewList"
      :initial-index="0"
      @close="previewVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, inject, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChatDotRound, Close } from '@element-plus/icons-vue'
import { getChats, getConversations, getMessageLogs, sendMessage, replyMessage, deleteMessage, getToken, getUserByOpenID } from '../api/client'
import { ElMessage } from 'element-plus'

const unreadMap = inject<Record<string, number>>('unreadMap', {})
const clearChatUnread = inject<(chatId: string) => void>('clearChatUnread', () => {})

interface ChatItem {
  chat_id: string
  name: string
  description: string
  avatar?: string
  sender_id?: string
}

interface Message {
  id?: number
  message_id?: string
  chat_id: string
  sender_id: string
  sender_name?: string
  direction: string
  msg_type: string
  content: string
  created_at: string
  recalled?: boolean
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
const loading = ref(false)
const sending = ref(false)
const connected = ref(false)
const messageContainer = ref<HTMLElement>()
const editorRef = ref<HTMLElement>()
const replyTo = ref<Message | null>(null)
const contextMenu = ref({ visible: false, x: 0, y: 0, msg: null as Message | null })

// Image preview
const previewVisible = ref(false)
const previewList = ref<string[]>([])

const handleImageClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (target.tagName === 'IMG' && (target.classList.contains('msg-image') || target.classList.contains('msg-sticker'))) {
    previewList.value = [(target as HTMLImageElement).src]
    previewVisible.value = true
  }
}

// User info cache (avatar + name)
const avatarMap = ref<Record<string, string>>({})
const userNameMap = ref<Record<string, string>>({})
const userInfoLoading = new Set<string>()
const userInfoFetched = new Set<string>() // tracks successfully fetched IDs

// Deduplicate private chats by sender_id: merge entries for the same user
const deduplicatePrivateChats = () => {
  const seen = new Map<string, number>() // sender_id -> index of best entry
  const toRemove = new Set<number>()

  for (let i = 0; i < privateChats.value.length; i++) {
    const chat = privateChats.value[i]
    const key = chat.sender_id || chat.chat_id
    if (!key) continue

    if (seen.has(key)) {
      const existingIdx = seen.get(key)!
      const existing = privateChats.value[existingIdx]
      // Keep the one with the real name, merge description
      const existingHasName = existing.name && !existing.name.startsWith('ou_') && !existing.name.startsWith('oc_')
      const currentHasName = chat.name && !chat.name.startsWith('ou_') && !chat.name.startsWith('oc_')
      if (!existingHasName && currentHasName) {
        // Current is better, swap
        existing.name = chat.name
        if (chat.avatar) existing.avatar = chat.avatar
      }
      toRemove.add(i)
    } else {
      seen.set(key, i)
    }
  }

  if (toRemove.size > 0) {
    privateChats.value = privateChats.value.filter((_, i) => !toRemove.has(i))
  }
}

const fetchUserInfo = async (openId: string) => {
  if (!openId || userInfoLoading.has(openId) || userInfoFetched.has(openId)) return
  userInfoLoading.add(openId)
  try {
    const res = await getUserByOpenID(openId)
    const user = res.data.data
    if (user) {
      userInfoFetched.add(openId)
      const avatar = user.avatar || ''
      avatarMap.value[openId] = avatar
      if (user.name) {
        userNameMap.value[openId] = user.name
        // Also store under matching sender_id/chat_id so template lookups work
        privateChats.value.forEach(chat => {
          if (chat.sender_id === openId && chat.chat_id !== openId) {
            avatarMap.value[chat.chat_id] = avatar
          }
          if (chat.chat_id === openId && chat.sender_id && chat.sender_id !== openId) {
            avatarMap.value[chat.sender_id] = avatar
            userNameMap.value[chat.sender_id] = user.name
          }
          // Update name if still showing an ID
          if ((chat.sender_id === openId || chat.chat_id === openId) &&
              (!chat.name || chat.name.startsWith('ou_') || chat.name.startsWith('oc_'))) {
            chat.name = user.name
          }
        })
        deduplicatePrivateChats()
        // Update active chat header if needed
        if (activeChatName.value.startsWith('ou_') || activeChatName.value.startsWith('oc_')) {
          if (activeChatId.value === openId || privateChats.value.some(c => c.chat_id === activeChatId.value && c.sender_id === openId)) {
            activeChatName.value = user.name
          }
        }
      }
    }
  } catch {
    // Don't cache failures — allow retry on next call
  } finally {
    userInfoLoading.delete(openId)
  }
}

const fetchAvatar = fetchUserInfo

const fetchAvatars = (msgs: Message[]) => {
  const ids = new Set(msgs.filter(m => m.direction === 'in' && m.sender_id).map(m => m.sender_id))
  ids.forEach(id => fetchUserInfo(id))
}

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

const moveToTop = (chatId: string) => {
  const groupIdx = groups.value.findIndex(g => g.chat_id === chatId)
  if (groupIdx > 0) {
    const item = groups.value[groupIdx]
    groups.value = [item, ...groups.value.slice(0, groupIdx), ...groups.value.slice(groupIdx + 1)]
    return
  }
  const privateIdx = privateChats.value.findIndex(c => c.chat_id === chatId)
  if (privateIdx > 0) {
    const item = privateChats.value[privateIdx]
    privateChats.value = [item, ...privateChats.value.slice(0, privateIdx), ...privateChats.value.slice(privateIdx + 1)]
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (messageContainer.value) {
      messageContainer.value.scrollTop = messageContainer.value.scrollHeight
    }
  })
}

const escapeHtml = (text: string): string => {
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

const highlightMentions = (text: string): string => {
  // Split on @[Name] markers, process each part separately
  return text.split(/(@\[[^\]]+\])/).map(part => {
    const m = part.match(/^@\[([^\]]+)\]$/)
    if (m) return `<span class="mention">@${escapeHtml(m[1])}</span>`
    // Old format fallback: @Word
    return escapeHtml(part).replace(/@(\S+)/g, '<span class="mention">@$1</span>')
  }).join('')
}

const resourceUrl = (messageId: string, key: string, type: string = 'image'): string => {
  return `/api/images/${encodeURIComponent(messageId)}/${encodeURIComponent(key)}?type=${type}&token=${getToken()}`
}

const imageUrl = (messageId: string, imageKey: string): string => {
  return resourceUrl(messageId, imageKey, 'image')
}

const renderContent = (content: string, msgType: string, messageId?: string): string => {
  // Image message: {"image_key":"img_xxx"}
  if (msgType === 'image' && messageId) {
    try {
      const parsed = JSON.parse(content)
      if (parsed.image_key) {
        return `<img class="msg-image" src="${imageUrl(messageId, parsed.image_key)}" alt="图片" loading="lazy" />`
      }
    } catch {}
    return '[图片]'
  }

  // Sticker message: {"file_key":"v2_xxx"}
  if (msgType === 'sticker' && messageId) {
    try {
      const parsed = JSON.parse(content)
      if (parsed.file_key) {
        return `<img class="msg-sticker" src="${resourceUrl(messageId, parsed.file_key, 'file')}" alt="表情" loading="lazy" />`
      }
    } catch {}
    return '[表情]'
  }

  try {
    const parsed = JSON.parse(content)

    // Plain text: {"text":"hello"}
    if (parsed.text) {
      const text = parsed.text.replace(/<at user_id="[^"]*">([^<]*)<\/at>/g, '@[$1]')
      return highlightMentions(text)
    }

    // Rich text (post): {"title":"","content":[[...]]}
    if (parsed.content && Array.isArray(parsed.content)) {
      return renderPostContent(parsed.content, messageId)
    }

    return escapeHtml(content)
  } catch {
    return escapeHtml(content)
  }
}

const renderPostContent = (content: any[][], messageId?: string): string => {
  const parts: string[] = []
  for (const line of content) {
    if (!Array.isArray(line)) continue
    for (const elem of line) {
      if (elem.tag === 'text' && elem.text) {
        parts.push(escapeHtml(elem.text))
      } else if (elem.tag === 'at' && elem.user_name) {
        parts.push(`<span class="mention">@${escapeHtml(elem.user_name)}</span>`)
      } else if (elem.tag === 'img' && elem.image_key && messageId) {
        parts.push(`<img class="msg-image" src="${imageUrl(messageId, elem.image_key)}" alt="图片" loading="lazy" />`)
      }
    }
  }
  return parts.join('')
}

const parseContent = (content: string): string => {
  try {
    const parsed = JSON.parse(content)
    if (parsed.text) return parsed.text.replace(/<at user_id="[^"]*">([^<]*)<\/at>/g, '@$1')
    return content
  } catch {
    return content
  }
}

const isResourceMsg = (msg: Message): boolean => {
  return ['image', 'file', 'media', 'audio', 'sticker'].includes(msg.msg_type)
}

const renderRecallContent = (msg: Message): string => {
  const resourceLabels: Record<string, string> = {
    image: '[图片]',
    file: '[文件]',
    media: '[视频]',
    audio: '[语音]',
    sticker: '[表情]',
  }

  // Simple resource message (image-only, file-only, etc.)
  if (resourceLabels[msg.msg_type]) {
    const label = resourceLabels[msg.msg_type]
    if (msg.msg_type === 'image' && msg.message_id) {
      try {
        const parsed = JSON.parse(msg.content)
        if (parsed.image_key) {
          return `<span class="recall-resource" data-action="preview" data-url="${escapeHtml(imageUrl(msg.message_id, parsed.image_key))}">${label}</span>`
        }
      } catch {}
    } else if (msg.msg_type === 'sticker' && msg.message_id) {
      try {
        const parsed = JSON.parse(msg.content)
        if (parsed.file_key) {
          return `<span class="recall-resource" data-action="preview" data-url="${escapeHtml(resourceUrl(msg.message_id, parsed.file_key, 'file'))}">${label}</span>`
        }
      } catch {}
    } else if (msg.msg_type === 'file' && msg.message_id) {
      try {
        const parsed = JSON.parse(msg.content)
        if (parsed.file_key) {
          const url = `/api/images/${encodeURIComponent(msg.message_id)}/${encodeURIComponent(parsed.file_key)}?token=${getToken()}`
          return `<span class="recall-resource" data-action="download" data-url="${escapeHtml(url)}">${label}</span>`
        }
      } catch {}
    }
    return `<span class="recall-resource">${label}</span>`
  }

  // Rich text (post) with mixed content
  try {
    const parsed = JSON.parse(msg.content)
    if (parsed.content && Array.isArray(parsed.content)) {
      return renderRecallPost(parsed.content, msg.message_id)
    }
    if (parsed.text) {
      const text = parsed.text.replace(/<at user_id="[^"]*">([^<]*)<\/at>/g, '@$1')
      return `<span class="recall-content" title="${escapeHtml(text)}">${escapeHtml(text)}</span>`
    }
  } catch {}

  return `<span class="recall-content" title="${escapeHtml(msg.content)}">${escapeHtml(msg.content)}</span>`
}

const renderRecallPost = (content: any[][], messageId?: string): string => {
  const parts: string[] = []
  for (const line of content) {
    if (!Array.isArray(line)) continue
    for (const elem of line) {
      if (elem.tag === 'text' && elem.text) {
        parts.push(escapeHtml(elem.text))
      } else if (elem.tag === 'at' && elem.user_name) {
        parts.push(`@${escapeHtml(elem.user_name)}`)
      } else if (elem.tag === 'img' && elem.image_key && messageId) {
        const url = imageUrl(messageId, elem.image_key)
        parts.push(`<span class="recall-resource" data-action="preview" data-url="${escapeHtml(url)}">[图片]</span>`)
      }
    }
  }
  const text = parts.join('')
  return `<span class="recall-content" title="${text.replace(/<[^>]*>/g, '')}">${text}</span>`
}

const handleRecallClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (!target.classList.contains('recall-resource')) return
  const action = target.dataset.action
  const url = target.dataset.url
  if (!action || !url) return
  if (action === 'preview') {
    previewList.value = [url]
    previewVisible.value = true
  } else if (action === 'download') {
    window.open(url, '_blank')
  }
}

const insertMention = (msg: Message) => {
  if (!editorRef.value || sending.value) return
  const name = msg.sender_name || msg.sender_id || '用户'
  const userId = msg.sender_id

  editorRef.value.focus()

  const mentionSpan = document.createElement('span')
  mentionSpan.className = 'mention-tag'
  mentionSpan.contentEditable = 'false'
  mentionSpan.dataset.userId = userId
  mentionSpan.textContent = `@${name}`

  const space = document.createTextNode('\u00A0')

  const selection = window.getSelection()
  if (selection && selection.rangeCount > 0 && editorRef.value.contains(selection.anchorNode)) {
    const range = selection.getRangeAt(0)
    range.deleteContents()
    range.insertNode(space)
    range.insertNode(mentionSpan)
    range.setStartAfter(space)
    range.collapse(true)
    selection.removeAllRanges()
    selection.addRange(range)
  } else {
    editorRef.value.appendChild(mentionSpan)
    editorRef.value.appendChild(space)
    const range = document.createRange()
    range.setStartAfter(space)
    range.collapse(true)
    const sel = window.getSelection()
    sel?.removeAllRanges()
    sel?.addRange(range)
  }
}

const getEditorText = (): string => {
  if (!editorRef.value) return ''
  let text = ''
  const walk = (node: Node) => {
    if (node.nodeType === Node.TEXT_NODE) {
      text += node.textContent || ''
    } else if (node instanceof HTMLElement) {
      if (node.classList.contains('mention-tag')) {
        const userId = node.dataset.userId
        const name = (node.textContent || '').replace(/^@/, '')
        if (userId) {
          text += `<at user_id="${userId}">${name}</at>`
        } else {
          text += node.textContent || ''
        }
      } else if (node.tagName === 'BR') {
        text += '\n'
      } else {
        node.childNodes.forEach(walk)
      }
    }
  }
  editorRef.value.childNodes.forEach(walk)
  return text.replace(/\u00A0/g, ' ').trim()
}

const clearEditor = () => {
  if (editorRef.value) {
    editorRef.value.innerHTML = ''
  }
}

const handleEditorKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
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
        name: c.sender_name || userNameMap.value[c.sender_id] || c.chat_id,
        description: `${c.msg_count} 条消息`,
        sender_id: c.sender_id || '',
      }))
    // Deduplicate then fetch user info (avatar + name)
    deduplicatePrivateChats()
    privateChats.value.forEach(c => {
      const id = c.sender_id || c.chat_id
      if (id) fetchUserInfo(id)
    })
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
    fetchAvatars(messages.value)
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
      const data = JSON.parse(event.data)
      // Handle recall event
      if (data.recalled && data.message_id) {
        const target = messages.value.find(m => m.message_id === data.message_id)
        if (target) target.recalled = true
        return
      }
      const msg = data as Message
      if (!msg.message_id && data.id) msg.message_id = data.id
      messages.value.push(msg)
      if (msg.direction === 'in' && msg.sender_id) fetchAvatar(msg.sender_id)
      moveToTop(activeChatId.value)
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
  // If name is still an ID, fetch user info to resolve it
  if (activeChatName.value.startsWith('ou_') || activeChatName.value.startsWith('oc_')) {
    const id = item.sender_id || item.chat_id
    fetchUserInfo(id)
  }
  loadHistory()
  connectSSE()
}

// Context menu
const showContextMenu = (event: MouseEvent, msg: Message) => {
  contextMenu.value = { visible: true, x: event.clientX, y: event.clientY, msg }
}

const hideContextMenu = () => {
  contextMenu.value.visible = false
}

const handleReply = () => {
  const msg = contextMenu.value.msg
  if (msg) {
    if (!msg.message_id) {
      ElMessage.warning('该消息不支持回复')
      hideContextMenu()
      return
    }
    replyTo.value = msg
    nextTick(() => editorRef.value?.focus())
  }
  hideContextMenu()
}

const handleRecall = async () => {
  const msg = contextMenu.value.msg
  hideContextMenu()
  if (!msg?.message_id) return
  try {
    await deleteMessage(msg.message_id)
    msg.recalled = true
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '撤回失败')
  }
}

const handleReedit = (msg: Message) => {
  const text = parseContent(msg.content)
  if (editorRef.value) {
    editorRef.value.textContent = text
    // Place cursor at end
    const range = document.createRange()
    range.selectNodeContents(editorRef.value)
    range.collapse(false)
    const sel = window.getSelection()
    sel?.removeAllRanges()
    sel?.addRange(range)
    editorRef.value.focus()
  }
  // Remove the recalled message from list
  messages.value = messages.value.filter(m => m !== msg)
}

const handleStartPrivateChat = async () => {
  const msg = contextMenu.value.msg
  hideContextMenu()
  if (!msg?.sender_id) return

  // Resolve name: prefer sender_name > cached name > fetch from API
  let senderName = msg.sender_name || userNameMap.value[msg.sender_id] || ''
  if (!senderName || senderName.startsWith('ou_')) {
    try {
      const res = await getUserByOpenID(msg.sender_id)
      const user = res.data.data
      if (user?.name) {
        senderName = user.name
        userNameMap.value[msg.sender_id] = user.name
        avatarMap.value[msg.sender_id] = user.avatar || ''
      }
    } catch { /* fallback below */ }
  }
  if (!senderName) senderName = msg.sender_id

  // Check if already in private chats list
  const existing = privateChats.value.find(c => c.sender_id === msg.sender_id || c.chat_id === msg.sender_id)
  if (existing) {
    if (senderName && !existing.name.startsWith('ou_')) existing.name = existing.name
    else existing.name = senderName
    activeTab.value = 'private'
    switchChat(existing)
    return
  }
  // Add as temporary private chat entry and switch to it
  const tempChat: ChatItem = {
    chat_id: msg.sender_id,
    name: senderName,
    description: '新对话',
    sender_id: msg.sender_id,
  }
  privateChats.value.unshift(tempChat)
  activeTab.value = 'private'
  switchChat(tempChat)
}

const handleSend = async () => {
  const text = getEditorText()
  if (!text || !activeChatId.value) return

  sending.value = true
  try {
    const content = JSON.stringify({ text })

    let msgId = ''
    if (replyTo.value?.message_id) {
      // Reply to a specific message
      const res = await replyMessage({
        message_id: replyTo.value.message_id,
        msg_type: 'text',
        content,
      })
      msgId = res.data.message_id || ''
    } else {
      // Send new message (detect if open_id or chat_id)
      const idType = activeChatId.value.startsWith('ou_') ? 'open_id' : 'chat_id'
      const res = await sendMessage({
        receive_id: activeChatId.value,
        receive_id_type: idType,
        msg_type: 'text',
        content,
      })
      msgId = res.data.message_id || ''
    }

    clearEditor()
    replyTo.value = null
    messages.value.push({
      message_id: msgId,
      chat_id: activeChatId.value,
      sender_id: '',
      direction: 'out',
      msg_type: 'text',
      content,
      created_at: new Date().toISOString(),
    })
    moveToTop(activeChatId.value)
    scrollToBottom()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '发送失败')
  } finally {
    sending.value = false
  }
}

// Close context menu on click anywhere
const onDocumentClick = () => hideContextMenu()

onMounted(async () => {
  document.addEventListener('click', onDocumentClick)
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
    // Resolve name if still showing an ID
    if (activeChatName.value.startsWith('ou_')) {
      fetchUserInfo(activeChatName.value)
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

// Move chat to top when it receives new unread messages
watch(
  () => JSON.stringify(unreadMap),
  (newStr, oldStr) => {
    const newMap = JSON.parse(newStr) as Record<string, number>
    const oldMap = oldStr ? JSON.parse(oldStr) as Record<string, number> : {}
    for (const chatId in newMap) {
      if (newMap[chatId] > (oldMap[chatId] || 0)) {
        moveToTop(chatId)
      }
    }
  }
)

onUnmounted(() => {
  document.removeEventListener('click', onDocumentClick)
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
  padding: 10px 12px;
  cursor: pointer;
  border-bottom: 1px solid #fafafa;
  transition: background 0.15s;
}

.chat-item-body {
  display: flex;
  align-items: center;
  gap: 10px;
}

.chat-item-avatar {
  flex-shrink: 0;
}

.chat-item-info {
  flex: 1;
  overflow: hidden;
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
  align-items: flex-start;
  margin-bottom: 12px;
}

.message-in {
  justify-content: flex-start;
}

.message-out {
  justify-content: flex-end;
}

.message-avatar {
  flex-shrink: 0;
  margin-right: 8px;
  margin-top: 2px;
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

.message-text :deep(.mention) {
  color: #3370ff;
  font-weight: 500;
}

.message-out .message-text :deep(.mention) {
  color: #bbdefb;
}

.message-text :deep(.msg-image) {
  max-width: 300px;
  max-height: 300px;
  border-radius: 6px;
  cursor: pointer;
  display: block;
  margin: 4px 0;
}

.message-text :deep(.msg-sticker) {
  max-width: 120px;
  max-height: 120px;
  cursor: pointer;
  display: block;
  margin: 4px 0;
}

.reply-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #f5f7fa;
  border-top: 1px solid #e4e7ed;
  border-left: 3px solid #409eff;
}

.reply-info {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 13px;
}

.reply-label {
  color: #909399;
  margin-right: 4px;
}

.reply-name {
  color: #409eff;
  font-weight: 500;
  margin-right: 6px;
}

.reply-text {
  color: #606266;
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

.sender-clickable {
  cursor: pointer;
  transition: color 0.15s;
}

.sender-clickable:hover {
  color: #3370ff !important;
  opacity: 1 !important;
}

.sender-clickable:hover::before {
  content: '@';
}

.mention-editor {
  flex: 1;
  min-height: 32px;
  max-height: 120px;
  overflow-y: auto;
  padding: 4px 11px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-size: 14px;
  line-height: 24px;
  outline: none;
  word-break: break-word;
  transition: border-color 0.2s;
}

.mention-editor:focus {
  border-color: #409eff;
  box-shadow: 0 0 0 1px rgba(64, 158, 255, 0.2);
}

.mention-editor.disabled {
  background-color: #f5f7fa;
  cursor: not-allowed;
  color: #a8abb2;
}

.mention-editor:empty::before {
  content: attr(data-placeholder);
  color: #a8abb2;
  pointer-events: none;
}

.mention-editor :deep(.mention-tag) {
  display: inline;
  background: #e8f0fe;
  color: #3370ff;
  padding: 2px 4px;
  border-radius: 4px;
  font-weight: 500;
  margin: 0 1px;
  user-select: all;
}
</style>

<style>
/* Non-scoped styles for Teleported context menu */
.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  padding: 4px 0;
  z-index: 9999;
  min-width: 100px;
}

.context-menu-item {
  padding: 8px 16px;
  font-size: 13px;
  color: #303133;
  cursor: pointer;
  transition: background 0.15s;
}

.context-menu-item:hover {
  background: #f5f7fa;
  color: #409eff;
}

.message-recall {
  justify-content: center;
}

.recall-notice {
  font-size: 12px;
  color: #909399;
  padding: 4px 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.recall-content {
  color: #606266;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
  vertical-align: bottom;
}

.recall-resource {
  color: #409eff;
  font-weight: 500;
  cursor: pointer;
}

.recall-resource:hover {
  text-decoration: underline;
}

.recall-reedit {
  color: #409eff;
  cursor: pointer;
  margin-left: 6px;
}

.recall-reedit:hover {
  text-decoration: underline;
}

.context-menu-danger {
  color: #f56c6c;
}

.context-menu-danger:hover {
  background: #fef0f0;
  color: #f56c6c;
}
</style>
