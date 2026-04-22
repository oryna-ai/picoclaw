## 利用InboundContext.Raw字段传递路由信息

你说得对！`InboundContext`中的`Raw`字段是解决问题的关键。这是一个通用的键值对映射，可以用来传递额外信息。

### 1. Raw字段的用途分析

```go
type InboundContext struct {
    // ... 标准字段
    Raw map[string]string `json:"raw,omitempty"`  // 通用扩展字段
}
```

**设计意图**：允许传递平台特定的原始数据或扩展信息。

### 2. 修改`buildDispatchView`使用Raw字段

#### 2.1 从Raw中提取路由信息
```go
func buildDispatchView(inbound bus.InboundContext, identityLinks map[string][]string) dispatchView {
    view := dispatchView{
        Channel:   strings.ToLower(strings.TrimSpace(inbound.Channel)),
        Account:   NormalizeAccountID(inbound.Account),
        Mentioned: inbound.Mentioned,
    }

    // 标准字段处理...
    
    // 从Raw中提取自定义路由信息
    if inbound.Raw != nil {
        // 1. 直接指定agent
        if agentID, ok := inbound.Raw["agent_id"]; ok {
            // 可以存储到view的扩展字段中
            // 或者修改view结构体添加CustomAgent字段
        }
        
        // 2. 指定路由规则名称
        if ruleName, ok := inbound.Raw["dispatch_rule"]; ok {
            // 可以用于特殊路由逻辑
        }
        
        // 3. 传递话题分类信息
        if topicCategory, ok := inbound.Raw["topic_category"]; ok {
            // 可以用于基于分类的路由
        }
    }
    
    return view
}
```

#### 2.2 扩展dispatchView结构体
```go
type dispatchView struct {
    Channel   string
    Account   string
    Space     string
    Chat      string
    Topic     string
    Sender    string
    Mentioned bool
    
    // 扩展字段
    CustomAgent   string            // 从Raw中提取的agent
    CustomRules   []string          // 自定义规则
    RawMetadata   map[string]string // 原始Raw数据副本
}
```

### 3. 修改路由匹配逻辑

#### 3.1 优先使用CustomAgent
```go
func (r *RouteResolver) matchDispatchRule(view dispatchView) *config.DispatchRule {
    // 如果view指定了CustomAgent，直接使用
    if view.CustomAgent != "" {
        // 检查agent是否存在
        if r.isValidAgent(view.CustomAgent) {
            // 返回一个虚拟规则
            return &config.DispatchRule{
                Agent: view.CustomAgent,
                When:  config.DispatchSelector{}, // 空条件
            }
        }
    }
    
    // 否则正常匹配
    for i := range r.cfg.Agents.Dispatch.Rules {
        rule := &r.cfg.Agents.Dispatch.Rules[i]
        if !selectorHasAnyConstraint(rule.When) {
            continue
        }
        if ruleMatchesView(*rule, view) {
            return rule
        }
    }
    return nil
}
```

#### 3.2 支持Raw中的通配符匹配
```go
func ruleMatchesView(rule config.DispatchRule, view dispatchView) bool {
    when := normalizeDispatchSelector(rule.When)
    
    // 标准字段匹配...
    
    // 检查Raw中的自定义匹配条件
    if rule.When.RawConstraints != nil && view.RawMetadata != nil {
        for key, pattern := range rule.When.RawConstraints {
            value, ok := view.RawMetadata[key]
            if !ok {
                return false // Raw中缺少所需字段
            }
            if !matchWithWildcard(pattern, value) {
                return false // 值不匹配
            }
        }
    }
    
    return true
}
```

### 4. 配置扩展

#### 4.1 扩展DispatchSelector
```go
type DispatchSelector struct {
    Channel   string `json:"channel,omitempty"`
    Account   string `json:"account,omitempty"`
    Space     string `json:"space,omitempty"`
    Chat      string `json:"chat,omitempty"`
    Topic     string `json:"topic,omitempty"`
    Sender    string `json:"sender,omitempty"`
    Mentioned *bool  `json:"mentioned,omitempty"`
    
    // 扩展：支持Raw字段匹配
    RawConstraints map[string]string `json:"raw_constraints,omitempty"`
}
```

### 5. 实际使用示例

#### 5.1 发送消息时指定agent
```json
{
  "context": {
    "channel": "telegram",
    "chat_id": "project-team",
    "topic_id": "project-a-design",
    "sender_id": "alice123",
    "raw": {
      "agent_id": "project-a-specialist",
      "topic_category": "design",
      "priority": "high"
    }
  },
  "content": "设计文档写好了吗？"
}
```

#### 5.2 配置基于Raw字段的路由规则
```json
{
  "agents": {
    "dispatch": {
      "rules": [
        {
          "name": "high-priority-design",
          "agent": "design-specialist",
          "when": {
            "channel": "telegram",
            "raw_constraints": {
              "topic_category": "design",
              "priority": "high"
            }
          }
        },
        {
          "name": "project-a-topics",
          "agent": "project-a-specialist",
          "when": {
            "channel": "telegram",
            "raw_constraints": {
              "project": "project-a"
            }
          }
        }
      ]
    }
  }
}
```

