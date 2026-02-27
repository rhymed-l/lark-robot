<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px">
      <h2 style="margin: 0">群组管理</h2>
      <el-button type="primary" @click="handleSync" :loading="syncing">
        从飞书同步
      </el-button>
    </div>

    <el-table :data="groups" stripe v-loading="loading">
      <el-table-column label="头像" width="70">
        <template #default="{ row }">
          <el-avatar v-if="row.avatar" :src="row.avatar" :size="36" />
          <el-avatar v-else :size="36">{{ (row.name || '?').charAt(0) }}</el-avatar>
        </template>
      </el-table-column>
      <el-table-column prop="name" label="群名称" min-width="150">
        <template #default="{ row }">
          <router-link
            :to="{ path: '/chat/' + row.chat_id, query: { name: row.name } }"
            style="color: #409eff; text-decoration: none"
          >
            {{ row.name }}
          </router-link>
        </template>
      </el-table-column>
      <el-table-column label="群模式" width="80">
        <template #default="{ row }">
          {{ chatModeLabel(row.chat_mode) }}
        </template>
      </el-table-column>
      <el-table-column label="群类型" width="80">
        <template #default="{ row }">
          {{ chatTypeLabel(row.chat_type) }}
        </template>
      </el-table-column>
      <el-table-column label="群标签" width="80">
        <template #default="{ row }">
          {{ chatTagLabel(row.chat_tag) }}
        </template>
      </el-table-column>
      <el-table-column prop="member_count" label="成员数" width="80" />
      <el-table-column prop="bot_count" label="机器人" width="80" />
      <el-table-column prop="synced_at" label="最后同步" width="170">
        <template #default="{ row }">
          {{ formatTime(row.synced_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-popconfirm
            title="确定要退出该群吗？"
            @confirm="handleLeave(row.chat_id)"
          >
            <template #reference>
              <el-button type="danger" size="small">退出</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-if="total > 0"
      style="margin-top: 20px; justify-content: flex-end"
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
import { getChats, syncChats, leaveChat } from '../api/client'
import { ElMessage } from 'element-plus'

interface Group {
  id: number
  chat_id: string
  name: string
  avatar: string
  chat_mode: string
  chat_type: string
  chat_tag: string
  member_count: number
  bot_count: number
  synced_at: string
}

const groups = ref<Group[]>([])
const loading = ref(false)
const syncing = ref(false)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

const loadGroups = async () => {
  loading.value = true
  try {
    const res = await getChats({ page: page.value, page_size: pageSize.value })
    groups.value = res.data.data || []
    total.value = res.data.total || 0
  } catch (e) {
    console.error('加载群列表失败', e)
  } finally {
    loading.value = false
  }
}

const handlePageChange = (p: number) => {
  page.value = p
  loadGroups()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  page.value = 1
  loadGroups()
}

const handleSync = async () => {
  syncing.value = true
  try {
    await syncChats()
    ElMessage.success('同步完成')
    page.value = 1
    await loadGroups()
  } catch (e) {
    ElMessage.error('同步失败')
  } finally {
    syncing.value = false
  }
}

const handleLeave = async (chatId: string) => {
  try {
    await leaveChat(chatId)
    ElMessage.success('已退出群组')
    await loadGroups()
  } catch (e) {
    ElMessage.error('退出群组失败')
  }
}

const chatModeLabel = (mode: string) => {
  const map: Record<string, string> = { group: '群组', topic: '话题', p2p: '单聊' }
  return map[mode] || mode || '-'
}

const chatTypeLabel = (type: string) => {
  const map: Record<string, string> = { private: '私有群', public: '公开群' }
  return map[type] || type || '-'
}

const chatTagLabel = (tag: string) => {
  const map: Record<string, string> = {
    inner: '内部群', tenant: '公司群', department: '部门群',
    edu: '教育群', meeting: '会议群', customer_service: '客服群',
  }
  return map[tag] || tag || '-'
}

const formatTime = (t: string) => {
  if (!t) return '-'
  return new Date(t).toLocaleString()
}

onMounted(loadGroups)
</script>
