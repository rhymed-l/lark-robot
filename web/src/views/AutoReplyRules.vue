<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px">
      <h2 style="margin: 0">自动回复规则</h2>
      <el-button type="primary" @click="showDialog()">添加规则</el-button>
    </div>

    <el-table :data="rules" stripe v-loading="loading">
      <el-table-column prop="keyword" label="关键词" />
      <el-table-column prop="reply_text" label="回复内容" show-overflow-tooltip />
      <el-table-column prop="match_mode" label="匹配方式" width="120">
        <template #default="{ row }">
          <el-tag size="small">{{ matchModeLabel(row.match_mode) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="适用范围" width="200">
        <template #default="{ row }">
          <span v-if="!row.chat_id">全部</span>
          <span v-else>{{ formatChatIds(row.chat_id) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="enabled" label="状态" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" @change="handleToggle(row.id)" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="showDialog(row)">编辑</el-button>
          <el-popconfirm title="确定删除该规则吗？" @confirm="handleDelete(row.id)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editingRule ? '编辑规则' : '添加规则'" width="560px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="关键词" required>
          <el-input v-model="form.keyword" placeholder="要匹配的关键词" />
        </el-form-item>
        <el-form-item label="匹配方式">
          <el-select v-model="form.match_mode" style="width: 100%">
            <el-option label="包含" value="contains" />
            <el-option label="精确匹配" value="exact" />
            <el-option label="前缀匹配" value="prefix" />
          </el-select>
        </el-form-item>
        <el-form-item label="回复内容" required>
          <el-input v-model="form.reply_text" type="textarea" :rows="3" placeholder="回复文本，支持模板变量" />
          <div class="var-hint">
            可用变量：
            <el-tooltip v-for="v in templateVars" :key="v.key" :content="v.desc" placement="top">
              <el-tag size="small" @click="insertVar(v.key)" class="var-tag" effect="plain">
                {{ v.key }}
              </el-tag>
            </el-tooltip>
          </div>
        </el-form-item>
        <el-form-item label="适用群组">
          <el-select
            v-model="form.chat_ids"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="留空表示所有群/私聊"
            style="width: 100%"
          >
            <el-option
              v-for="g in groups"
              :key="g.chat_id"
              :label="g.name || g.chat_id"
              :value="g.chat_id"
            />
          </el-select>
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
  getAutoReplyRules,
  createAutoReplyRule,
  updateAutoReplyRule,
  deleteAutoReplyRule,
  toggleAutoReplyRule,
  getChats,
} from '../api/client'
import { ElMessage } from 'element-plus'

interface Rule {
  id: number
  keyword: string
  reply_text: string
  match_mode: string
  chat_id: string
  enabled: boolean
}

interface Group {
  chat_id: string
  name: string
}

const templateVars = [
  { key: '{{chat_id}}', desc: '会话 ID' },
  { key: '{{chat_type}}', desc: '会话类型' },
  { key: '{{sender_id}}', desc: '发送者 ID' },
  { key: '{{sender_name}}', desc: '发送者名称' },
  { key: '{{message_id}}', desc: '消息 ID' },
  { key: '{{content}}', desc: '消息内容' },
]

const rules = ref<Rule[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const dialogVisible = ref(false)
const submitting = ref(false)
const editingRule = ref<Rule | null>(null)

const form = ref({
  keyword: '',
  reply_text: '',
  match_mode: 'contains',
  chat_ids: [] as string[],
})

// Map chatId -> group name for display
const groupNameMap = ref<Record<string, string>>({})

const matchModeLabel = (mode: string) => {
  const map: Record<string, string> = { contains: '包含', exact: '精确', prefix: '前缀' }
  return map[mode] || mode
}

const formatChatIds = (chatId: string): string => {
  if (!chatId) return '全部'
  const ids = chatId.split(',')
  return ids.map(id => groupNameMap.value[id] || id.slice(0, 12) + '...').join(', ')
}

const insertVar = (varKey: string) => {
  form.value.reply_text += varKey
}

const loadGroups = async () => {
  try {
    const res = await getChats({ page: 1, page_size: 100 })
    groups.value = res.data.data || []
    const map: Record<string, string> = {}
    for (const g of groups.value) {
      if (g.name) map[g.chat_id] = g.name
    }
    groupNameMap.value = map
  } catch {
    // ignore
  }
}

const loadRules = async () => {
  loading.value = true
  try {
    const res = await getAutoReplyRules({ page: page.value, page_size: pageSize.value })
    rules.value = res.data.data || []
    total.value = res.data.total || 0
  } catch (e) {
    console.error('加载规则失败', e)
  } finally {
    loading.value = false
  }
}

const handlePageChange = (p: number) => {
  page.value = p
  loadRules()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  page.value = 1
  loadRules()
}

const showDialog = (rule?: Rule) => {
  if (rule) {
    editingRule.value = rule
    form.value = {
      keyword: rule.keyword,
      reply_text: rule.reply_text,
      match_mode: rule.match_mode,
      chat_ids: rule.chat_id ? rule.chat_id.split(',') : [],
    }
  } else {
    editingRule.value = null
    form.value = { keyword: '', reply_text: '', match_mode: 'contains', chat_ids: [] }
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!form.value.keyword || !form.value.reply_text) {
    ElMessage.warning('关键词和回复内容为必填项')
    return
  }
  submitting.value = true
  const data = {
    keyword: form.value.keyword,
    reply_text: form.value.reply_text,
    match_mode: form.value.match_mode,
    chat_id: form.value.chat_ids.join(','),
  }
  try {
    if (editingRule.value) {
      await updateAutoReplyRule(editingRule.value.id, data)
      ElMessage.success('规则已更新')
    } else {
      await createAutoReplyRule(data)
      ElMessage.success('规则已创建')
    }
    dialogVisible.value = false
    await loadRules()
  } catch (e) {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

const handleToggle = async (id: number) => {
  try {
    await toggleAutoReplyRule(id)
    await loadRules()
  } catch (e) {
    ElMessage.error('切换状态失败')
  }
}

const handleDelete = async (id: number) => {
  try {
    await deleteAutoReplyRule(id)
    ElMessage.success('规则已删除')
    await loadRules()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

onMounted(async () => {
  await loadGroups()
  loadRules()
})
</script>

<style scoped>
.var-hint {
  margin-top: 6px;
  line-height: 2;
}
.var-tag {
  cursor: pointer;
  margin-right: 4px;
  margin-bottom: 4px;
}
.var-tag:hover {
  color: #409eff;
  border-color: #409eff;
}
</style>
