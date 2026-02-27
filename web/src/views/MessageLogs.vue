<template>
  <div class="page-container">
    <h2 style="flex-shrink: 0">消息日志</h2>

    <el-row :gutter="10" style="margin-bottom: 20px; flex-shrink: 0">
      <el-col :span="5">
        <el-input v-model="filters.chat_id" placeholder="按会话 ID 筛选" clearable @clear="loadLogs" />
      </el-col>
      <el-col :span="3">
        <el-select v-model="filters.chat_type" placeholder="会话类型" clearable @change="loadLogs">
          <el-option label="群聊" value="group" />
          <el-option label="私聊" value="p2p" />
        </el-select>
      </el-col>
      <el-col :span="3">
        <el-select v-model="filters.direction" placeholder="方向" clearable @change="loadLogs">
          <el-option label="接收" value="in" />
          <el-option label="发送" value="out" />
        </el-select>
      </el-col>
      <el-col :span="3">
        <el-select v-model="filters.source" placeholder="来源" clearable @change="loadLogs">
          <el-option label="事件" value="event" />
          <el-option label="定时任务" value="scheduled" />
          <el-option label="手动" value="manual" />
        </el-select>
      </el-col>
      <el-col :span="3">
        <el-button type="primary" @click="loadLogs">搜索</el-button>
      </el-col>
    </el-row>

    <div style="flex: 1; min-height: 0; overflow: hidden">
    <el-table :data="logs" stripe v-loading="loading" height="100%">
      <el-table-column prop="direction" label="方向" width="70">
        <template #default="{ row }">
          <el-tag :type="row.direction === 'in' ? 'success' : 'primary'" size="small">
            {{ row.direction === 'in' ? '收' : '发' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="chat_type" label="类型" width="70">
        <template #default="{ row }">
          <el-tag v-if="row.chat_type === 'p2p'" type="warning" size="small">私聊</el-tag>
          <el-tag v-else-if="row.chat_type === 'group'" size="small">群聊</el-tag>
          <el-tag v-else type="info" size="small">-</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="会话" width="160">
        <template #default="{ row }">
          <el-tooltip :content="row.chat_id" placement="top">
            <span class="name-cell">{{ chatName(row) }}</span>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column label="发送者" width="130">
        <template #default="{ row }">
          <el-tooltip v-if="row.sender_id" :content="row.sender_id" placement="top">
            <span class="name-cell">{{ row.sender_name || row.sender_id }}</span>
          </el-tooltip>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column prop="content" label="内容" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-html="renderContent(row.content, row.msg_type, row.message_id)"></span>
        </template>
      </el-table-column>
      <el-table-column prop="handled_by" label="处理器" width="120" />
      <el-table-column prop="source" label="来源" width="80">
        <template #default="{ row }">
          {{ sourceLabel(row.source) }}
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="时间" width="170">
        <template #default="{ row }">
          {{ formatTime(row.created_at) }}
        </template>
      </el-table-column>
    </el-table>
    </div>

    <el-pagination
      v-if="total > 0"
      style="margin-top: 12px; justify-content: flex-end; flex-shrink: 0"
      :current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="total, sizes, prev, pager, next"
      :page-sizes="[10, 20, 50]"
      @current-change="handlePageChange"
      @size-change="handleSizeChange"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMessageLogs, getChats, getConversations, getToken } from '../api/client'

interface Log {
  id: number
  message_id: string
  chat_id: string
  chat_type: string
  sender_id: string
  sender_name: string
  direction: string
  msg_type: string
  content: string
  handled_by: string
  source: string
  created_at: string
}

const logs = ref<Log[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

// Chat name lookup: chatId -> group name
const chatNameMap = ref<Record<string, string>>({})

const filters = ref({
  chat_id: '',
  chat_type: '',
  direction: '',
  source: '',
})

const sourceLabel = (s: string) => {
  const map: Record<string, string> = { event: '事件', scheduled: '定时', manual: '手动' }
  return map[s] || s
}

const escapeHtml = (text: string): string => {
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

const highlightMentions = (text: string): string => {
  return text.split(/(@\[[^\]]+\])/).map(part => {
    const m = part.match(/^@\[([^\]]+)\]$/)
    if (m) return `<span style="color: #3370ff; font-weight: 500">@${escapeHtml(m[1])}</span>`
    return escapeHtml(part).replace(/@(\S+)/g, '<span style="color: #3370ff; font-weight: 500">@$1</span>')
  }).join('')
}

const resourceUrl = (messageId: string, key: string, type: string = 'image'): string => {
  return `/api/images/${encodeURIComponent(messageId)}/${encodeURIComponent(key)}?type=${type}&token=${getToken()}`
}

const imageUrl = (messageId: string, imageKey: string): string => {
  return resourceUrl(messageId, imageKey, 'image')
}

const renderContent = (content: string, msgType: string, messageId?: string): string => {
  if (msgType === 'image' && messageId) {
    try {
      const parsed = JSON.parse(content)
      if (parsed.image_key) {
        return `<img style="max-width:120px;max-height:120px;border-radius:4px" src="${imageUrl(messageId, parsed.image_key)}" alt="图片" />`
      }
    } catch {}
    return '[图片]'
  }

  if (msgType === 'sticker' && messageId) {
    try {
      const parsed = JSON.parse(content)
      if (parsed.file_key) {
        return `<img style="max-width:80px;max-height:80px" src="${resourceUrl(messageId, parsed.file_key, 'file')}" alt="表情" />`
      }
    } catch {}
    return '[表情]'
  }

  try {
    const parsed = JSON.parse(content)
    if (parsed.text) {
      const text = parsed.text.replace(/<at user_id="[^"]*">([^<]*)<\/at>/g, '@[$1]')
      return highlightMentions(text)
    }
    if (parsed.content && Array.isArray(parsed.content)) {
      const parts: string[] = []
      for (const line of parsed.content) {
        if (!Array.isArray(line)) continue
        for (const elem of line) {
          if (elem.tag === 'text' && elem.text) parts.push(escapeHtml(elem.text))
          else if (elem.tag === 'at' && elem.user_name) parts.push(`<span style="color: #3370ff; font-weight: 500">@${escapeHtml(elem.user_name)}</span>`)
          else if (elem.tag === 'img' && elem.image_key && messageId) parts.push(`<img style="max-width:120px;max-height:120px;border-radius:4px" src="${imageUrl(messageId, elem.image_key)}" alt="图片" />`)
        }
      }
      return parts.join('')
    }
    return escapeHtml(content)
  } catch {
    return escapeHtml(content)
  }
}

const chatName = (row: Log): string => {
  // For group chats, show group name from lookup
  if (chatNameMap.value[row.chat_id]) {
    return chatNameMap.value[row.chat_id]
  }
  // For p2p chats, show sender name
  if (row.chat_type === 'p2p' && row.sender_name) {
    return row.sender_name
  }
  return row.chat_id
}

const loadChatNames = async () => {
  const map: Record<string, string> = {}
  try {
    // Load group names
    const groupRes = await getChats({ page: 1, page_size: 100 })
    for (const g of (groupRes.data.data || [])) {
      if (g.name) map[g.chat_id] = g.name
    }
    // Load p2p conversation names
    const convRes = await getConversations()
    for (const c of (convRes.data.data || [])) {
      if (c.chat_type === 'p2p' && c.sender_name && !map[c.chat_id]) {
        map[c.chat_id] = c.sender_name
      }
    }
  } catch {
    // ignore
  }
  chatNameMap.value = map
}

const loadLogs = async () => {
  loading.value = true
  try {
    const res = await getMessageLogs({
      page: page.value,
      page_size: pageSize.value,
      ...filters.value,
    })
    logs.value = res.data.data || []
    total.value = res.data.total || 0
  } catch (e) {
    console.error('加载日志失败', e)
  } finally {
    loading.value = false
  }
}

const handlePageChange = (p: number) => {
  page.value = p
  loadLogs()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  page.value = 1
  loadLogs()
}

const formatTime = (t: string) => {
  if (!t) return '-'
  return new Date(t).toLocaleString()
}

onMounted(async () => {
  await loadChatNames()
  loadLogs()
})
</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 40px);
}
.name-cell {
  cursor: default;
  border-bottom: 1px dashed #c0c4cc;
}
</style>
