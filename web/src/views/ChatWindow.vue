<template>
  <div class="chat-layout">
    <!-- Left: Conversation list -->
    <div class="chat-sidebar">
      <div class="chat-sidebar-header">
        <el-input v-model="searchText" placeholder="搜索会话" size="small" clearable />
      </div>
      <el-tabs v-model="activeTab" class="chat-tabs" stretch>
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
      </el-tabs>
    </div>

    <!-- Right: Chat area -->
    <div class="chat-main">
      <template v-if="activeChatId">
        <!-- Chat header -->
        <div class="chat-header">
          <span
            class="chat-header-name"
            :class="{ 'chat-header-clickable': isGroupChat }"
            @click="isGroupChat && toggleMembersPanel()"
          >
            {{ activeChatName }}
            <el-icon v-if="isGroupChat" class="chat-header-arrow"><ArrowRight /></el-icon>
          </span>
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
                class="message-avatar message-avatar-clickable"
                @click="showUserProfile(msg.sender_id)"
              >
                {{ (msg.sender_name || msg.sender_id || '?').charAt(0) }}
              </el-avatar>
              <div class="message-bubble">
                <div class="message-meta">
                  <span v-if="msg.direction === 'in'" class="sender sender-clickable" @click="insertMention(msg)">{{ msg.sender_name || msg.sender_id || '用户' }}</span>
                  <span v-else class="sender">{{ botDisplayName }}</span>
                  <span class="time">{{ formatTime(msg.created_at) }}</span>
                </div>
                <div class="message-text" v-html="renderContent(msg.content, msg.msg_type, msg.message_id)"></div>
              </div>
              <el-avatar
                v-if="msg.direction === 'out'"
                :size="32"
                :src="botInfo.avatar_url || undefined"
                class="message-avatar-out"
              >
                {{ botDisplayName.charAt(0) }}
              </el-avatar>
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
            <span class="reply-name">{{ replyTo.direction === 'in' ? (replyTo.sender_name || replyTo.sender_id || '用户') : botDisplayName }}</span>
            <span class="reply-text">{{ parseContent(replyTo.content) }}</span>
          </div>
          <el-button text size="small" @click="replyTo = null" style="color: #909399">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>

        <!-- Attachment preview -->
        <div v-if="pendingFiles.length > 0" class="attachment-preview">
          <div v-for="(f, idx) in pendingFiles" :key="idx" class="attachment-item">
            <img v-if="f.previewUrl" :src="f.previewUrl" class="attachment-thumb" />
            <el-icon v-else :size="24"><Document /></el-icon>
            <span class="attachment-name">{{ f.file.name }}</span>
            <el-icon class="attachment-remove" @click="removePendingFile(idx)"><Close /></el-icon>
          </div>
        </div>

        <!-- Input -->
        <div class="input-area">
          <div class="input-actions">
            <el-tooltip content="发送图片" placement="top">
              <el-icon class="action-icon" @click="triggerImagePicker"><PictureFilled /></el-icon>
            </el-tooltip>
            <el-tooltip content="发送文件" placement="top">
              <el-icon class="action-icon" @click="triggerFilePicker"><FolderOpened /></el-icon>
            </el-tooltip>
          </div>
          <div
            ref="editorRef"
            class="mention-editor"
            :contenteditable="sending ? 'false' : 'true'"
            :class="{ disabled: sending }"
            :data-placeholder="replyTo ? '回复消息...' : '输入消息，按 Enter 发送...'"
            @keydown="handleEditorKeydown"
            @paste="handlePaste"
          ></div>
          <el-button type="primary" @click="handleSend" :loading="sending" style="margin-left: 8px">
            发送
          </el-button>
          <input ref="imageInputRef" type="file" accept="image/*" multiple style="display:none" @change="handleImagePicked" />
          <input ref="fileInputRef" type="file" multiple style="display:none" @change="handleFilePicked" />
        </div>
      </template>

      <!-- No chat selected -->
      <div v-else class="chat-placeholder">
        <el-icon :size="48" color="#c0c4cc"><ChatDotRound /></el-icon>
        <p style="color: #909399; margin-top: 12px">选择一个会话开始聊天</p>
      </div>
    </div>

    <!-- Right: Members panel (group chat only) -->
    <transition name="slide-right">
      <div v-if="showMembersPanel" class="members-panel">
        <div class="members-panel-header">
          <span>群成员 ({{ membersTotal || chatMembers.length }})</span>
          <div>
            <el-button text size="small" @click="loadChatMembers(true)" :loading="membersLoading">
              <el-icon><Refresh /></el-icon>
            </el-button>
            <el-button text size="small" @click="showMembersPanel = false">
              <el-icon><Close /></el-icon>
            </el-button>
          </div>
        </div>
        <div v-if="membersLoading && chatMembers.length === 0" class="members-loading">
          <el-text type="info">加载中...</el-text>
        </div>
        <div v-else ref="membersListRef" class="members-list" @scroll="onMembersScroll">
          <div v-for="member in chatMembers" :key="member.member_id" class="member-item member-item-clickable" @click="showUserProfile(member.member_id)">
            <el-avatar :size="32" :src="avatarMap[member.member_id] || undefined" class="member-avatar">
              {{ (member.name || '?').charAt(0) }}
            </el-avatar>
            <div class="member-info">
              <span class="member-name">{{ member.name || member.member_id }}</span>
            </div>
          </div>
          <div v-if="membersLoading && chatMembers.length > 0" class="members-loading">
            <el-text type="info" size="small">加载更多...</el-text>
          </div>
          <div v-if="chatMembers.length === 0" class="members-empty">
            <el-text type="info" size="small">暂无成员</el-text>
          </div>
        </div>
      </div>
    </transition>

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

    <!-- User profile dialog -->
    <el-dialog
      v-model="userProfileVisible"
      width="400px"
      :show-close="true"
      align-center
      class="user-profile-dialog"
    >
      <template #header>
        <span style="font-weight: 600">用户信息</span>
      </template>
      <div v-if="userProfileLoading" style="text-align: center; padding: 40px">
        <el-text type="info">加载中...</el-text>
      </div>
      <div v-else-if="userProfile" class="user-profile-card">
        <div class="profile-header">
          <el-avatar
            :size="64"
            :src="userProfile.avatar || undefined"
            class="profile-avatar profile-avatar-clickable"
            @click="userProfile.avatar && previewProfileAvatar(userProfile.avatar)"
          >
            {{ (userProfile.name || '?').charAt(0) }}
          </el-avatar>
          <div class="profile-name-area">
            <div class="profile-name">{{ userProfile.name }}</div>
            <div v-if="userProfile.en_name" class="profile-en-name">{{ userProfile.en_name }}</div>
            <div v-if="userProfile.job_title" class="profile-job-title">{{ userProfile.job_title }}</div>
          </div>
        </div>
        <el-tooltip v-if="userProfile.description" placement="bottom" :show-after="300" :disabled="userProfile.description.length <= 80" popper-class="profile-description-tooltip">
          <template #content>
            <div style="max-width: 340px; white-space: pre-wrap; word-break: break-word;">{{ userProfile.description }}</div>
          </template>
          <div class="profile-description">{{ userProfile.description }}</div>
        </el-tooltip>
        <el-divider style="margin: 16px 0" />
        <div class="profile-info-list">
          <div v-if="userProfile.department_names" class="profile-info-item">
            <span class="profile-info-label">部门</span>
            <span class="profile-info-value">{{ formatJsonArray(userProfile.department_names) }}</span>
          </div>
          <div v-if="userProfile.city" class="profile-info-item">
            <span class="profile-info-label">城市</span>
            <span class="profile-info-value">{{ userProfile.city }}</span>
          </div>
          <div v-if="userProfile.employee_no" class="profile-info-item">
            <span class="profile-info-label">工号</span>
            <span class="profile-info-value">{{ userProfile.employee_no }}</span>
          </div>
          <div v-if="userProfile.email" class="profile-info-item">
            <span class="profile-info-label">邮箱</span>
            <span class="profile-info-value">{{ userProfile.email }}</span>
          </div>
          <div v-if="userProfile.work_station" class="profile-info-item">
            <span class="profile-info-label">工位</span>
            <span class="profile-info-value">{{ userProfile.work_station }}</span>
          </div>
          <div v-if="userProfile.gender" class="profile-info-item">
            <span class="profile-info-label">性别</span>
            <span class="profile-info-value">{{ formatGender(userProfile.gender) }}</span>
          </div>
          <div v-if="userProfile.join_time" class="profile-info-item">
            <span class="profile-info-label">入职时间</span>
            <span class="profile-info-value">{{ formatDate(userProfile.join_time) }}</span>
          </div>
          <div v-if="userProfile.msg_count" class="profile-info-item">
            <span class="profile-info-label">消息数</span>
            <span class="profile-info-value">{{ userProfile.msg_count }}</span>
          </div>
          <div v-if="userProfile.last_seen" class="profile-info-item">
            <span class="profile-info-label">最后活跃</span>
            <span class="profile-info-value">{{ formatDate(userProfile.last_seen) }}</span>
          </div>
        </div>
        <el-divider style="margin: 16px 0" />
        <div class="profile-actions">
          <el-button type="primary" @click="handleProfilePrivateChat">发起私聊</el-button>
        </div>
      </div>
    </el-dialog>

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
import { ChatDotRound, Close, ArrowRight, Refresh, PictureFilled, FolderOpened, Document } from '@element-plus/icons-vue'
import { getChats, getConversations, getMessageLogs, sendMessage, replyMessage, deleteMessage, getToken, getUserByOpenID, getChatMembers, uploadImage, uploadFile } from '../api/client'
import { ElMessage } from 'element-plus'

