# Agent 与 Channel 绑定配置指南

## 概述

在 PicoClaw 中，Agent 与 Channel 的绑定通过 `bindings` 配置实现。当消息从特定 Channel（如 weixin）到达时，系统会根据绑定规则决定使用哪个 Agent 处理。

## 绑定机制

### 路由优先级（7级级联）

系统按以下优先级匹配绑定：

1. **Peer 匹配** - 最具体：特定用户/群组
2. **Parent Peer 匹配** - 回复消息的父消息
3. **Guild 匹配** - Discord 服务器
4. **Team 匹配** - Slack 团队
5. **Account 匹配** - 特定账号
6. **Channel 通配符** - 整个 Channel
7. **默认 Agent** - 无匹配时的回退

### 配置结构

```json
{
  "agents": {
    "defaults": { ... },
    "list": [
      {
        "name": "agent01",
        "model_name": "deepseek-chat",
        "max_tokens": 8192,
        "temperature": 0.7
      },
      {
        "name": "agent02",
        "model_name": "gpt-4",
        "max_tokens": 4096,
        "temperature": 0.5
      }
    ]
  },
  "bindings": [
    // 绑定规则
  ]
}
```

## 配置示例

### 示例 1：weixin 通道使用 agent01

```json
{
  "agents": {
    "defaults": {
      "model_name": "deepseek-chat",
      "max_tokens": 8192,
      "temperature": 0.7
    },
    "list": [
      {
        "name": "agent01",
        "model_name": "deepseek-chat",
        "max_tokens": 8192,
        "temperature": 0.7,
        "tools": {
          "web_fetch": { "enabled": true },
          "calculator": { "enabled": true }
        }
      },
      {
        "name": "agent02",
        "model_name": "gpt-4",
        "max_tokens": 4096,
        "temperature": 0.5
      }
    ]
  },
  "bindings": [
    {
      "agent_id": "agent01",
      "match": {
        "channel": "weixin"
      }
    }
  ]
}
```

### 示例 2：按用户绑定

```json
{
  "bindings": [
    {
      "agent_id": "vip-agent",
      "match": {
        "channel": "weixin",
        "peer": {
          "kind": "direct",
          "id": "user123"  // 微信用户ID
        }
      }
    },
    {
      "agent_id": "group-agent",
      "match": {
        "channel": "weixin",
        "peer": {
          "kind": "group",
          "id": "group456"  // 微信群ID
        }
      }
    }
  ]
}
```

### 示例 3：多级匹配

```json
{
  "bindings": [
    // 1. 特定用户使用专属 Agent
    {
      "agent_id": "personal-assistant",
      "match": {
        "channel": "weixin",
        "peer": {
          "kind": "direct",
          "id": "vip-user-001"
        }
      }
    },
    // 2. 特定群组使用群组 Agent
    {
      "agent_id": "group-support",
      "match": {
        "channel": "weixin",
        "peer": {
          "kind": "group",
          "id": "support-group"
        }
      }
    },
    // 3. 所有 weixin 消息使用默认 Agent
    {
      "agent_id": "agent01",
      "match": {
        "channel": "weixin"
      }
    },
    // 4. 其他 Channel 使用另一个 Agent
    {
      "agent_id": "agent02",
      "match": {
        "channel": "telegram"
      }
    }
  ]
}
```

## 详细配置说明

