# WitchHunt (獵巫鎮 Salem 1692)

基于 Salem 1692 桌游规则的 Web 端多人卡牌游戏。

## 技术栈

- **后端**: Go + Gin + gorilla/websocket + GORM + MySQL
- **前端**: Vue 3 + TypeScript + Vite + Tailwind CSS v4 + Pinia + Vue Router
- **通信**: 原生 WebSocket，服务端游戏引擎权威状态

## 快速开始

### 前置要求

- Go 1.23+
- Node.js 20+
- MySQL 8+

### 1. 数据库

```sql
CREATE DATABASE witchhunt CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 2. 后端

```bash
cd server
go run cmd/main.go
```

可通过环境变量配置：`DB_USER`、`DB_PASSWORD`、`DB_HOST`、`DB_PORT`、`DB_NAME`、`JWT_SECRET`、`SERVER_PORT`。

### 3. 前端

```bash
cd web
npm install
npm run dev
```

访问 `http://localhost:5173`，API 通过 Vite proxy 转发至后端 `:8080`。

---

## 游戏规则

### 概述

玩家分为**村民阵营**和**女巫阵营**。村民需要通过指控和审判找出所有女巫身份牌；女巫则要在夜晚杀害村民，直到所有存活者都是女巫阵营。

- 游戏人数：4-12 人
- 推荐人数：7-8 人

### 身份牌

每位玩家持有多张身份牌（4-9人各5张，10-12人各3张），面朝下放置，对其他人保密。

| 身份 | 说明 |
|------|------|
| 村民 | 无特殊能力，通过指控找出女巫 |
| 女巫 | 夜晚睁眼选择一名玩家杀害。曾持有女巫牌的玩家永久属于女巫阵营 |
| 警长 | 夜晚可用锤子保护一名其他玩家。警长身份被翻开后失去能力 |

### 游戏卡牌

共 57 张游戏卡，分四种颜色：

**红牌（指控）**：对其他玩家发起指控（1/2/3 点）。累计达 7 点触发审判。

**黑牌（事件，抽到立即生效）**：
- **夜晚**：进入夜晚阶段。放在牌堆最底部
- **传染**：持有黑猫的玩家翻开一张身份牌；所有玩家从左边玩家获取一张未翻开的身份牌

**蓝牌（装备，持续生效）**：
- **黑猫**：传染发生时，持有者被迫翻开一张身份牌
- **避难所**：夜晚不会被女巫杀死
- **信徒**：其他玩家不能对持有者使用红牌

**绿牌（一次性效果）**：
- **嫁祸**：将一名玩家的指控转移到另一名玩家身上
- **抢劫**：将一名玩家的手牌全部交给另一名玩家
- **纵火**：丢弃一名玩家所有手牌
- **诅咒**：移除一名玩家的一张装备牌
- **拘留**：目标玩家跳过下一回合
- **辩护**：移除一名玩家的指控

### 游戏流程

#### 白天阶段

每位玩家轮流行动，选择以下之一：

1. **抽牌**：从牌堆抽 2 张牌。抽到黑牌立即生效
2. **出牌**：对其他玩家使用任意数量的手牌，然后结束回合

#### 审判

当一名玩家的指控累计达到 7 点，触发审判：
- 打出第 7 点的玩家可翻开被审判者的一张身份牌
- 审判后清除被审判者所有指控
- 如果翻到女巫牌，该身份暴露

#### 夜晚阶段

当抽到「夜晚」卡时进入：

1. **女巫行动**：所有女巫投票选择一名非女巫玩家杀害
2. **警长行动**：如有未暴露的警长，选择一名其他玩家保护
3. **自首阶段**：所有存活玩家可选择自首（翻开一张身份牌以免于被杀）或跳过
4. **结算**：目标玩家如未自首、无避难所、无锤子保护，则死亡

夜晚结束后弃牌堆洗入牌堆（夜晚牌放最底），回到白天。

#### 玩家死亡

以下情况导致死亡：
- 夜晚被女巫杀害
- 审判中被翻开身份牌导致所有身份暴露
- 所有身份牌均被翻开

死亡时弃掉所有手牌和装备，公开所有身份牌和阵营。

### 胜利条件

- **村民胜利**：所有女巫身份牌被翻开
- **女巫胜利**：所有存活玩家都属于女巫阵营

---

## 项目结构

```
WitchHunt/
├── server/                        # Go 后端
│   ├── cmd/main.go
│   └── internal/
│       ├── config/                # 配置
│       ├── model/                 # GORM 用户模型
│       ├── handler/               # HTTP & WebSocket 处理
│       ├── middleware/            # JWT & CORS
│       ├── ws/                    # WebSocket Hub
│       ├── room/                  # 房间管理
│       └── game/                  # 游戏引擎
│           ├── types.go           # 类型定义
│           ├── deck.go            # 牌组生成
│           ├── engine.go          # 引擎核心 + 状态机
│           ├── actions.go         # 各阶段行动处理
│           └── view.go            # 玩家视角生成
└── web/                           # Vue 3 前端
    └── src/
        ├── views/                 # 登录、大厅、游戏房间
        ├── stores/                # Pinia 状态管理
        ├── composables/           # WebSocket 封装
        └── types/                 # TypeScript 类型
```

## API

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/api/auth/register` | 注册 | 否 |
| POST | `/api/auth/login` | 登录 | 否 |
| POST | `/api/room/create` | 创建房间 | JWT |
| POST | `/api/room/join` | 加入房间 | JWT |
| GET  | `/api/ws` | WebSocket | query token |

## WebSocket 消息

```json
// 客户端 → 服务端
{ "type": "start_game" }
{ "type": "game_action", "payload": { "type": "draw" } }
{ "type": "game_action", "payload": { "type": "play_card", "card_id": "...", "target_id": 5 } }
{ "type": "game_action", "payload": { "type": "witch_kill", "target_id": 5 } }
{ "type": "game_action", "payload": { "type": "confess" } }
{ "type": "game_action", "payload": { "type": "flip_identity", "card_index": 0 } }

// 服务端 → 客户端
{ "type": "state_update", "payload": { /* 个性化游戏状态 */ } }
{ "type": "game_events", "payload": [{ "message": "..." }] }
{ "type": "error", "payload": { "message": "..." } }
```
