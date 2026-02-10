<template>
  <div>
    <h2>仪表盘</h2>
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>群组数量</template>
          <div style="font-size: 32px; text-align: center; color: #409EFF">
            {{ stats.group_count }}
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>回复规则</template>
          <div style="font-size: 32px; text-align: center; color: #67C23A">
            {{ stats.rule_count }}
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>定时任务</template>
          <div style="font-size: 32px; text-align: center; color: #E6A23C">
            {{ stats.task_count }}
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>今日消息</template>
          <div style="font-size: 32px; text-align: center; color: #F56C6C">
            {{ stats.messages_today }}
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getDashboardStats } from '../api/client'

const stats = ref({
  group_count: 0,
  rule_count: 0,
  task_count: 0,
  messages_today: 0,
})

onMounted(async () => {
  try {
    const res = await getDashboardStats()
    stats.value = res.data
  } catch (e) {
    console.error('加载统计数据失败', e)
  }
})
</script>