### 1. Agent 配置

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model_name": "deepseek-chat",
      "max_tokens": 8192,
      "context_window": 131072,
      "temperature": 0.7,
      "max_tool_iterations": 20,
      "summarize_message_threshold": 20,
      "summarize_token_percent": 75
    },
    "list": [
      {
        "name": "agent01",
        "model_name": "deepseek-chat",
        "max_tokens": 8192,
        "temperature": 0.7,
        "tools": {
          "web_fetch": { "enabled": true },
          "calculator": { "enabled": true },
          "subagent": { "enabled": true }
        },
        "subagents": {
          "allow_agents": ["data-analyst", "research-assistant"]
        }
      }
    ]
  }
}
```

### 2. Binding 匹配字段

#### `channel` (必需)
- 通道名称：`"weixin"`, `"telegram"`, `"discord"`, `"slack"` 等
- 必须与 Channel 配置中的名称一致

#### `account_id` (可选)
- 账号ID，用于多账号场景
- 示例：`"bot-account-1"`

#### `peer` (可选)
- 对等体匹配
```json
"peer": {
  "kind": "direct",  // "direct" | "group" | "channel"
  "id": "user123"    // 用户ID或群组ID
}
```

#### `guild_id` (可选)
- Discord 服务器ID
- 示例：`"guild-abc123"`

#### `team_id` (可选)
- Slack 团队ID
- 示例：`"T123456"`

## 实际配置步骤

### 步骤 1：定义 Agents

首先在 `agents.list` 中定义你的 Agents：

```json
{
  "agents": {
    "defaults": {
      "model_name": "deepseek-chat",
      "max_tokens": 8192,
      "temperature": 0.7
    },
    "list": [
      {
        "name": "agent01",
        "model_name": "deepseek-chat",
        "max_tokens": 8192,
        "temperature": 0.7,
        "tools": {
          "web_fetch": { "enabled": true }
        }
      },
      {
        "name": "agent02",
        "model_name": "gpt-4",
        "max_tokens": 4096,
        "temperature": 0.5
      }
    ]
  }
}
```

### 步骤 2：配置 Channel

确保 weixin Channel 已启用：

```json
{
  "channels": {
    "weixin": {
      "enabled": true,
      "app_id": "YOUR_WECHAT_APP_ID",
      "app_secret": "YOUR_WECHAT_APP_SECRET",
      "token": "YOUR_WECHAT_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "allow_from": ["*"],
      "reasoning_channel_id": ""
    }
  }
}
```

### 步骤 3：创建绑定

添加绑定规则，将 weixin 通道路由到 agent01：

```json
{
  "bindings": [
    {
      "agent_id": "agent01",
      "match": {
        "channel": "weixin"
      }
    }
  ]
}
```

### 步骤 4：完整配置示例

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model_name": "deepseek-chat",
      "max_tokens": 8192,
      "context_window": 131072,
      "temperature": 0.7,
      "max_tool_iterations": 20
    },
    "list": [
      {
        "name": "agent01",
        "model_name": "deepseek-chat",
        "max_tokens": 8192,
        "temperature": 0.7,
        "tools": {
          "web_fetch": { "enabled": true },
          "calculator": { "enabled": true },
          "subagent": { "enabled": true }
        }
      }
    ]
  },
  "bindings": [
    {
      "agent_id": "agent01",
      "match": {
        "channel": "weixin"
      }
    }
  ],
  "channels": {
    "weixin": {
      "enabled": true,
      "app_id": "YOUR_WECHAT_APP_ID",
      "app_secret": "YOUR_WECHAT_APP_SECRET",
      "token": "YOUR_WECHAT_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "allow_from": ["*"]
    }
  }
}
```

## 高级配置

### 1. 条件绑定

```json
{
  "bindings": [
    // 工作时间使用工作 Agent
    {
      "agent_id": "work-agent",
      "match": {
        "channel": "weixin"
      },
      "condition": {
        "time_range": "09:00-18:00",
        "weekdays": ["Mon", "Tue", "Wed", "Thu", "Fri"]
      }
    },
    // 非工作时间使用休闲 Agent
    {
      "agent_id": "casual-agent",
      "match": {
        "channel": "weixin"
      },
      "condition": {
        "time_range": "18:00-09:00",
        "weekdays": ["Mon", "Tue", "Wed", "Thu", "Fri"]
      }
    },
    // 周末使用周末 Agent
    {
      "agent_id": "weekend-agent",
      "match": {
        "channel": "weixin"
      },
      "condition": {
        "weekdays": ["Sat", "Sun"]
      }
    }
  ]
}
```

### 2. 基于内容的动态路由