### 6. 完整实现方案

#### 6.1 修改`buildDispatchView`
```go
func buildDispatchView(inbound bus.InboundContext, identityLinks map[string][]string) dispatchView {
    view := dispatchView{
        Channel:   strings.ToLower(strings.TrimSpace(inbound.Channel)),
        Account:   NormalizeAccountID(inbound.Account),
        Mentioned: inbound.Mentioned,
    }

    // 标准字段处理...
    
    // 处理Raw字段
    if inbound.Raw != nil {
        // 复制Raw数据
        view.RawMetadata = make(map[string]string, len(inbound.Raw))
        for k, v := range inbound.Raw {
            view.RawMetadata[k] = strings.ToLower(strings.TrimSpace(v))
        }
        
        // 提取特殊字段
        if agentID, ok := view.RawMetadata["agent_id"]; ok {
            view.CustomAgent = agentID
        }
    }
    
    return view
}
```

#### 6.2 修改`ruleMatchesView`
```go
func ruleMatchesView(rule config.DispatchRule, view dispatchView) bool {
    when := normalizeDispatchSelector(rule.When)
    
    // 标准字段匹配...
    
    // Raw字段匹配
    if len(when.RawConstraints) > 0 {
        if view.RawMetadata == nil {
            return false // 规则需要Raw字段，但view没有
        }
        
        for key, pattern := range when.RawConstraints {
            value, ok := view.RawMetadata[key]
            if !ok {
                return false // 缺少字段
            }
            if !matchRawValue(pattern, value) {
                return false // 值不匹配
            }
        }
    }
    
    return true
}

func matchRawValue(pattern, value string) bool {
    // 支持通配符
    if strings.Contains(pattern, "*") {
        rePattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*") + "$"
        matched, _ := regexp.MatchString(rePattern, value)
        return matched
    }
    return pattern == value
}
```

#### 6.3 修改`matchDispatchRule`
```go
func (r *RouteResolver) matchDispatchRule(view dispatchView) *config.DispatchRule {
    // 1. 检查是否有CustomAgent
    if view.CustomAgent != "" {
        // 验证agent是否存在
        if r.isValidAgent(view.CustomAgent) {
            return &config.DispatchRule{
                Name:  "custom-agent-override",
                Agent: view.CustomAgent,
                When:  config.DispatchSelector{},
            }
        }
    }
    
    // 2. 正常规则匹配
    if r.cfg == nil || r.cfg.Agents.Dispatch == nil || len(r.cfg.Agents.Dispatch.Rules) == 0 {
        return nil
    }

    for i := range r.cfg.Agents.Dispatch.Rules {
        rule := &r.cfg.Agents.Dispatch.Rules[i]
        if !selectorHasAnyConstraint(rule.When) {
            continue
        }
        if ruleMatchesView(*rule, view) {
            return rule
        }
    }
    return nil
}

func (r *RouteResolver) isValidAgent(agentID string) bool {
    normalized := routing.NormalizeAgentID(agentID)
    for _, a := range r.cfg.Agents.List {
        if routing.NormalizeAgentID(a.ID) == normalized {
            return true
        }
    }
    return false
}
```

### 7. 使用场景

#### 场景1：直接指定agent
```json
{
  "context": {
    "channel": "telegram",
    "raw": {"agent_id": "project-a-specialist"}
  }
}
```

#### 场景2：基于话题分类路由
```json
{
  "context": {
    "channel": "telegram",
    "topic_id": "project-a-design",
    "raw": {
      "project": "project-a",
      "category": "design",
      "language": "zh-CN"
    }
  }
}
```

配置：
```json
{
  "rules": [
    {
      "agent": "chinese-design-specialist",
      "when": {
        "channel": "telegram",
        "raw_constraints": {
          "project": "project-a",
          "category": "design",
          "language": "zh-CN"
        }
      }
    }
  ]
}
```

#### 场景3：动态优先级
```json
{
  "context": {
    "raw": {
      "priority": "urgent",
      "response_time": "<5min"
    }
  }
}
```

### 8. 优势

1. **无需修改客户端**：现有客户端可以继续工作
2. **向后兼容**：不影响现有功能
3. **高度灵活**：可以传递任意元数据
4. **易于扩展**：添加新字段不需要修改结构体
5. **性能良好**：只在需要时处理Raw字段

### 9. 总结

**利用`InboundContext.Raw`字段是最佳方案**：

1. **传递agent_id**：直接指定使用哪个agent
2. **传递分类信息**：用于基于规则的路由
3. **保持灵活性**：可以随时添加新字段
4. **最小化改动**：只需要修改`buildDispatchView`和路由匹配逻辑

**实施步骤**：
1. 扩展`dispatchView`结构体，添加`RawMetadata`字段
2. 修改`buildDispatchView`复制Raw数据
3. 扩展`DispatchSelector`支持`raw_constraints`
4. 修改`ruleMatchesView`支持Raw字段匹配
5. 在`matchDispatchRule`中支持`CustomAgent`覆盖

这样就能优雅地解决10万话题的路由问题了！