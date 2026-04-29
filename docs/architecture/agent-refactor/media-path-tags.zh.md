# 媒体路径标签机制

## 背景

当用户发送图片给不支持视觉的模型（如 DeepSeek V4）时，原有的流程会：

1. `resolveMediaRefs` 把图片编码成 base64 data URL
2. `provider.Chat` 把 data URL 发给模型 API
3. 模型 API 不支持图片输入，既不返回 `vision_unsupported` 错误，也不返回正常响应，导致卡住/超时

## 解决方案

在 `CallLLM` 中，调用 LLM 之前检测消息是否有 media。如果当前模型不支持视觉（没有配置 `ImageModel`），**不发送 data URL**，而是把图片路径以 `[image:/path]` 标签注入到消息 Content 中。

### 流程

```
用户发图片
  ↓
resolveMediaRefs 编码成 data URL（正常流程）
  ↓
CallLLM 检测：有 media 但无 ImageModel？
  ↓ 是
injectImagePathTags 替换 data URL 为 [image:/path] 标签
  ↓
LLM 收到纯文本消息，看到 [image:/path/to/photo.jpg]
  ↓
LLM 自主决定：
  ├─ 有 load_image 等工具 → 调用工具处理图片
  └─ 没有工具 → 回复用户"我不支持图片处理"
```

### 关键改动

#### `buildPathTag` — 新增 `[image:/path]` 标签

```go
func buildPathTag(mime, localPath string) string {
    switch {
    case strings.HasPrefix(mime, "image/"):
        return "[image:" + localPath + "]"
    case strings.HasPrefix(mime, "audio/"):
        return "[audio:" + localPath + "]"
    case strings.HasPrefix(mime, "video/"):
        return "[video:" + localPath + "]"
    default:
        return "[file:" + localPath + "]"
    }
}
```

#### `injectImagePathTags` — 新增函数

将消息中的 `media://` 图片引用替换为 `[image:/path]` 标签，非图片媒体保持原样。

```go
func injectImagePathTags(messages []providers.Message, store media.MediaStore) []providers.Message
```

#### `CallLLM` — 检测并替换

```go
if hasMediaRefs(exec.messages) && p.Cfg.Agents.Defaults.ImageModel == "" {
    exec.messages = injectImagePathTags(exec.messages, p.MediaStore)
}
```

### 标签类型

| MIME 类型 | 标签格式 | 说明 |
|-----------|---------|------|
| `image/*` | `[image:/path]` | 图片文件，可通过 `load_image` 等工具处理 |
| `audio/*` | `[audio:/path]` | 音频文件，可通过转录工具处理 |
| `video/*` | `[video:/path]` | 视频文件 |
| 其他 | `[file:/path]` | 文档等文件，可通过 `read_file` 处理 |

### 与 `ImageModel` 的关系

- **配置了 `ImageModel`**：图片编码为 data URL，发给视觉模型处理
- **未配置 `ImageModel`**：图片替换为 `[image:/path]` 标签，由 LLM 通过工具处理