```json
{
  "bindings": [
    {
      "agent_id": "tech-support",
      "match": {
        "channel": "weixin"
      },
      "condition": {
        "keywords": ["error", "bug", "crash", "technical", "help"]
      }
    },
    {
      "agent_id": "general-assistant",
      "match": {
        "channel": "weixin"
      }
    }
  ]
}
```

### 3. 多账号支持

```json
{
  "bindings": [
    {
      "agent_id": "business-bot",
      "match": {
        "channel": "weixin",
        "account_id": "business-account"
      }
    },
    {
      "agent_id": "personal-bot",
      "match": {
        "channel": "weixin",
        "account_id": "personal-account"
      }
    }
  ]
}
```

## 调试和验证

### 1. 检查路由日志

启动 PicoClaw 时添加调试日志：

```bash
LOG_LEVEL=debug ./picoclaw
```

查看路由决策日志：
```
[D] Resolving route for channel=weixin, peer=direct/user123
[D] Matched binding: agent_id=agent01, channel=weixin
[D] Selected agent: agent01
```

### 2. 测试路由

使用测试工具验证绑定：

```go
// 测试代码示例
func TestWeixinBinding() {
    input := routing.RouteInput{
        Channel: "weixin",
        Peer: &routing.RoutePeer{Kind: "direct", ID: "test-user"},
    }
    route := resolver.ResolveRoute(input)
    if route.AgentID != "agent01" {
        t.Errorf("Expected agent01, got %s", route.AgentID)
    }
}
```

### 3. 验证配置

使用配置验证工具：

```bash
./picoclaw validate-config config.json
```

## 常见问题

### 问题 1：绑定不生效

**可能原因**：
1. Agent 名称拼写错误
2. Channel 名称不匹配
3. 绑定优先级被更高优先级的规则覆盖

**解决方案**：
1. 检查 `agent_id` 是否与 `agents.list` 中的 `name` 一致
2. 确认 `channel` 名称与 Channel 配置中的名称一致
3. 检查是否有更具体的绑定规则（如 peer 绑定）覆盖了 channel 绑定

### 问题 2：多个绑定冲突

**解决方案**：
理解优先级顺序：
1. Peer 匹配 > Parent Peer > Guild > Team > Account > Channel > Default
2. 更具体的匹配优先于更通用的匹配

### 问题 3：默认 Agent 被使用

**可能原因**：
1. 没有匹配的绑定规则
2. 绑定的 Agent 不存在

**解决方案**：
1. 添加默认绑定：`{"agent_id": "agent01", "match": {"channel": "weixin"}}`
2. 确保 Agent 在 `agents.list` 中定义

## 最佳实践

### 1. 明确的命名约定
- Agent 名称：`{用途}-{环境}`，如 `support-prod`, `assistant-dev`
- Channel 名称：使用平台标准名称，如 `weixin`, `telegram`, `discord`

### 2. 渐进式配置
1. 先配置简单的 channel 绑定
2. 逐步添加 peer、guild 等更具体的绑定
3. 使用默认绑定作为回退

### 3. 文档化绑定规则
```json
{
  "bindings": [
    {
      "agent_id": "agent01",
      "match": {
        "channel": "weixin"
      },
      "_comment": "所有微信消息使用 agent01 处理"
    }
  ]
}
```

### 4. 测试所有场景
- 测试直接消息
- 测试群组消息
- 测试不同用户
- 测试边缘情况

## 总结

通过 `bindings` 配置，你可以灵活地将不同 Channel 的消息路由到不同的 Agent：

1. **简单绑定**：整个 Channel 使用一个 Agent
2. **精细绑定**：按用户、群组、时间等条件路由
3. **优先级系统**：确保最具体的规则优先匹配
4. **回退机制**：默认 Agent 确保所有消息都有处理者

配置示例：
```json
{
  "agents": {
    "list": [
      {"name": "agent01", "model_name": "deepseek-chat"}
    ]
  },
  "bindings": [
    {
      "agent_id": "agent01",
      "match": {
        "channel": "weixin"
      }
    }
  ]
}
```

这样，所有从 weixin 通道来的消息都会由 agent01 处理。