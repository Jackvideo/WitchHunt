# WitchHunt

Web 端多人卡牌游戏框架（类似狼人杀）。

## 技术栈

- **后端**: Go + Gin + gorilla/websocket + GORM + MySQL
- **前端**: Vue 3 + TypeScript + Vite + Tailwind CSS v4 + Pinia + Vue Router
- **通信**: 原生 WebSocket

## 项目结构

```
WitchHunt/
├── server/                     # Go 后端
│   ├── cmd/main.go             # 入口
│   └── internal/
│       ├── config/             # 配置
│       ├── model/              # 数据模型 (GORM)
│       ├── handler/            # HTTP & WebSocket 处理器
│       ├── middleware/         # JWT 鉴权 & CORS
│       ├── ws/                 # WebSocket Hub & Client
│       ├── room/               # 房间管理 (内存, 并发安全)
│       └── game/               # 游戏引擎接口 (待实现)
└── web/                        # Vue 3 前端
    └── src/
        ├── views/              # 页面 (登录、大厅、房间)
        ├── stores/             # Pinia 状态管理
        ├── composables/        # WebSocket 组合式函数
        ├── types/              # TypeScript 类型
        └── router/             # 路由配置
```

## 前置要求

- Go 1.23+
- Node.js 20+
- MySQL 8+

## 快速开始

### 1. 数据库

确保 MySQL 运行中，并创建数据库：

```sql
CREATE DATABASE witchhunt CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 2. 后端

```bash
cd server

# 可通过环境变量自定义配置（默认连接 root@127.0.0.1:3306/witchhunt）
# export DB_USER=root DB_PASSWORD=yourpass DB_HOST=127.0.0.1 DB_PORT=3306 DB_NAME=witchhunt
# export JWT_SECRET=your-secret-key

go run cmd/main.go
```

后端默认监听 `:8080`。

### 3. 前端

```bash
cd web
npm install
npm run dev
```

前端默认运行在 `http://localhost:5173`，API 请求通过 Vite proxy 转发至后端。

## API

| 方法   | 路径                | 说明       | 鉴权 |
| ------ | ------------------- | ---------- | ---- |
| POST   | `/api/auth/register`| 注册       | 否   |
| POST   | `/api/auth/login`   | 登录       | 否   |
| POST   | `/api/room/create`  | 创建房间   | JWT  |
| POST   | `/api/room/join`    | 加入房间   | JWT  |
| GET    | `/api/ws`           | WebSocket  | query token |

## WebSocket 消息

```json
{ "type": "player_join | player_leave | game_action", "payload": {} }
```

## 扩展游戏逻辑

实现 `server/internal/game/Engine` 接口即可接入自定义游戏规则：

```go
type Engine interface {
    OnPlayerJoin(roomCode string, userID uint)
    OnPlayerLeave(roomCode string, userID uint)
    OnPlayerAction(roomCode string, userID uint, action json.RawMessage) (json.RawMessage, error)
    GetState(roomCode string) json.RawMessage
}
```
