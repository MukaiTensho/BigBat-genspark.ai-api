package server

import (
	"bigbat/internal/anthropic"
	"bigbat/internal/openai"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (a *App) handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req anthropic.MessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "invalid request body")
		return
	}

	oaiReq, err := anthropic.ToOpenAI(&req)
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	if req.Stream {
		a.handleAnthropicStream(w, r, oaiReq)
		return
	}
	a.handleAnthropicNonStream(w, r, oaiReq)
}

func (a *App) handleAnthropicMessagesHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cookies := a.CookiePool.All()
	if len(cookies) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"name":    "Big Bat",
			"service": "anthropic",
			"ready":   false,
			"reason":  "no cookies configured",
		})
		return
	}

	type probe struct {
		Status     string `json:"status"`
		HTTPStatus int    `json:"http_status,omitempty"`
		Message    string `json:"message,omitempty"`
	}

	results := make([]probe, 0, len(cookies))
	healthyCount := 0
	for _, cookie := range cookies {
		p := a.probeCookieByChat(cookie, false)
		if p.Status == "error" && isTransientProbeError(p.Message) {
			p = a.probeCookieByChat(cookie, false)
		}
		results = append(results, probe{Status: p.Status, HTTPStatus: p.HTTPStatus, Message: p.Message})
		if p.Status == "healthy" {
			healthyCount++
		}
	}

	ready := healthyCount > 0
	reason := "no healthy cookies for anthropic endpoint"
	if ready {
		reason = "anthropic endpoint ready"
	}

	uptime := time.Since(a.StartedAt).Round(time.Second).String()
	writeJSON(w, http.StatusOK, map[string]any{
		"name":            "Big Bat",
		"service":         "anthropic",
		"ready":           ready,
		"reason":          reason,
		"uptime":          uptime,
		"cookies_total":   len(cookies),
		"cookies_healthy": healthyCount,
		"probes":          results,
		"endpoint":        "/v1/messages",
	})
}

func (a *App) handleOpenAIChatHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	cookies := a.CookiePool.All()
	if len(cookies) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"name":    "Big Bat",
			"service": "openai_chat",
			"ready":   false,
			"reason":  "no cookies configured",
		})
		return
	}

	type probe struct {
		Status     string `json:"status"`
		HTTPStatus int    `json:"http_status,omitempty"`
		Message    string `json:"message,omitempty"`
	}

	results := make([]probe, 0, len(cookies))
	healthyCount := 0
	for _, cookie := range cookies {
		p := a.probeCookieByChat(cookie, false)
		if p.Status == "error" && isTransientProbeError(p.Message) {
			p = a.probeCookieByChat(cookie, false)
		}
		results = append(results, probe{Status: p.Status, HTTPStatus: p.HTTPStatus, Message: p.Message})
		if p.Status == "healthy" {
			healthyCount++
		}
	}

	ready := healthyCount > 0
	reason := "no healthy cookies for openai chat endpoint"
	if ready {
		reason = "openai chat endpoint ready"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":            "Big Bat",
		"service":         "openai_chat",
		"ready":           ready,
		"reason":          reason,
		"cookies_total":   len(cookies),
		"cookies_healthy": healthyCount,
		"probes":          results,
		"endpoint":        "/v1/chat/completions",
	})
}

func (a *App) handleAnthropicNonStream(w http.ResponseWriter, r *http.Request, req *openai.ChatCompletionRequest) {
	rr := responseRecorder{headers: make(http.Header)}
	a.handleChatNonStream(&rr, r, req)
	if rr.status >= 400 {
		writeAnthropicError(w, rr.status, "api_error", normalizeAnthropicErrMessage(rr.body, "upstream failed"))
		return
	}
	var chatResp openai.ChatCompletionResponse
	if err := json.Unmarshal(rr.body, &chatResp); err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "invalid upstream response")
		return
	}
	resp := anthropic.FromOpenAINonStream(req.Model, chatResp)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (a *App) handleAnthropicStream(w http.ResponseWriter, r *http.Request, req *openai.ChatCompletionRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	rr := responseRecorder{headers: make(http.Header)}
	reqClone := *req
	reqClone.Stream = true
	a.handleChatStream(&rr, r, &reqClone)

	if rr.status >= 400 {
		writeAnthropicSSEError(w, rr.status, normalizeAnthropicErrMessage(rr.body, "upstream failed"))
		return
	}

	responseID := "msg_" + openai.NewResponseID()
	created := time.Now().Unix()

	if err := writeAnthropicSSE(w, "message_start", map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            responseID,
			"type":          "message",
			"role":          "assistant",
			"model":         req.Model,
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  0,
				"output_tokens": 0,
			},
			"created": created,
		},
	}); err != nil {
		return
	}
	if err := writeAnthropicSSE(w, "content_block_start", map[string]any{
		"type":          "content_block_start",
		"index":         0,
		"content_block": map[string]any{"type": "text", "text": ""},
	}); err != nil {
		return
	}

	lines := strings.Split(string(rr.body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk openai.ChatCompletionResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta != "" {
			if err := writeAnthropicSSE(w, "content_block_delta", map[string]any{
				"type":  "content_block_delta",
				"index": 0,
				"delta": map[string]any{"type": "text_delta", "text": delta},
			}); err != nil {
				return
			}
		}
		if chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason != "" {
			if err := writeAnthropicSSE(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": 0}); err != nil {
				return
			}
			if err := writeAnthropicSSE(w, "message_delta", map[string]any{
				"type": "message_delta",
				"delta": map[string]any{
					"stop_reason":   "end_turn",
					"stop_sequence": nil,
				},
				"usage": map[string]any{"output_tokens": chunk.Usage.CompletionTokens},
			}); err != nil {
				return
			}
			if err := writeAnthropicSSE(w, "message_stop", map[string]any{"type": "message_stop"}); err != nil {
				return
			}
			return
		}
	}

	_ = writeAnthropicSSE(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
	_ = writeAnthropicSSE(w, "message_stop", map[string]any{"type": "message_stop"})
}

func writeAnthropicError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    errType,
			"message": message,
		},
	})
}

func writeAnthropicSSE(w http.ResponseWriter, event string, payload map[string]any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(b)); err != nil {
		return err
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func writeAnthropicSSEError(w http.ResponseWriter, status int, message string) {
	_ = writeAnthropicSSE(w, "error", map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    "api_error",
			"message": message,
			"status":  status,
		},
	})
}

func normalizeAnthropicErrMessage(raw []byte, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err == nil {
		if e, ok := payload["error"].(map[string]any); ok {
			if msg, _ := e["message"].(string); strings.TrimSpace(msg) != "" {
				return msg
			}
		}
	}
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return fallback
	}
	if len(s) > 200 {
		return s[:200]
	}
	return s
}

type responseRecorder struct {
	headers http.Header
	body    []byte
	status  int
}

func (r *responseRecorder) Header() http.Header {
	return r.headers
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	r.body = append(r.body, b...)
	return len(b), nil
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
}

func (r *responseRecorder) Flush() {}
