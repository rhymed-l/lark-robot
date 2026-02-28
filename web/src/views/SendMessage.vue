<template>
  <div>
    <h2>发送消息</h2>
    <el-card style="max-width: 600px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="发送到" required>
          <el-select
            v-model="form.receive_id"
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
            <el-radio value="image">图片</el-radio>
            <el-radio value="file">文件</el-radio>
            <el-radio value="interactive">卡片消息</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 文本消息 -->
        <el-form-item v-if="form.msg_type === 'text'" label="消息内容" required>
          <el-input
            v-model="form.text"
            type="textarea"
            :rows="4"
            placeholder="输入消息文本"
          />
        </el-form-item>

        <!-- 图片消息 -->
        <el-form-item v-if="form.msg_type === 'image'" label="选择图片" required>
          <el-upload
            ref="imageUploadRef"
            :auto-upload="false"
            :limit="1"
            accept="image/*"
            :on-change="handleImageChange"
            :on-remove="handleImageRemove"
            list-type="picture"
          >
            <el-button type="primary">选择图片</el-button>
            <template #tip>
              <div class="el-upload__tip">支持 JPEG/PNG/WEBP/GIF/TIFF/BMP，最大 10MB</div>
            </template>
          </el-upload>
        </el-form-item>

        <!-- 文件消息 -->
        <el-form-item v-if="form.msg_type === 'file'" label="选择文件" required>
          <el-upload
            ref="fileUploadRef"
            :auto-upload="false"
            :limit="1"
            :on-change="handleFileChange"
            :on-remove="handleFileRemove"
          >
            <el-button type="primary">选择文件</el-button>
            <template #tip>
              <div class="el-upload__tip">支持 PDF/DOC/XLS/PPT/MP4 等格式，最大 30MB</div>
            </template>
          </el-upload>
        </el-form-item>

        <!-- 卡片消息 -->
        <el-form-item v-if="form.msg_type === 'interactive'" label="消息内容" required>
          <el-input
            v-model="form.cardJson"
            type="textarea"
            :rows="6"
            placeholder='{"type":"template","data":{"template_id":"..."}}'
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
import { ref, onMounted } from 'vue'
import { sendMessage, getChats, getConversations, uploadImage, uploadFile } from '../api/client'
import { ElMessage } from 'element-plus'
import type { UploadFile } from 'element-plus'

interface ChatItem {
  chat_id: string
  name: string
}

const groups = ref<ChatItem[]>([])
const privateChats = ref<ChatItem[]>([])

const form = ref({
  receive_id: '',
  msg_type: 'text',
  text: '',
  cardJson: '',
})

const imageFile = ref<File | null>(null)
const fileFile = ref<File | null>(null)
const imageUploadRef = ref()
const fileUploadRef = ref()

const sending = ref(false)
const result = ref('')

const loadChats = async () => {
  try {
    const groupRes = await getChats({ page: 1, page_size: 100 })
    groups.value = (groupRes.data.data || []).map((g: any) => ({
      chat_id: g.chat_id,
      name: g.name || g.chat_id,
    }))

    const convRes = await getConversations()
    const conversations = convRes.data.data || []
    privateChats.value = conversations
      .filter((c: any) => c.chat_type === 'p2p')
      .map((c: any) => ({
        chat_id: c.chat_id,
        name: c.sender_name || c.chat_id,
      }))
  } catch {
    // ignore
  }
}

const handleImageChange = (uploadFile: UploadFile) => {
  imageFile.value = uploadFile.raw || null
}

const handleImageRemove = () => {
  imageFile.value = null
}

const handleFileChange = (uploadFile: UploadFile) => {
  fileFile.value = uploadFile.raw || null
}

const handleFileRemove = () => {
  fileFile.value = null
}

const handleSend = async () => {
  if (!form.value.receive_id) {
    ElMessage.warning('请选择发送对象')
    return
  }

  sending.value = true
  result.value = ''

  try {
    let content = ''
    let msgType = form.value.msg_type

    if (msgType === 'text') {
      if (!form.value.text.trim()) {
        ElMessage.warning('请输入消息内容')
        sending.value = false
        return
      }
      content = JSON.stringify({ text: form.value.text })
    } else if (msgType === 'image') {
      if (!imageFile.value) {
        ElMessage.warning('请选择图片')
        sending.value = false
        return
      }
      // Upload image first
      const uploadRes = await uploadImage(imageFile.value)
      const imageKey = uploadRes.data.image_key
      content = JSON.stringify({ image_key: imageKey })
    } else if (msgType === 'file') {
      if (!fileFile.value) {
        ElMessage.warning('请选择文件')
        sending.value = false
        return
      }
      // Upload file first
      const uploadRes = await uploadFile(fileFile.value)
      const fileKey = uploadRes.data.file_key
      content = JSON.stringify({ file_key: fileKey, file_name: fileFile.value.name })
    } else if (msgType === 'interactive') {
      if (!form.value.cardJson.trim()) {
        ElMessage.warning('请输入卡片 JSON')
        sending.value = false
        return
      }
      content = form.value.cardJson
    }

    const res = await sendMessage({
      receive_id: form.value.receive_id,
      receive_id_type: 'chat_id',
      msg_type: msgType,
      content,
    })
    result.value = `发送成功！消息 ID: ${res.data.message_id}`

    // Reset form content
    if (msgType === 'text') {
      form.value.text = ''
    } else if (msgType === 'image') {
      imageFile.value = null
      imageUploadRef.value?.clearFiles()
    } else if (msgType === 'file') {
      fileFile.value = null
      fileUploadRef.value?.clearFiles()
    }
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '发送失败')
  } finally {
    sending.value = false
  }
}

onMounted(loadChats)
</script>