const unreadMap = inject<Record<string, number>>('unreadMap', {})
const clearChatUnread = inject<(chatId: string) => void>('clearChatUnread', () => {})
const botInfo = inject<{ name: string; open_id: string; avatar_url: string }>('botInfo', { name: '', open_id: '', avatar_url: '' })
const botDisplayName = computed(() => botInfo.name || '机器人')

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

interface UserProfile {
  open_id: string
  name: string
  en_name: string
  avatar: string
  description: string
  email: string
  city: string
  job_title: string
  department_ids: string
  department_names: string
  work_station: string
  employee_no: string
  gender: number
  join_time: number
  msg_count: number
  first_seen: string
  last_seen: string
}

const route = useRoute()
const router = useRouter()

const activeTab = ref('private')
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
const imageInputRef = ref<HTMLInputElement>()
const fileInputRef = ref<HTMLInputElement>()
const replyTo = ref<Message | null>(null)

// Pending attachments (images/files)
interface PendingFile {
  file: File
  type: 'image' | 'file'
  previewUrl?: string
}
const pendingFiles = ref<PendingFile[]>([])
const contextMenu = ref({ visible: false, x: 0, y: 0, msg: null as Message | null })

// Members panel
const showMembersPanel = ref(false)
const chatMembers = ref<{ member_id: string; name: string }[]>([])
const membersLoading = ref(false)
const membersTotal = ref(0)
const membersPageToken = ref('')
const membersHasMore = ref(false)
const membersListRef = ref<HTMLElement>()

