<template>
  <div>
    <h2>发送消息</h2>
    <el-card style="max-width: 600px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="接收 ID" required>
          <el-input v-model="form.receive_id" placeholder="群 ID、用户 Open ID 等" />
        </el-form-item>
        <el-form-item label="ID 类型" required>
          <el-select v-model="form.receive_id_type" style="width: 100%">
            <el-option label="群 ID (chat_id)" value="chat_id" />
            <el-option label="Open ID" value="open_id" />
            <el-option label="User ID" value="user_id" />
            <el-option label="Union ID" value="union_id" />
            <el-option label="邮箱" value="email" />
          </el-select>
        </el-form-item>
        <el-form-item label="消息类型">
          <el-select v-model="form.msg_type" style="width: 100%">
            <el-option label="文本" value="text" />
            <el-option label="卡片消息" value="interactive" />
          </el-select>
        </el-form-item>
        <el-form-item label="消息内容" required>
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="5"
            :placeholder="contentPlaceholder"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSend" :loading="sending">发送</el-button>
        </el-form-item>
      </el-form>

      <el-alert v-if="result" :title="result" type="success" show-icon style="margin-top: 10px" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { sendMessage } from '../api/client'
import { ElMessage } from 'element-plus'

const form = ref({
  receive_id: '',
  receive_id_type: 'chat_id',
  msg_type: 'text',
  content: '',
})

const sending = ref(false)
const result = ref('')

const contentPlaceholder = computed(() => {
  if (form.value.msg_type === 'text') {
    return '{"text":"你好！"}'
  }
  return '{"type":"template","data":{"template_id":"..."}}'
})

const handleSend = async () => {
  if (!form.value.receive_id || !form.value.content) {
    ElMessage.warning('接收 ID 和消息内容为必填项')
    return
  }
  sending.value = true
  result.value = ''
  try {
    const res = await sendMessage(form.value)
    result.value = `消息发送成功！ID: ${res.data.message_id}`
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '发送失败')
  } finally {
    sending.value = false
  }
}
</script>
