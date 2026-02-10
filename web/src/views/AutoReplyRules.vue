<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px">
      <h2 style="margin: 0">自动回复规则</h2>
      <el-button type="primary" @click="showDialog()">添加规则</el-button>
    </div>

    <el-table :data="rules" stripe v-loading="loading">
      <el-table-column prop="keyword" label="关键词" />
      <el-table-column prop="reply_text" label="回复内容" />
      <el-table-column prop="match_mode" label="匹配方式" width="120">
        <template #default="{ row }">
          <el-tag size="small">{{ matchModeLabel(row.match_mode) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="chat_id" label="群 ID" width="200">
        <template #default="{ row }">
          {{ row.chat_id || '全部' }}
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

    <el-dialog v-model="dialogVisible" :title="editingRule ? '编辑规则' : '添加规则'" width="500px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="关键词" required>
          <el-input v-model="form.keyword" placeholder="要匹配的关键词" />
        </el-form-item>
        <el-form-item label="回复内容" required>
          <el-input v-model="form.reply_text" type="textarea" :rows="3" placeholder="回复文本" />
        </el-form-item>
        <el-form-item label="匹配方式">
          <el-select v-model="form.match_mode" style="width: 100%">
            <el-option label="包含" value="contains" />
            <el-option label="精确匹配" value="exact" />
            <el-option label="前缀匹配" value="prefix" />
          </el-select>
        </el-form-item>
        <el-form-item label="群 ID">
          <el-input v-model="form.chat_id" placeholder="留空表示所有群" />
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

const rules = ref<Rule[]>([])
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
  chat_id: '',
})

const matchModeLabel = (mode: string) => {
  const map: Record<string, string> = { contains: '包含', exact: '精确', prefix: '前缀' }
  return map[mode] || mode
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
      chat_id: rule.chat_id,
    }
  } else {
    editingRule.value = null
    form.value = { keyword: '', reply_text: '', match_mode: 'contains', chat_id: '' }
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!form.value.keyword || !form.value.reply_text) {
    ElMessage.warning('关键词和回复内容为必填项')
    return
  }
  submitting.value = true
  try {
    if (editingRule.value) {
      await updateAutoReplyRule(editingRule.value.id, form.value)
      ElMessage.success('规则已更新')
    } else {
      await createAutoReplyRule(form.value)
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

onMounted(loadRules)
</script>
