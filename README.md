# Lark Robot

飞书机器人管理平台，提供自动回复、定时消息、群组管理等功能，内置 Web 管理后台。

## 功能特性

- **自动回复** — 支持精确匹配、包含匹配、前缀匹配多种模式，可按群组或全局生效，支持模板变量
- **定时消息** — 基于 Cron 表达式的定时任务，支持发送到群组或私聊
- **群组管理** — 自动同步已加入的群组信息，支持查看群组详情和退群操作
- **用户管理** — 自动同步飞书通讯录用户信息，支持搜索和按需同步
- **消息日志** — 记录所有收发消息，支持分页筛选，自动清理过期记录
- **实时聊天** — Web 端通过 SSE 实时接收消息，支持在线回复、消息撤回和图片查看文下载
- **Web 管理后台** — 响应式界面，统一管理所有功能

## 技术栈

**后端：** Go + Gin + GORM + SQLite

**前端：** Vue 3 + TypeScript + Element Plus + Vite

**通信：** WebSocket（飞书事件）+ SSE（前端推送）+ REST API

## 快速开始

### 前置条件

- Go 1.22+
- Node.js 18+
- 飞书开放平台应用（需获取 App ID 和 App Secret）

### 配置

复制配置文件并修改：

```bash
cp config.yaml.example config.yaml
```

```yaml
server:
  port: 8080
  mode: debug           # debug 或 release

auth:
  username: "admin"
  password: "admin123"
  secret: "change-me-to-a-random-string"  # 请修改为随机字符串

lark:
  app_id: "cli_xxxxxxxxxx"                # 飞书 App ID
  app_secret: "xxxxxxxxxxxxxxxxxxxxxxxx"  # 飞书 App Secret
  base_url: "https://open.feishu.cn"      # 国际版使用 https://open.larksuite.com

database:
  path: "./data/lark-robot.db"

log:
  level: "info"         # debug, info, warn, error
  file: ""              # 留空则仅输出到 stdout
```

### 构建与运行

```bash
# 构建前端
make frontend

# 构建完整应用
make build

# 运行
./lark-robot.exe -config config.yaml
```

开发模式：

```bash
# 终端 1：启动后端
make dev

# 终端 2：启动前端开发服务器（自动代理后端）
make frontend-dev
```

### 跨平台编译

使用 `build.bat` 一键编译多平台产物：

```
build/linux-amd64/lark-robot        # Linux x86_64
build/linux-arm64/lark-robot        # Linux ARM64
build/windows-amd64/lark-robot.exe  # Windows x86_64
build/darwin-amd64/lark-robot       # macOS x86_64
build/darwin-arm64/lark-robot       # macOS ARM64 (Apple Silicon)
```

编译后的二进制文件已嵌入前端静态资源，可直接部署运行。

## 项目结构

```
lark-robot/
├── main.go                 # 应用入口
├── config/                 # 配置加载
├── internal/
│   ├── app/                # 应用初始化与启动
│   ├── broadcast/          # SSE 消息广播
│   ├── database/           # 数据库初始化
│   ├── handler/            # 消息处理链（关键词匹配、默认处理）
│   ├── larkbot/            # 飞书 API 客户端
│   ├── model/              # 数据模型
│   ├── repository/         # 数据访问层
│   ├── scheduler/          # 定时任务调度
│   ├── server/             # HTTP 路由、中间件、API 接口
│   └── service/            # 业务逻辑层
├── static/                 # 嵌入的前端静态资源
├── web/                    # Vue 3 前端项目
├── Makefile                # 构建脚本
└── config.yaml.example     # 配置示例
```

## API 接口

所有接口（登录除外）需在请求头携带 `Authorization: Bearer {token}`。

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/login` | 登录获取 Token |

### 仪表盘

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/dashboard/stats` | 获取统计数据 |

### 消息

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/messages/send` | 发送消息 |
| POST | `/api/messages/reply` | 回复消息 |
| DELETE | `/api/messages/:message_id` | 撤回消息 |
| GET | `/api/messages/logs` | 获取消息日志 |
| GET | `/api/messages/conversations` | 获取会话列表 |
| GET | `/api/messages/stream` | SSE 实时消息流 |
| GET | `/api/images/:message_id/:file_key` | 获取消息中的图片资源 |

### 群组

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/chats` | 获取群组列表 |
| POST | `/api/chats/sync` | 同步群组信息 |
| POST | `/api/chats/:chat_id/leave` | 退出群组 |

### 用户

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/users` | 获取用户列表 |
| POST | `/api/users/sync` | 同步飞书通讯录用户 |
| GET | `/api/users/:open_id` | 获取用户详情 |

### 自动回复规则

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/auto-reply-rules` | 获取规则列表 |
| POST | `/api/auto-reply-rules` | 创建规则 |
| GET | `/api/auto-reply-rules/:id` | 获取规则详情 |
| PUT | `/api/auto-reply-rules/:id` | 更新规则 |
| DELETE | `/api/auto-reply-rules/:id` | 删除规则 |
| POST | `/api/auto-reply-rules/:id/toggle` | 启用/禁用规则 |

### 定时任务

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/scheduled-tasks` | 获取任务列表 |
| POST | `/api/scheduled-tasks` | 创建任务 |
| GET | `/api/scheduled-tasks/:id` | 获取任务详情 |
| PUT | `/api/scheduled-tasks/:id` | 更新任务 |
| DELETE | `/api/scheduled-tasks/:id` | 删除任务 |
| POST | `/api/scheduled-tasks/:id/toggle` | 启用/禁用任务 |
| POST | `/api/scheduled-tasks/:id/run` | 立即执行任务 |

## 飞书应用配置

1. 前往 [飞书开放平台](https://open.feishu.cn/app) 创建企业自建应用
2. 获取 **App ID** 和 **App Secret**，填入 `config.yaml`
3. 在"事件订阅"中启用 **WebSocket 模式**
4. 添加以下事件订阅：
   - `im.message.receive_v1` — 接收消息
5. 添加以下权限：
   - `im:message` — 读写消息
   - `im:chat` — 读取群组信息
   - `contact:user.base:readonly` — 读取用户基本信息
6. 发布应用版本并审批通过

## License

MIT
