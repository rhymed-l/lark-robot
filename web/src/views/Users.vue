<template>
  <div class="page-container">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; flex-shrink: 0">
      <h2 style="margin: 0">用户管理</h2>
      <div style="display: flex; gap: 12px; align-items: center">
        <el-input
          v-model="keyword"
          placeholder="搜索用户名/工号/邮箱"
          clearable
          style="width: 260px"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        >
          <template #append>
            <el-button @click="handleSearch">搜索</el-button>
          </template>
        </el-input>
        <el-button type="primary" @click="handleSync" :loading="syncing">
          从飞书同步
        </el-button>
      </div>
    </div>

    <div style="flex: 1; min-height: 0; overflow: hidden">
      <el-table :data="users" stripe v-loading="loading" height="100%" @sort-change="handleSortChange">
        <el-table-column label="头像" width="70">
          <template #default="{ row }">
            <el-avatar v-if="row.avatar" :src="row.avatar" :size="36" />
            <el-avatar v-else :size="36">{{ (row.name || '?').charAt(0) }}</el-avatar>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="姓名" min-width="100" sortable="custom">
          <template #default="{ row }">
            <router-link
              :to="{ path: '/chat/' + row.open_id, query: { name: row.name } }"
              style="color: #409eff; text-decoration: none"
            >
              {{ row.name || row.open_id }}
            </router-link>
          </template>
        </el-table-column>
        <el-table-column prop="en_name" label="英文名" width="120" sortable="custom" />
        <el-table-column prop="employee_no" label="工号" width="80" sortable="custom" />
        <el-table-column prop="job_title" label="职务" min-width="120" sortable="custom" />
        <el-table-column prop="email" label="邮箱" min-width="180" sortable="custom">
          <template #default="{ row }">
            <span style="font-size: 12px">{{ row.email || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="work_station" label="工位" width="100" sortable="custom" />
        <el-table-column prop="gender" label="性别" width="60">
          <template #default="{ row }">
            {{ row.gender === 1 ? '男' : row.gender === 2 ? '女' : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="msg_count" label="消息数" width="80" sortable="custom" />
        <el-table-column prop="join_time" label="入职时间" width="120" sortable="custom">
          <template #default="{ row }">
            {{ formatJoinTime(row.join_time) }}
          </template>
        </el-table-column>
        <el-table-column prop="last_seen" label="最后活跃" width="170" sortable="custom">
          <template #default="{ row }">
            {{ formatTime(row.last_seen) }}
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
import { getUsers, syncUsers } from '../api/client'
import { ElMessage } from 'element-plus'

interface User {
  id: number
  open_id: string
  union_id: string
  user_id: string
  name: string
  en_name: string
  avatar: string
  email: string
  job_title: string
  work_station: string
  employee_no: string
  gender: number
  leader_user_id: string
  join_time: number
  first_seen: string
  last_seen: string
  msg_count: number
}

const users = ref<User[]>([])
const loading = ref(false)
const syncing = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const sortBy = ref('')
const sortDir = ref('')

const loadUsers = async () => {
  loading.value = true
  try {
    const res = await getUsers({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value || undefined,
      sort_by: sortBy.value || undefined,
      sort_dir: sortDir.value || undefined,
    })
    users.value = res.data.data || []
    total.value = res.data.total || 0
  } catch (e) {
    console.error('加载用户列表失败', e)
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  page.value = 1
  loadUsers()
}

const handleSync = async () => {
  syncing.value = true
  try {
    const res = await syncUsers()
    ElMessage.success(`同步完成，已更新 ${res.data.synced} 个用户`)
    page.value = 1
    await loadUsers()
  } catch (e) {
    ElMessage.error('同步失败')
  } finally {
    syncing.value = false
  }
}

const handlePageChange = (p: number) => {
  page.value = p
  loadUsers()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  page.value = 1
  loadUsers()
}

const handleSortChange = ({ prop, order }: { prop: string; order: string | null }) => {
  sortBy.value = order ? prop : ''
  sortDir.value = order === 'ascending' ? 'asc' : order === 'descending' ? 'desc' : ''
  page.value = 1
  loadUsers()
}

const formatTime = (t: string) => {
  if (!t) return '-'
  return new Date(t).toLocaleString()
}

const formatJoinTime = (t: number) => {
  if (!t) return '-'
  return new Date(t * 1000).toLocaleDateString()
}

onMounted(loadUsers)
</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 40px);
}
.id-cell {
  font-family: monospace;
  font-size: 12px;
  color: #909399;
}
</style>
