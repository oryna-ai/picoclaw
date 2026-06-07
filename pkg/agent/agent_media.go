// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/h2non/filetype"

	"github.com/oryna-ai/picoclaw/pkg/logger"
	"github.com/oryna-ai/picoclaw/pkg/media"
	"github.com/oryna-ai/picoclaw/pkg/providers"
)

// genericPlaceholderRegex matches generic media placeholders emitted by various
// channels: [image], [image: photo], [image: filename.jpg] — but NOT path tags
// like [image:/path/to/file] (path tags have no space after the colon).
var (
	imagePlaceholderRegex = regexp.MustCompile(`\[image(:\s+[^\]]*)?\]`)
	audioPlaceholderRegex = regexp.MustCompile(`\[audio(:\s+[^\]]*)?\]`)
	videoPlaceholderRegex = regexp.MustCompile(`\[video(:\s+[^\]]*)?\]`)
	filePlaceholderRegex  = regexp.MustCompile(`\[file(:\s+[^\]]*)?\]`)
)

// resolveMediaRefs resolves media:// refs in messages.
// For user messages: images get path tags only ([image:/path]) so the LLM
// can decide whether to view them via load_image or operate on the file.
// For tool messages: images are base64-encoded and appended as a synthetic
// user message only after the contiguous tool-message block ends, so we don't
// break the tool-results-must-immediately-follow-assistant constraint that
// LLM APIs enforce.
// Non-image files always get path tags regardless of role.
// Returns a new slice; original messages are not mutated.
func resolveMediaRefs(messages []providers.Message, store media.MediaStore, maxSize int) []providers.Message {
	if store == nil {
		return messages
	}

	result := make([]providers.Message, 0, len(messages))
	var pendingToolImages []string

	totalRefs := 0
	for _, m := range messages {
		for _, ref := range m.Media {
			if strings.HasPrefix(ref, "media://") {
				totalRefs++
			}
		}
	}
	if totalRefs > 0 {
		logger.InfoCF("agent", "resolveMediaRefs: starting",
			map[string]any{
				"total_refs": totalRefs,
				"messages":   len(messages),
			})
	}

	resolvedCount := 0
	for idx, m := range messages {
		// When leaving a tool-message block, flush any accumulated images
		// as a synthetic user message.
		if m.Role != "tool" && len(pendingToolImages) > 0 {
			result = append(result, providers.Message{
				Role:    "user",
				Content: "[Loaded image from tool result above]",
				Media:   pendingToolImages,
			})
			pendingToolImages = nil
		}

		if len(m.Media) == 0 {
			result = append(result, m)
			if idx == len(messages)-1 && len(pendingToolImages) > 0 {
				result = append(result, providers.Message{
					Role:    "user",
					Content: "[Loaded image from tool result above]",
					Media:   pendingToolImages,
				})
				pendingToolImages = nil
			}
			continue
		}

		msg := m
		resolved := make([]string, 0, len(m.Media))
		var pathTags []string

		for _, ref := range m.Media {
			if !strings.HasPrefix(ref, "media://") {
				resolved = append(resolved, ref)
				continue
			}

			localPath, meta, err := store.ResolveWithMeta(ref)
			if err != nil {
				logger.WarnCF("agent", "Failed to resolve media ref", map[string]any{
					"ref":   ref,
					"error": err.Error(),
				})
				continue
			}

			info, err := os.Stat(localPath)
			if err != nil {
				logger.WarnCF("agent", "Failed to stat media file", map[string]any{
					"path":  localPath,
					"error": err.Error(),
				})
				continue
			}

			mime := detectMIME(localPath, meta)
			pathTags = append(pathTags, buildPathTag(mime, localPath))

			if m.Role == "tool" && strings.HasPrefix(mime, "image/") {
				resolvedCount++
				logger.InfoCF("agent", "resolveMediaRefs: encoding image (%d/%d) size=%d path=%s",
					map[string]any{
						"index":    resolvedCount,
						"total":    totalRefs,
						"size":     info.Size(),
						"path":     localPath,
						"mime":     mime,
						"max_size": maxSize,
					})
				dataURL := encodeImageToDataURL(localPath, mime, info, maxSize)
				if dataURL != "" {
					pendingToolImages = append(pendingToolImages, dataURL)
					logger.InfoCF("agent", "resolveMediaRefs: encoded image (%d/%d) done len=%d",
						map[string]any{
							"index":       resolvedCount,
							"total":       totalRefs,
							"encoded_len": len(dataURL),
						})
				}
			}
		}

		msg.Media = resolved
		if len(pathTags) > 0 {
			msg.Content = injectPathTags(msg.Content, pathTags)
		}
		result = append(result, msg)

		// If this is the last message and we have pending images, flush them.
		if idx == len(messages)-1 && len(pendingToolImages) > 0 {
			result = append(result, providers.Message{
				Role:    "user",
				Content: "[Loaded image from tool result above]",
				Media:   pendingToolImages,
			})
			pendingToolImages = nil
		}
	}

	if totalRefs > 0 {
		logger.InfoCF("agent", "resolveMediaRefs: completed, resolved %d refs",
			map[string]any{
				"total_refs": totalRefs,
			})
	}

	return result
}

