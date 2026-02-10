<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px">
      <h2 style="margin: 0">群组管理</h2>
      <el-button type="primary" @click="handleSync" :loading="syncing">
        从飞书同步
      </el-button>
    </div>

    <el-table :data="groups" stripe v-loading="loading">
      <el-table-column prop="name" label="群名称">
        <template #default="{ row }">
          <router-link
            :to="{ path: '/chat/' + row.chat_id, query: { name: row.name } }"
            style="color: #409eff; text-decoration: none"
          >
            {{ row.name }}
          </router-link>
        </template>
      </el-table-column>
      <el-table-column prop="chat_id" label="群 ID" width="280" />
      <el-table-column prop="description" label="描述" />
      <el-table-column prop="member_count" label="成员数" width="100" />
      <el-table-column prop="synced_at" label="最后同步" width="180">
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
  description: string
  member_count: number
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

const formatTime = (t: string) => {
  if (!t) return '-'
  return new Date(t).toLocaleString()
}

onMounted(loadGroups)
</script>