const isGroupChat = computed(() => {
  return groups.value.some(g => g.chat_id === activeChatId.value)
})

const toggleMembersPanel = async () => {
  showMembersPanel.value = !showMembersPanel.value
  if (showMembersPanel.value && chatMembers.value.length === 0) {
    await loadChatMembers(true)
  }
}

const loadChatMembers = async (reset: boolean) => {
  if (!activeChatId.value || membersLoading.value) return
  if (reset) {
    chatMembers.value = []
    membersPageToken.value = ''
    membersHasMore.value = false
    membersTotal.value = 0
  }
  membersLoading.value = true
  try {
    const res = await getChatMembers(activeChatId.value, {
      page_token: membersPageToken.value || undefined,
      page_size: 50,
    })
    const newItems = res.data.data || []
    chatMembers.value = reset ? newItems : [...chatMembers.value, ...newItems]
    membersPageToken.value = res.data.page_token || ''
    membersHasMore.value = res.data.has_more || false
    membersTotal.value = res.data.total || chatMembers.value.length
    // Fetch avatars for new members
    newItems.forEach((m: { member_id: string }) => {
      if (m.member_id) fetchUserInfo(m.member_id)
    })
  } catch (e) {
    console.error('加载群成员失败', e)
  } finally {
    membersLoading.value = false
  }
}

