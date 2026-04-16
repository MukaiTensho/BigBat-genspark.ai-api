package anthropic

import (
	"bigbat/internal/openai"
	"encoding/json"
	"fmt"
	"strings"
)

func ToOpenAI(req *MessagesRequest) (*openai.ChatCompletionRequest, error) {
	out := &openai.ChatCompletionRequest{
		Model:  strings.TrimSpace(req.Model),
		Stream: req.Stream,
	}

	if sys := normalizeSystem(req.System); sys != "" {
		out.Messages = append(out.Messages, openai.ChatMessage{Role: "system", Content: sys})
	}

	for _, m := range req.Messages {
		role := strings.TrimSpace(strings.ToLower(m.Role))
		if role != "user" && role != "assistant" && role != "system" {
			continue
		}
		content, err := normalizeMessageContent(m.Content)
		if err != nil {
			return nil, err
		}
		out.Messages = append(out.Messages, openai.ChatMessage{Role: role, Content: content})
	}

	if out.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(out.Messages) == 0 {
		return nil, fmt.Errorf("messages is required")
	}
	return out, nil
}

func normalizeSystem(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case []any:
		parts := make([]string, 0, len(x))
		for _, it := range x {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := m["type"].(string); t != "text" {
				continue
			}
			if txt, _ := m["text"].(string); strings.TrimSpace(txt) != "" {
				parts = append(parts, txt)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func normalizeMessageContent(v any) (any, error) {
	if v == nil {
		return "", nil
	}
	switch x := v.(type) {
	case string:
		return x, nil
	case []any:
		out := make([]map[string]any, 0, len(x))
		for _, it := range x {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			t, _ := m["type"].(string)
			switch t {
			case "text":
				text, _ := m["text"].(string)
				out = append(out, map[string]any{"type": "text", "text": text})
			case "image", "image_url":
				if src, ok := m["source"].(map[string]any); ok {
					if dt, _ := src["type"].(string); dt == "base64" {
						mediaType, _ := src["media_type"].(string)
						data, _ := src["data"].(string)
						if data != "" {
							if mediaType == "" {
								mediaType = "image/jpeg"
							}
							out = append(out, map[string]any{
								"type":      "image_url",
								"image_url": map[string]any{"url": "data:" + mediaType + ";base64," + data},
							})
						}
					}
				}
				if iu, ok := m["image_url"].(map[string]any); ok {
					url, _ := iu["url"].(string)
					if url != "" {
						out = append(out, map[string]any{"type": "image_url", "image_url": map[string]any{"url": url}})
					}
				}
			}
		}
		if len(out) == 0 {
			return "", nil
		}
		b, err := json.Marshal(out)
		if err != nil {
			return nil, err
		}
		var normalized any
		if err = json.Unmarshal(b, &normalized); err != nil {
			return nil, err
		}
		return normalized, nil
	default:
		return fmt.Sprintf("%v", x), nil
	}
}

func FromOpenAINonStream(model string, resp openai.ChatCompletionResponse) MessageResponse {
	text := ""
	if len(resp.Choices) > 0 {
		text = resp.Choices[0].Message.Content
	}
	id := resp.ID
	if id == "" {
		id = "msg_unknown"
	}
	if !strings.HasPrefix(id, "msg_") {
		id = "msg_" + id
	}
	if model == "" {
		model = resp.Model
	}
	return MessageResponse{
		ID:         id,
		Type:       "message",
		Role:       "assistant",
		Model:      model,
		Content:    []ContentBlock{{Type: "text", Text: text}},
		StopReason: "end_turn",
		Usage: ResponseUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}
}
