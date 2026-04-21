# SessionKey 格式示例

## 1. Agent 作用域 SessionKey（标准格式）

所有 Agent 作用域的 SessionKey 都以 `agent:` 前缀开头，格式为：`agent:<agent_id>:<rest>`

### 1.1 主会话（Main Session）
**格式**：`agent:<agent_id>:main`

| 示例 | 说明 |
|------|------|
| `agent:main:main` | 默认 agent 的主会话 |
| `agent:sales:main` | "sales" agent 的主会话 |
| `agent:support-bot:main` | "support-bot" agent 的主会话（agent_id 会被规范化） |

### 1.2 直接消息会话（Direct Message Sessions）

#### a) 按对等体（Per-Peer） - `DMScopePerPeer`
**格式**：`agent:<agent_id>:direct:<peer_id>`

| 示例 | 说明 |
|------|------|
| `agent:main:direct:user123` | 用户 user123 在所有频道的共享会话 |
| `agent:main:direct:john` | 用户 john（通过身份链接映射）的会话 |
| `agent:sales:direct:customer456` | sales agent 与客户 customer456 的会话 |

#### b) 按频道对等体（Per-Channel-Peer） - `DMScopePerChannelPeer`
**格式**：`agent:<agent_id>:<channel>:direct:<peer_id>`

| 示例 | 说明 |
|------|------|
| `agent:main:telegram:direct:user123` | Telegram 频道中用户 user123 的会话 |
| `agent:main:discord:direct:john#1234` | Discord 频道中用户 john#1234 的会话 |
| `agent:support:whatsapp:direct:+1234567890` | WhatsApp 频道中用户的会话 |

#### c) 按账户频道对等体（Per-Account-Channel-Peer） - `DMScopePerAccountChannelPeer`
**格式**：`agent:<agent_id>:<channel>:<account_id>:direct:<peer_id>`

| 示例 | 说明 |
|------|------|
| `agent:main:telegram:bot1:direct:user123` | Telegram 账户 bot1 中用户 user123 的会话 |
| `agent:main:telegram:bot2:direct:user123` | Telegram 账户 bot2 中同一用户 user123 的独立会话 |
| `agent:main:discord:production:direct:john` | Discord 生产环境账户中用户 john 的会话 |

#### d) 主会话（Main） - `DMScopeMain`
**格式**：`agent:<agent_id>:main`

| 示例 | 说明 |
|------|------|
| `agent:main:main` | 所有直接消息共享同一主会话 |
| `agent:assistant:main` | assistant agent 的所有直接消息共享会话 |

### 1.3 群组/频道会话（Group/Channel Sessions）
**格式**：`agent:<agent_id>:<channel>:<peer_kind>:<peer_id>`

| 示例 | 说明 |
|------|------|
| `agent:main:telegram:group:chat456` | Telegram 群组 chat456 的会话 |
| `agent:main:discord:channel:general` | Discord 频道 #general 的会话 |
| `agent:main:slack:channel:random` | Slack 频道 #random 的会话 |
| `agent:main:matrix:room:!abc123:matrix.org` | Matrix 房间的会话 |
| `agent:sales:telegram:group:sales-team` | sales agent 在销售团队群组的会话 |

### 1.4 子代理会话（Subagent Sessions）
**格式**：`agent:<agent_id>:subagent:<task_id>`

| 示例 | 说明 |
|------|------|
| `agent:main:subagent:task-1` | main agent 的子代理任务 task-1 |
| `agent:main:subagent:research-20240420` | 研究任务的子代理 |
| `agent:analyst:subagent:data-processing` | analyst agent 的数据处理子代理 |

## 2. 子代理专用 SessionKey
**格式**：`subagent:<task_id>`

| 示例 | 说明 |
|------|------|
| `subagent:task-1` | 简单的子代理任务 |
| `subagent:background-job` | 后台作业 |
| `subagent:async-process` | 异步处理任务 |

## 3. 非 Agent 作用域 SessionKey
**格式**：任意字符串，不以 `agent:` 开头

| 示例 | 说明 |
|------|------|
| `heartbeat` | 心跳服务的会话 |
| `test-session` | 测试用的会话 |
| `session-1` | 通用会话标识 |
| `cli-direct` | CLI 直接调用的会话 |
| `system-monitor` | 系统监控会话 |

## 4. 身份链接（Identity Links）示例

身份链接允许将不同平台的用户映射到同一规范名称：

### 配置示例：
```json
{
  "identity_links": {
    "john": ["telegram:user123", "discord:john#1234", "slack:U123456"],
    "alice": ["telegram:alice789", "whatsapp:+1234567890"]
  }
}
```

### 生成的 SessionKey：
| 用户输入 | 实际 SessionKey | 说明 |
|----------|----------------|------|
| Telegram: user123 | `agent:main:direct:john` | 映射到规范名称 "john" |
| Discord: john#1234 | `agent:main:direct:john` | 同样映射到 "john" |
| Telegram: alice789 | `agent:main:direct:alice` | 映射到规范名称 "alice" |

## 5. 实际使用场景示例

### 场景 1：多平台客户支持
- **用户**：客户 "john" 在 Telegram 和 Discord 联系支持
- **SessionKey**：`agent:support:direct:john`（使用 `DMScopePerPeer`）
- **效果**：两个平台的消息在同一个会话中，支持人员可以看到完整对话历史

### 场景 2：团队协作机器人
- **频道**：Slack 团队频道 #engineering
- **SessionKey**：`agent:eng-bot:slack:channel:engineering`
- **效果**：工程频道的所有消息共享同一会话上下文

### 场景 3：多账户管理
- **场景**：同一用户在两个 Telegram 机器人账户咨询
- **SessionKey 1**：`agent:main:telegram:bot1:direct:user123`
- **SessionKey 2**：`agent:main:telegram:bot2:direct:user123`
- **效果**：两个账户有独立的会话，不会混淆

### 场景 4：异步任务处理
- **任务**：处理大量数据
- **SessionKey**：`agent:main:subagent:data-processing-20240420`
- **效果**：子代理独立运行，不影响主会话

## 6. 代码生成示例

```go
// 生成主会话键
mainKey := routing.BuildAgentMainSessionKey("main") // "agent:main:main"

// 生成按对等体会话键
peerKey := routing.BuildAgentPeerSessionKey(routing.SessionKeyParams{
    AgentID: "main",
    Channel: "telegram",
    Peer:    &routing.RoutePeer{Kind: "direct", ID: "user123"},
    DMScope: routing.DMScopePerPeer,
}) // "agent:main:direct:user123"

// 生成群组会话键
groupKey := routing.BuildAgentPeerSessionKey(routing.SessionKeyParams{
    AgentID: "main",
    Channel: "telegram",
    Peer:    &routing.RoutePeer{Kind: "group", ID: "chat456"},
}) // "agent:main:telegram:group:chat456"

// 解析会话键
parsed := routing.ParseAgentSessionKey("agent:main:telegram:direct:user123")
// parsed.AgentID = "main"
// parsed.Rest = "telegram:direct:user123"

// 检测子代理
isSubagent := routing.IsSubagentSessionKey("agent:main:subagent:task-1") // true
```

## 总结

SessionKey 的设计提供了灵活的会话管理：
1. **前缀标识**：`agent:` 标识 Agent 作用域
2. **结构清晰**：`agent:<agent_id>:<scope_details>`
3. **粒度可控**：通过 DMScope 控制隔离级别
4. **平台集成**：支持身份链接和跨平台会话
5. **类型丰富**：覆盖主会话、直接消息、群组、子代理等多种场景