// encodeImageToDataURL base64-encodes an image file into a data URL.
// Returns empty string if the file exceeds maxSize or encoding fails.
func encodeImageToDataURL(localPath, mime string, info os.FileInfo, maxSize int) string {
	if info.Size() > int64(maxSize) {
		logger.WarnCF("agent", "Media file too large, skipping", map[string]any{
			"path":     localPath,
			"size":     info.Size(),
			"max_size": maxSize,
		})
		return ""
	}

	f, err := os.Open(localPath)
	if err != nil {
		logger.WarnCF("agent", "Failed to open media file", map[string]any{
			"path":  localPath,
			"error": err.Error(),
		})
		return ""
	}
	defer f.Close()

	prefix := "data:" + mime + ";base64,"
	encodedLen := base64.StdEncoding.EncodedLen(int(info.Size()))
	var buf bytes.Buffer
	buf.Grow(len(prefix) + encodedLen)
	buf.WriteString(prefix)

	encodeStart := time.Now()
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if _, err := io.Copy(encoder, f); err != nil {
		logger.WarnCF("agent", "Failed to encode media file", map[string]any{
			"path":  localPath,
			"error": err.Error(),
		})
		return ""
	}
	encoder.Close()
	encodeElapsed := time.Since(encodeStart)

	logger.DebugCF("agent", "encodeImageToDataURL: base64 encoding done",
		map[string]any{
			"path":        localPath,
			"file_size":   info.Size(),
			"encoded_len": buf.Len(),
			"elapsed_ms":  encodeElapsed.Milliseconds(),
		})

	return buf.String()
}

func buildArtifactTags(store media.MediaStore, refs []string) []string {
	if store == nil || len(refs) == 0 {
		return nil
	}

	tags := make([]string, 0, len(refs))
	for _, ref := range refs {
		localPath, meta, err := store.ResolveWithMeta(ref)
		if err != nil {
			continue
		}
		mime := detectMIME(localPath, meta)
		tags = append(tags, buildPathTag(mime, localPath))
	}

	return tags
}

func buildProviderAttachments(store media.MediaStore, refs []string) []providers.Attachment {
	if store == nil || len(refs) == 0 {
		return nil
	}

	attachments := make([]providers.Attachment, 0, len(refs))
	for _, ref := range refs {
		attachment := providers.Attachment{Ref: ref}
		if _, meta, err := store.ResolveWithMeta(ref); err == nil {
			attachment.Filename = meta.Filename
			attachment.ContentType = meta.ContentType
			attachment.Type = inferMediaType(meta.Filename, meta.ContentType)
		}
		attachments = append(attachments, attachment)
	}

	return attachments
}

// detectMIME determines the MIME type from metadata or magic-bytes detection.
// Returns empty string if detection fails.
func detectMIME(localPath string, meta media.MediaMeta) string {
	if meta.ContentType != "" {
		return meta.ContentType
	}
	kind, err := filetype.MatchFile(localPath)
	if err != nil || kind == filetype.Unknown {
		return ""
	}
	return kind.MIME.Value
}