const onMembersScroll = () => {
  if (!membersHasMore.value || membersLoading.value) return
  const el = membersListRef.value
  if (!el) return
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 50) {
    loadChatMembers(false)
  }
}

// User profile dialog
const userProfileVisible = ref(false)
const userProfileLoading = ref(false)
const userProfile = ref<UserProfile | null>(null)

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

const showUserProfile = async (openId: string) => {
  if (!openId) return
  userProfileVisible.value = true
  userProfileLoading.value = true
  userProfile.value = null
  try {
    const res = await getUserByOpenID(openId)
    const user = res.data.data
    if (user) {
      userProfile.value = {
        open_id: user.open_id || openId,
        name: user.name || '',
        en_name: user.en_name || '',
        avatar: user.avatar || '',
        description: user.description || '',
        email: user.email || '',
        city: user.city || '',
        job_title: user.job_title || '',
        department_ids: user.department_ids || '',
        department_names: user.department_names || '',
        work_station: user.work_station || '',
        employee_no: user.employee_no || '',
        gender: user.gender || 0,
        join_time: user.join_time || 0,
        msg_count: user.msg_count || 0,
        first_seen: user.first_seen || '',
        last_seen: user.last_seen || '',
      }
    }
  } catch {
    ElMessage.error('获取用户信息失败')
    userProfileVisible.value = false
  } finally {
    userProfileLoading.value = false
  }
}

const previewProfileAvatar = (avatarUrl: string) => {
  previewList.value = [avatarUrl]
  previewVisible.value = true
}

const handleProfilePrivateChat = () => {
  if (!userProfile.value) return
  const openId = userProfile.value.open_id
  const name = userProfile.value.name || openId
  userProfileVisible.value = false

  // Check if already in private chats list
  const existing = privateChats.value.find(c => c.sender_id === openId || c.chat_id === openId)
  if (existing) {
    activeTab.value = 'private'
    switchChat(existing)
    return
  }
  // Add as temporary private chat entry and switch to it
  const tempChat: ChatItem = {
    chat_id: openId,
    name,
    description: '新对话',
    sender_id: openId,
  }
  privateChats.value.unshift(tempChat)
  activeTab.value = 'private'
  switchChat(tempChat)
}

const formatJsonArray = (s: string): string => {
  try {
    const arr = JSON.parse(s)
    if (Array.isArray(arr)) return arr.join(' ')
  } catch {}
  return s
}

const formatGender = (g: number) => g === 1 ? '男' : g === 2 ? '女' : '未知'

