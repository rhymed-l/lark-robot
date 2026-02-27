<template>
  <div class="page-container">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; flex-shrink: 0">
      <h2 style="margin: 0">定时任务</h2>
      <el-button type="primary" @click="showDialog()">添加任务</el-button>
    </div>

    <div style="flex: 1; min-height: 0; overflow: hidden">
    <el-table :data="tasks" stripe v-loading="loading" height="100%">
      <el-table-column prop="name" label="任务名称" />
      <el-table-column prop="cron_expr" label="Cron 表达式" width="160" />
      <el-table-column label="发送到" width="160">
        <template #default="{ row }">
          {{ groupNameMap[row.chat_id] || row.chat_id }}
        </template>
      </el-table-column>
      <el-table-column prop="msg_type" label="类型" width="80">
        <template #default="{ row }">
          {{ row.msg_type === 'text' ? '文本' : '卡片' }}
        </template>
      </el-table-column>
      <el-table-column prop="content" label="内容" show-overflow-tooltip>
        <template #default="{ row }">
          {{ parseContent(row.content) }}
        </template>
      </el-table-column>
      <el-table-column prop="enabled" label="状态" width="80">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" @change="handleToggle(row.id)" />
        </template>
      </el-table-column>
      <el-table-column prop="last_run_at" label="上次执行" width="170">
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
    </div>

    <el-dialog v-model="dialogVisible" :title="editingTask ? '编辑任务' : '添加任务'" width="550px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="任务名称" required>
          <el-input v-model="form.name" placeholder="任务名称" />
        </el-form-item>
        <el-form-item label="Cron 表达式" required>
          <el-input v-model="form.cron_expr" placeholder="例如: 0 0 9 * * 1-5" />
          <div style="color: #909399; font-size: 12px; margin-top: 4px">
            格式: 秒 分 时 日 月 星期。如 <code>0 0 9 * * 1-5</code> = 工作日每天 9 点
          </div>
        </el-form-item>
        <el-form-item label="发送到" required>
          <el-select
            v-model="form.chat_id"
            filterable
            placeholder="选择群组或私聊"
            style="width: 100%"
          >
            <el-option-group label="群聊">
              <el-option
                v-for="g in groups"
                :key="g.chat_id"
                :label="g.name || g.chat_id"
                :value="g.chat_id"
              />
            </el-option-group>
            <el-option-group v-if="privateChats.length > 0" label="私聊">
              <el-option
                v-for="c in privateChats"
                :key="c.chat_id"
                :label="c.name || c.chat_id"
                :value="c.chat_id"
              />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="消息类型">
          <el-radio-group v-model="form.msg_type">
            <el-radio value="text">文本</el-radio>
            <el-radio value="interactive">卡片消息</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="消息内容" required>
          <el-input
            v-if="form.msg_type === 'text'"
            v-model="form.text"
            type="textarea"
            :rows="4"
            placeholder="输入消息文本"
          />
          <el-input
            v-else
            v-model="form.cardJson"
            type="textarea"
            :rows="6"
            placeholder='{"type":"template","data":{"template_id":"..."}}'
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>

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
import {
  getScheduledTasks,
  createScheduledTask,
  updateScheduledTask,
  deleteScheduledTask,
  toggleScheduledTask,
  runScheduledTask,
  getChats,
  getConversations,
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

interface ChatItem {
  chat_id: string
  name: string
}

const tasks = ref<Task[]>([])
const groups = ref<ChatItem[]>([])
const privateChats = ref<ChatItem[]>([])
const groupNameMap = ref<Record<string, string>>({})
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
  text: '',
  cardJson: '',
})

const parseContent = (content: string): string => {
  try {
    const parsed = JSON.parse(content)
    if (parsed.text) return parsed.text
    return content
  } catch {
    return content
  }
}

const loadChats = async () => {
  try {
    const groupRes = await getChats({ page: 1, page_size: 100 })
    const groupList = groupRes.data.data || []
    groups.value = groupList.map((g: any) => ({ chat_id: g.chat_id, name: g.name || g.chat_id }))
    const map: Record<string, string> = {}
    for (const g of groupList) {
      if (g.name) map[g.chat_id] = g.name
    }

    const convRes = await getConversations()
    const conversations = convRes.data.data || []
    privateChats.value = conversations
      .filter((c: any) => c.chat_type === 'p2p')
      .map((c: any) => {
        const name = c.sender_name || c.chat_id
        map[c.chat_id] = name
        return { chat_id: c.chat_id, name }
      })

    groupNameMap.value = map
  } catch {
    // ignore
  }
}

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

const contentToText = (content: string): string => {
  try {
    const parsed = JSON.parse(content)
    if (parsed.text) return parsed.text
    return content
  } catch {
    return content
  }
}

const showDialog = (task?: Task) => {
  if (task) {
    editingTask.value = task
    const isText = task.msg_type === 'text'
    form.value = {
      name: task.name,
      cron_expr: task.cron_expr,
      chat_id: task.chat_id,
      msg_type: task.msg_type,
      text: isText ? contentToText(task.content) : '',
      cardJson: isText ? '' : task.content,
    }
  } else {
    editingTask.value = null
    form.value = { name: '', cron_expr: '', chat_id: '', msg_type: 'text', text: '', cardJson: '' }
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!form.value.name || !form.value.cron_expr || !form.value.chat_id) {
    ElMessage.warning('请填写所有必填项')
    return
  }

  let content = ''
  if (form.value.msg_type === 'text') {
    if (!form.value.text.trim()) {
      ElMessage.warning('请输入消息内容')
      return
    }
    content = JSON.stringify({ text: form.value.text })
  } else {
    if (!form.value.cardJson.trim()) {
      ElMessage.warning('请输入卡片 JSON')
      return
    }
    content = form.value.cardJson
  }

  submitting.value = true
  const data = {
    name: form.value.name,
    cron_expr: form.value.cron_expr,
    chat_id: form.value.chat_id,
    msg_type: form.value.msg_type,
    content,
  }
  try {
    if (editingTask.value) {
      await updateScheduledTask(editingTask.value.id, data)
      ElMessage.success('任务已更新')
    } else {
      await createScheduledTask(data)
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

onMounted(async () => {
  await loadChats()
  loadTasks()
})
</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 40px);
}
</style>
