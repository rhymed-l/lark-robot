<template>
  <div style="padding: 20px 0; text-align: center; color: #fff; font-size: 16px; font-weight: bold;">
    飞书机器人
  </div>
  <el-menu
    :default-active="activeMenu"
    router
    background-color="#304156"
    text-color="#bfcbd9"
    active-text-color="#409EFF"
  >
    <el-menu-item index="/dashboard">
      <el-icon><Odometer /></el-icon>
      <span>仪表盘</span>
    </el-menu-item>
    <el-menu-item index="/chat">
      <el-icon><ChatDotRound /></el-icon>
      <span>聊天</span>
      <el-badge v-if="totalUnread > 0" :value="totalUnread" :max="99" class="chat-badge" />
    </el-menu-item>
    <el-menu-item index="/groups">
      <el-icon><List /></el-icon>
      <span>群组管理</span>
    </el-menu-item>
    <el-menu-item index="/auto-reply">
      <el-icon><ChatLineSquare /></el-icon>
      <span>自动回复</span>
    </el-menu-item>
    <el-menu-item index="/scheduled-tasks">
      <el-icon><Timer /></el-icon>
      <span>定时任务</span>
    </el-menu-item>
    <el-menu-item index="/send-message">
      <el-icon><Promotion /></el-icon>
      <span>发送消息</span>
    </el-menu-item>
    <el-menu-item index="/message-logs">
      <el-icon><Document /></el-icon>
      <span>消息日志</span>
    </el-menu-item>
  </el-menu>
  <div class="logout-area">
    <el-button text style="color: #bfcbd9; width: 100%" @click="handleLogout">
      退出登录
    </el-button>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, type ComputedRef } from 'vue'
import { useRoute } from 'vue-router'
import {
  Odometer,
  ChatDotRound,
  ChatLineSquare,
  Timer,
  Promotion,
  Document,
  List,
} from '@element-plus/icons-vue'
import { logout } from '../api/client'

const route = useRoute()
const activeMenu = computed(() => {
  if (route.path.startsWith('/chat')) return '/chat'
  return route.path
})

const totalUnread = inject<ComputedRef<number>>('totalUnread', computed(() => 0))

const handleLogout = () => {
  logout()
}
</script>

<style scoped>
.logout-area {
  position: absolute;
  bottom: 12px;
  width: 100%;
  padding: 0 12px;
  box-sizing: border-box;
}
.chat-badge {
  margin-left: 4px;
}
.chat-badge :deep(.el-badge__content) {
  top: 2px;
}
</style>
