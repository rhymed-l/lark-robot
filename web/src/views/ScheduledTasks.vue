<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px">
      <h2 style="margin: 0">定时任务</h2>
      <el-button type="primary" @click="showDialog()">添加任务</el-button>
    </div>

    <el-table :data="tasks" stripe v-loading="loading">
      <el-table-column prop="name" label="任务名称" />
      <el-table-column prop="cron_expr" label="Cron 表达式" width="160" />
      <el-table-column prop="chat_id" label="目标群 ID" width="200" />
      <el-table-column prop="msg_type" label="类型" width="80" />
      <el-table-column prop="enabled" label="状态" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" @change="handleToggle(row.id)" />
        </template>
      </el-table-column>
      <el-table-column prop="last_run_at" label="上次执行" width="180">
        <template #default="{ row }">
          {{ formatTime(row.last_run_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="success" @click="handleRun(row.id)">立即执行</el-button>
          <el-button size="small" @click="showDialog(row)">编辑</el-button>
          <el-popconfirm title="确定删除该任务吗？" @confirm="handleDelete(row.id)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editingTask ? '编辑任务' : '添加任务'" width="550px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="任务名称" required>
          <el-input v-model="form.name" placeholder="任务名称" />
        </el-form-item>
        <el-form-item label="Cron 表达式" required>
          <el-input v-model="form.cron_expr" placeholder="例如: 0 0 9 * * 1-5（6位含秒）" />
          <div style="color: #909399; font-size: 12px; margin-top: 4px">
            格式: 秒 分 时 日 月 星期
          </div>
        </el-form-item>
        <el-form-item label="目标群 ID" required>
          <el-input v-model="form.chat_id" placeholder="目标群的 Chat ID" />
        </el-form-item>
        <el-form-item label="消息类型">
          <el-select v-model="form.msg_type" style="width: 100%">
            <el-option label="文本" value="text" />
            <el-option label="卡片消息" value="interactive" />
          </el-select>
        </el-form-item>
        <el-form-item label="消息内容" required>
          <el-input v-model="form.content" type="textarea" :rows="4" placeholder='{"text":"你好！"}' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>

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
import {
  getScheduledTasks,
  createScheduledTask,
  updateScheduledTask,
  deleteScheduledTask,
  toggleScheduledTask,
  runScheduledTask,
} from '../api/client'
import { ElMessage } from 'element-plus'

interface Task {
  id: number
  name: string
  cron_expr: string
  chat_id: string
  msg_type: string
  content: string
  enabled: boolean
  last_run_at: string | null
}

const tasks = ref<Task[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const dialogVisible = ref(false)
const submitting = ref(false)
const editingTask = ref<Task | null>(null)

const form = ref({
  name: '',
  cron_expr: '',
  chat_id: '',
  msg_type: 'text',
  content: '',
})

const loadTasks = async () => {
  loading.value = true
  try {
    const res = await getScheduledTasks({ page: page.value, page_size: pageSize.value })
    tasks.value = res.data.data || []
    total.value = res.data.total || 0
  } catch (e) {
    console.error('加载任务失败', e)
  } finally {
    loading.value = false
  }
}

const handlePageChange = (p: number) => {
  page.value = p
  loadTasks()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  page.value = 1
  loadTasks()
}

const showDialog = (task?: Task) => {
  if (task) {
    editingTask.value = task
    form.value = {
      name: task.name,
      cron_expr: task.cron_expr,
      chat_id: task.chat_id,
      msg_type: task.msg_type,
      content: task.content,
    }
  } else {
    editingTask.value = null
    form.value = { name: '', cron_expr: '', chat_id: '', msg_type: 'text', content: '' }
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!form.value.name || !form.value.cron_expr || !form.value.chat_id || !form.value.content) {
    ElMessage.warning('请填写所有必填项')
    return
  }
  submitting.value = true
  try {
    if (editingTask.value) {
      await updateScheduledTask(editingTask.value.id, form.value)
      ElMessage.success('任务已更新')
    } else {
      await createScheduledTask(form.value)
      ElMessage.success('任务已创建')
    }
    dialogVisible.value = false
    await loadTasks()
  } catch (e) {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

const handleToggle = async (id: number) => {
  try {
    await toggleScheduledTask(id)
    await loadTasks()
  } catch (e) {
    ElMessage.error('切换状态失败')
  }
}

const handleDelete = async (id: number) => {
  try {
    await deleteScheduledTask(id)
    ElMessage.success('任务已删除')
    await loadTasks()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

const handleRun = async (id: number) => {
  try {
    await runScheduledTask(id)
    ElMessage.success('任务已执行')
    await loadTasks()
  } catch (e) {
    ElMessage.error('执行失败')
  }
}

const formatTime = (t: string | null) => {
  if (!t) return '-'
  return new Date(t).toLocaleString()
}

onMounted(loadTasks)
</script>
