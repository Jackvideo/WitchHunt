# 后端架构分析与总结

## 1. 总体架构概览

WitchHunt 的后端服务采用 Go 语言编写，基于 Gin Web 框架和 Gorilla WebSocket 库构建。项目遵循标准的 Go 项目布局（Standard Go Project Layout），采用了模块化设计，将核心业务逻辑封装在 `internal` 目录下。

### 核心组件

*   **API Server (Gin)**: 处理 HTTP 请求，包括用户认证（注册/登录）和房间管理（创建/加入）。
*   **WebSocket Hub**: 管理所有实时的 WebSocket 连接，负责消息的路由、广播以及与游戏引擎的交互。
*   **Game Engine (SalemEngine)**: 核心游戏逻辑引擎，管理所有进行中的游戏实例，处理玩家行动并维护游戏状态。
*   **Room Manager**: 管理房间的生命周期（创建、加入、离开）以及房间内的玩家列表（在游戏开始前）。
*   **Bot Logic**: 集成在游戏引擎中的 AI 逻辑，支持自动执行机器人玩家的行动。

## 2. 系统架构图
[1](./internal/mermaid-20260225_225659.png)

## 3. 模块详细分析

### 3.1 目录结构

*   **`cmd/`**: 程序的入口点 (`main.go`)，负责初始化配置、数据库连接、组件实例化和启动服务器。
*   **`internal/`**: 私有库代码，包含具体的业务逻辑。
    *   **`config/`**: 配置加载与管理。
    *   **`handler/`**: HTTP 请求处理器 (Auth, Room, WebSocket Upgrade)。
    *   **`middleware/`**: 中间件 (CORS, JWT Auth)。
    *   **`model/`**: 数据模型与数据库操作。
    *   **`room/`**: 房间管理逻辑。
    *   **`game/`**: 核心游戏逻辑。
    *   **`ws/`**: WebSocket 连接管理与消息分发。

### 3.2 关键逻辑流程

#### A. 游戏初始化流程
1.  **创建房间**: 用户通过 HTTP POST `/api/room/create` 请求创建房间，`RoomManager` 生成唯一房间码。
2.  **加入房间**: 其他用户通过 `/api/room/join` 加入。
3.  **建立连接**: 客户端携带 JWT Token 和 Room Code 连接 WebSocket `/api/ws`。
4.  **添加机器人 (可选)**: 房主发送 `add_bot` 指令，`RoomManager` 在房间中添加 AI 玩家。
5.  **开始游戏**: 房主发送 `start_game` 指令，`Hub` 调用 `Engine.StartGame`，初始化游戏状态（发牌、分配身份）。

#### B. 游戏行动循环 (Action Loop)
1.  **客户端行动**: 玩家发送 `game_action` 消息（如：打出卡牌、投票）。
2.  **处理行动**: `Hub` 接收消息并调用 `Engine.HandleAction`。
3.  **状态更新**: `Engine` 验证行动合法性，更新游戏状态（Phase, Hand, Alive status 等），并生成事件日志 (`Events`)。
4.  **广播更新**: 
    *   `Hub` 获取每个玩家的特定视角数据 (`Engine.GetViewForPlayer`) 并推送 `state_update` 消息。
    *   如有事件产生，广播 `game_events` 消息。
5.  **机器人介入**:
    *   `Hub` 在每次处理完玩家行动后，异步调用 `Engine.RunBotTurn`。
    *   `Bot Logic` 检查当前阶段是否有机器人需要行动。
    *   如有，机器人执行行动，再次触发状态更新和广播。

### 3.3 核心数据结构

#### Game State (`server/internal/game/types.go`)
游戏状态是内存中的核心数据结构，包含：
*   **Phase**: 当前游戏阶段 (Day, NightWitch, NightSheriff, Trial 等)。
*   **Players**: 玩家列表，包含手牌、装备、身份牌、存活状态等。
*   **DrawPile/DiscardPile**: 牌堆和弃牌堆。
*   **Turn/Target Info**: 当前回合玩家、夜晚目标、审判信息等。

#### Player View (`server/internal/game/view.go`)
为了防止作弊，服务器只向客户端发送其可见的数据（PlayerView）：
*   **隐藏信息**: 其他玩家的手牌、未翻开的身份牌对当前玩家不可见。
*   **公开信息**: 存活状态、装备牌、已翻开的身份牌、手牌数量。

## 4. 并发模型

*   **HTTP 请求**: 每个请求在独立的 Goroutine 中处理。
*   **WebSocket**: 每个连接有两个 Goroutine（ReadPump 和 WritePump）。
*   **Hub**: 单个 Goroutine 运行主循环 (`run()`)，通过 Channel (`Register`, `Unregister`, `Incoming`) 串行化处理所有 WebSocket 事件，避免了复杂的锁竞争。
*   **Engine**: 使用 `sync.Mutex` 保护 `games` map 和单个游戏的状态，确保并发安全。
*   **RoomManager**: 使用 `sync.RWMutex` 保护房间数据的读写。

## 5. 总结

WitchHunt 后端架构清晰、轻量，适合即时回合制卡牌游戏。
*   **优点**:
    *   逻辑解耦：游戏逻辑与网络层分离。
    *   状态安全：通过 Mutex 和 View 机制保证数据一致性和安全性。
    *   扩展性：添加 Bot 和新卡牌逻辑相对容易。
*   **潜在改进**:
    *   当前游戏状态全内存存储，服务重启会导致游戏丢失（可考虑 Redis 持久化）。
    *   Bot 逻辑目前较简单（随机行动），可进一步优化策略。