// buildPathTag creates a structured tag exposing the local file path.
// Tag type is derived from MIME: [image:/path], [audio:/path], [video:/path], or [file:/path].
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

// injectPathTags replaces generic media tags in content with path-bearing versions,
// or appends if no matching generic tag is found. Channels emit a few different
// placeholder formats — [image], [image: photo], [image: filename.jpg] — so we
// match all of them via regex while leaving path tags ([image:/path]) untouched.
//
// When content is structured data (e.g., JSON from Feishu interactive cards or
// post messages), tags are only injected via placeholder replacement — never
// appended — to avoid corrupting the payload.
func injectPathTags(content string, tags []string) string {
	isStructured := looksLikeJSON(content)
	for _, tag := range tags {
		var pattern *regexp.Regexp
		switch {
		case strings.HasPrefix(tag, "[image:"):
			pattern = imagePlaceholderRegex
		case strings.HasPrefix(tag, "[audio:"):
			pattern = audioPlaceholderRegex
		case strings.HasPrefix(tag, "[video:"):
			pattern = videoPlaceholderRegex
		case strings.HasPrefix(tag, "[file:"):
			pattern = filePlaceholderRegex
		}

		if pattern != nil {
			if loc := pattern.FindStringIndex(content); loc != nil {
				content = content[:loc[0]] + tag + content[loc[1]:]
				continue
			}
		}

		if isStructured {
			content = tag + "\n" + content
			continue
		}

		if content == "" {
			content = tag
		} else {
			content += " " + tag
		}
	}
	return content
}

// injectImagePathTags replaces base64-encoded image data URLs in message Media
// with local file path tags ([image:/path]) injected into Content. This is used
// when the current model does not support vision — instead of sending raw image
// data that would hang or be rejected, we expose the file path so the LLM can
// use tools (e.g. load_image, MCP tools) to process the image.
// Returns a new slice; original messages are not mutated.
func injectImagePathTags(messages []providers.Message, store media.MediaStore) []providers.Message {
	if store == nil {
		return messages
	}

	result := make([]providers.Message, len(messages))
	copy(result, messages)

	for i, m := range result {
		if len(m.Media) == 0 {
			continue
		}

		var pathTags []string
		var descriptions []string
		resolved := make([]string, 0, len(m.Media))

		for _, ref := range m.Media {
			// Handle media:// refs: resolve to local path and inject path tag
			if strings.HasPrefix(ref, "media://") {
				localPath, meta, err := store.ResolveWithMeta(ref)
				if err != nil {
					logger.WarnCF("agent", "injectImagePathTags: failed to resolve media ref", map[string]any{
						"ref":   ref,
						"error": err.Error(),
					})
					continue
				}

				mime := detectMIME(localPath, meta)
				if strings.HasPrefix(mime, "image/") {
					pathTags = append(pathTags, buildPathTag(mime, localPath))
					filename := meta.Filename
					if filename == "" {
						filename = localPath
					}
					descriptions = append(descriptions,
						fmt.Sprintf("The user sent an image file: %s. You can use the load_image tool to load and analyze it, or use other MCP tools that support image processing. If no image processing tool is available, let the user know that you cannot process images.", filename))
				} else {
					// Non-image media: keep the ref as-is (audio/video/docs still need the ref)
					resolved = append(resolved, ref)
				}
				continue
			}

			// Handle data URLs (already resolved by resolveMediaRefs): discard them
			// since the model does not support vision. The path tag and description
			// above will guide the LLM to use tools instead.
			if strings.HasPrefix(ref, "data:") {
				continue
			}

			// Other refs: keep as-is
			resolved = append(resolved, ref)
		}

		result[i].Media = resolved
		if len(pathTags) > 0 {
			result[i].Content = injectPathTags(result[i].Content, pathTags)
			// Append a description to help the LLM understand what to do with the image
			for _, desc := range descriptions {
				if result[i].Content != "" {
					result[i].Content += "\n\n" + desc
				} else {
					result[i].Content = desc
				}
			}
		}
	}

	return result
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) > 1 && s[0] == '{'
}