const formatDate = (t: string | number) => {
  if (!t) return '-'
  const d = typeof t === 'number' ? new Date(t * 1000) : new Date(t)
  return d.toLocaleDateString()
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

  // File message: {"file_key":"xxx","file_name":"xxx"}
  if (msgType === 'file') {
    try {
      const parsed = JSON.parse(content)
      if (parsed.file_key) {
        const fileName = parsed.file_name || '文件'
        if (messageId) {
          const url = resourceUrl(messageId, parsed.file_key, 'file') + `&filename=${encodeURIComponent(fileName)}`
          return `<a class="msg-file" href="${url}" download="${escapeHtml(fileName)}">📎 ${escapeHtml(fileName)}</a>`
        }
        return `📎 ${escapeHtml(fileName)}`
      }
    } catch {}
    return '[文件]'
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

const handlePaste = (e: ClipboardEvent) => {
  const items = e.clipboardData?.items
  if (!items) return

  const imageItems: DataTransferItem[] = []
  for (const item of items) {
    if (item.type.startsWith('image/')) {
      imageItems.push(item)
    }
  }

  if (imageItems.length > 0) {
    e.preventDefault()
    for (const item of imageItems) {
      const file = item.getAsFile()
      if (file) {
        addPendingFile(file, 'image')
      }
    }
  }
}

const addPendingFile = (file: File, type: 'image' | 'file') => {
  const pf: PendingFile = { file, type }
  if (type === 'image') {
    pf.previewUrl = URL.createObjectURL(file)
  }
  pendingFiles.value.push(pf)
}

const removePendingFile = (idx: number) => {
  const pf = pendingFiles.value[idx]
  if (pf.previewUrl) URL.revokeObjectURL(pf.previewUrl)
  pendingFiles.value.splice(idx, 1)
}

const clearPendingFiles = () => {
  pendingFiles.value.forEach(pf => {
    if (pf.previewUrl) URL.revokeObjectURL(pf.previewUrl)
  })
  pendingFiles.value = []
}

const triggerImagePicker = () => {
  imageInputRef.value?.click()
}

const triggerFilePicker = () => {
  fileInputRef.value?.click()
}

const handleImagePicked = (e: Event) => {
  const input = e.target as HTMLInputElement
  if (input.files) {
    for (const file of input.files) {
      addPendingFile(file, 'image')
    }
  }
  input.value = ''
}

const handleFilePicked = (e: Event) => {
  const input = e.target as HTMLInputElement
  if (input.files) {
    for (const file of input.files) {
      addPendingFile(file, 'file')
    }
  }
  input.value = ''
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
  showMembersPanel.value = false
  chatMembers.value = []
  membersPageToken.value = ''
  membersHasMore.value = false
  membersTotal.value = 0
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

const sendOneMessage = async (msgType: string, content: string): Promise<string> => {
  const idType = activeChatId.value.startsWith('ou_') ? 'open_id' : 'chat_id'
  if (replyTo.value?.message_id) {
    const res = await replyMessage({
      message_id: replyTo.value.message_id,
      msg_type: msgType,
      content,
    })
    return res.data.message_id || ''
  } else {
    const res = await sendMessage({
      receive_id: activeChatId.value,
      receive_id_type: idType,
      msg_type: msgType,
      content,
    })
    return res.data.message_id || ''
  }
}

const handleSend = async () => {
  const text = getEditorText()
  const hasFiles = pendingFiles.value.length > 0
  if ((!text && !hasFiles) || !activeChatId.value) return

  sending.value = true
  try {
    // Send text message first (if any)
    if (text) {
      const content = JSON.stringify({ text })
      const msgId = await sendOneMessage('text', content)
      messages.value.push({
        message_id: msgId,
        chat_id: activeChatId.value,
        sender_id: '',
        direction: 'out',
        msg_type: 'text',
        content,
        created_at: new Date().toISOString(),
      })
    }

    // Send each attachment
    for (const pf of pendingFiles.value) {
      if (pf.type === 'image') {
        const uploadRes = await uploadImage(pf.file)
        const imageKey = uploadRes.data.image_key
        const content = JSON.stringify({ image_key: imageKey })
        const msgId = await sendOneMessage('image', content)
        messages.value.push({
          message_id: msgId,
          chat_id: activeChatId.value,
          sender_id: '',
          direction: 'out',
          msg_type: 'image',
          content,
          created_at: new Date().toISOString(),
        })
      } else {
        const uploadRes = await uploadFile(pf.file)
        const fileKey = uploadRes.data.file_key
        const content = JSON.stringify({ file_key: fileKey, file_name: pf.file.name })
        const msgId = await sendOneMessage('file', content)
        messages.value.push({
          message_id: msgId,
          chat_id: activeChatId.value,
          sender_id: '',
          direction: 'out',
          msg_type: 'file',
          content,
          created_at: new Date().toISOString(),
        })
      }
    }

    clearEditor()
    clearPendingFiles()
    replyTo.value = null
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
    if (match) {
      activeChatName.value = match.name || chatId
    }
    // Auto-select correct tab
    if (!groups.value.find((g) => g.chat_id === chatId)) {
      activeTab.value = 'private'
      // If not in private chats list either, add a temporary entry
      if (!privateChats.value.find((c) => c.chat_id === chatId)) {
        const name = (route.query.name as string) || chatId
        privateChats.value.unshift({
          chat_id: chatId,
          name,
          description: '新对话',
          sender_id: chatId,
        })
      }
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
      // If not in private chats list either, add a temporary entry
      if (!privateChats.value.find((c) => c.chat_id === chatId)) {
        const name = (route.query.name as string) || chatId
        privateChats.value.unshift({
          chat_id: chatId,
          name,
          description: '新对话',
          sender_id: chatId,
        })
      }
    } else {
      activeTab.value = 'group'
    }
    messages.value = []
    clearChatUnread(chatId)
    // Resolve name if still showing an ID
    if (activeChatName.value.startsWith('ou_')) {
      fetchUserInfo(activeChatName.value)
    }
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
  min-width: 0;
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
  display: flex;
  align-items: center;
  gap: 4px;
}

.chat-header-clickable {
  cursor: pointer;
  transition: color 0.15s;
}

.chat-header-clickable:hover {
  color: #409eff;
}

.chat-header-arrow {
  font-size: 14px;
  color: #909399;
  transition: color 0.15s;
}

.chat-header-clickable:hover .chat-header-arrow {
  color: #409eff;
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

.message-avatar-out {
  flex-shrink: 0;
  margin-left: 8px;
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

.message-out .message-meta {
  opacity: 0.9;
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

.message-text :deep(.msg-file) {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: #f4f4f5;
  border-radius: 4px;
  color: #409eff;
  text-decoration: none;
  font-size: 13px;
  transition: background 0.2s;
}

.message-text :deep(.msg-file:hover) {
  background: #ecf5ff;
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

.attachment-preview {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 8px 16px 0;
  border-top: 1px solid #e4e7ed;
}

.attachment-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: #f4f4f5;
  border-radius: 4px;
  font-size: 12px;
  color: #606266;
}

.attachment-thumb {
  width: 40px;
  height: 40px;
  object-fit: cover;
  border-radius: 4px;
}

.attachment-name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.attachment-remove {
  cursor: pointer;
  color: #909399;
  transition: color 0.2s;
}

.attachment-remove:hover {
  color: #f56c6c;
}

.input-area {
  display: flex;
  align-items: flex-end;
  padding: 12px 16px;
  border-top: 1px solid #e4e7ed;
}

.input-area .attachment-preview + & {
  border-top: none;
}

.input-actions {
  display: flex;
  gap: 4px;
  margin-right: 8px;
  padding-bottom: 4px;
}

.action-icon {
  font-size: 20px;
  color: #909399;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s;
}

.action-icon:hover {
  color: #409eff;
  background: #ecf5ff;
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

/* Members panel */
.members-panel {
  width: 240px;
  min-width: 240px;
  background: #fff;
  border-left: 1px solid #e4e7ed;
  border-radius: 0 8px 8px 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.members-panel-header {
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.members-loading {
  text-align: center;
  padding: 20px;
}

.members-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.member-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 16px;
  transition: background 0.15s;
}

.member-item:hover {
  background: #f5f7fa;
}

.member-avatar {
  flex-shrink: 0;
}

.member-info {
  flex: 1;
  overflow: hidden;
}

.member-name {
  font-size: 13px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
}

.members-empty {
  text-align: center;
  padding: 20px;
}

/* Slide transition */
.slide-right-enter-active,
.slide-right-leave-active {
  transition: all 0.2s ease;
}

.slide-right-enter-from,
.slide-right-leave-to {
  width: 0;
  min-width: 0;
  opacity: 0;
}

/* Clickable avatar */
.message-avatar-clickable {
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
}

.message-avatar-clickable:hover {
  transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.member-item-clickable {
  cursor: pointer;
}

/* User profile card */
.user-profile-card {
  padding: 0 4px;
}

.profile-header {
  display: flex;
  align-items: center;
  gap: 16px;
}

.profile-avatar {
  flex-shrink: 0;
}

.profile-description {
  font-size: 13px;
  color: #606266;
  margin-top: 12px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 5;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
  word-break: break-word;
}

.profile-avatar-clickable {
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
}

.profile-avatar-clickable:hover {
  transform: scale(1.1);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.2);
}

.profile-name-area {
  flex: 1;
  overflow: hidden;
}

.profile-name {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.profile-en-name {
  font-size: 13px;
  color: #909399;
  margin-top: 2px;
}

.profile-job-title {
  font-size: 13px;
  color: #606266;
  margin-top: 4px;
}

.profile-info-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.profile-info-item {
  display: flex;
  align-items: center;
  font-size: 13px;
}

.profile-info-label {
  width: 72px;
  flex-shrink: 0;
  color: #909399;
}

.profile-info-value {
  color: #303133;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-actions {
  display: flex;
  justify-content: center;
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
