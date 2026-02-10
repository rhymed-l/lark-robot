import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('../views/Login.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      redirect: '/dashboard',
    },
    {
      path: '/dashboard',
      name: 'Dashboard',
      component: () => import('../views/Dashboard.vue'),
    },
    {
      path: '/groups',
      name: 'Groups',
      component: () => import('../views/Groups.vue'),
    },
    {
      path: '/auto-reply',
      name: 'AutoReply',
      component: () => import('../views/AutoReplyRules.vue'),
    },
    {
      path: '/scheduled-tasks',
      name: 'ScheduledTasks',
      component: () => import('../views/ScheduledTasks.vue'),
    },
    {
      path: '/send-message',
      name: 'SendMessage',
      component: () => import('../views/SendMessage.vue'),
    },
    {
      path: '/message-logs',
      name: 'MessageLogs',
      component: () => import('../views/MessageLogs.vue'),
    },
    {
      path: '/chat',
      name: 'Chat',
      component: () => import('../views/ChatWindow.vue'),
    },
    {
      path: '/chat/:chatId',
      name: 'ChatWindow',
      component: () => import('../views/ChatWindow.vue'),
    },
  ],
})

// Navigation guard: redirect to login if not authenticated
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.public || token) {
    next()
  } else {
    next('/login')
  }
})

export default router
