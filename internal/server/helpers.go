package server

import (
	"bigbat/internal/config"
	"bigbat/internal/genspark"
	"bigbat/internal/openai"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	chatTypeLegacy = "COPILOT_MOA_CHAT"
	chatTypeAgent  = "ai_chat"
	imageType      = "COPILOT_MOA_IMAGE"
	videoType      = "COPILOT_MOA_VIDEO"
)

type retryDecision int

const (
	retryNo retryDecision = iota
	retryWithNextCookie
	retryFatal
)

type retryResult struct {
	Decision   retryDecision
	Message    string
	StatusCode int
	LockFor    time.Duration
	Remove     bool
}

func (a *App) classifyFailure(status int, body string) retryResult {
	body = strings.TrimSpace(body)
	if genspark.IsRetiredCopilot(body) {
		return retryResult{Decision: retryFatal, Message: "retired_copilot_endpoint", StatusCode: http.StatusBadGateway}
	}
	if status == http.StatusTooManyRequests || genspark.IsRateLimit(body) {
		return retryResult{Decision: retryWithNextCookie, Message: "rate limit", LockFor: a.Config.RateLimitCookieLockDuration}
	}
	if genspark.IsFreeLimit(body) {
		return retryResult{Decision: retryWithNextCookie, Message: "daily free limit", LockFor: 24 * time.Hour}
	}
	if status == http.StatusUnauthorized || genspark.IsNotLogin(body) {
		return retryResult{Decision: retryWithNextCookie, Message: "cookie not login", Remove: true}
	}
	if genspark.IsCloudflareChallenge(body) {
		return retryResult{Decision: retryFatal, Message: "Detected Cloudflare Challenge Page", StatusCode: http.StatusBadGateway}
	}
	if genspark.IsCloudflareBlocked(body) {
		return retryResult{Decision: retryFatal, Message: "CloudFlare: Sorry, you have been blocked", StatusCode: http.StatusBadGateway}
	}
	if genspark.IsServiceUnavailablePage(body) {
		return retryResult{Decision: retryFatal, Message: "Genspark Service Unavailable", StatusCode: http.StatusBadGateway}
	}
	if genspark.IsServerOverloaded(body) {
		return retryResult{Decision: retryFatal, Message: "Server overloaded, please try again later.", StatusCode: http.StatusBadGateway}
	}
	if status >= 500 || genspark.IsServerError(body) {
		return retryResult{Decision: retryFatal, Message: "An error occurred with the current request, please try again.", StatusCode: http.StatusBadGateway}
	}
	if status >= 400 {
		msg := fmt.Sprintf("upstream request failed with status %d", status)
		return retryResult{Decision: retryFatal, Message: msg, StatusCode: http.StatusBadGateway}
	}
	return retryResult{Decision: retryNo}
}

func (a *App) processRetryResult(cookie string, rr retryResult) {
	if rr.LockFor > 0 {
		a.CookiePool.Lock(cookie, time.Now().Add(rr.LockFor))
	}
	if rr.Remove {
		a.CookiePool.Remove(cookie)
	}
}

func cloneChatRequest(src *openai.ChatCompletionRequest) (*openai.ChatCompletionRequest, error) {
	b, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dst openai.ChatCompletionRequest
	if err = json.Unmarshal(b, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

func (a *App) attachRecaptchaToken(ctx context.Context, cookie string, body map[string]any) error {
	if !a.Recaptcha.Enabled() {
		return nil
	}
	token, err := a.Recaptcha.GetToken(ctx, cookie)
	if err != nil {
		return err
	}
	if token != "" {
		body["g_recaptcha_token"] = token
	}
	return nil
}

func normalizeModelName(model string) string {
	model = strings.TrimSpace(model)
	if strings.HasPrefix(model, "deepseek") {
		model = strings.Replace(model, "deepseek", "deep-seek", 1)
	}
	model = config.ResolveModelAlias(model)
	return model
}

func removeSearchSuffix(model string) (string, bool) {
	if strings.HasSuffix(model, "-search") {
		return strings.TrimSuffix(model, "-search"), true
	}
	return model, false
}

func normalizeImageInput(ctx context.Context, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		data, err := genspark.FetchBytes(ctx, raw, 60*time.Second)
		if err != nil {
			return "", err
		}
		contentType := http.DetectContentType(data)
		if !strings.HasPrefix(contentType, "image/") {
			return "", fmt.Errorf("provided url is not an image")
		}
		return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data), nil
	}

	base64Part := raw
	if strings.Contains(raw, ";base64,") {
		parts := strings.SplitN(raw, ";base64,", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid base64 image")
		}
		base64Part = parts[1]
	}
	if _, err := base64.StdEncoding.DecodeString(base64Part); err != nil {
		return "", fmt.Errorf("invalid base64 image")
	}
	if strings.HasPrefix(raw, "data:image") {
		return raw, nil
	}
	return "data:image/jpeg;base64," + base64Part, nil
}

func randomizePairs[K comparable, V any](m map[K]V) []struct {
	Key K
	Val V
} {
	pairs := make([]struct {
		Key K
		Val V
	}, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, struct {
			Key K
			Val V
		}{Key: k, Val: v})
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(pairs), func(i, j int) {
		pairs[i], pairs[j] = pairs[j], pairs[i]
	})
	return pairs
}

func dataLine(raw string) string {
	line := strings.TrimSpace(raw)
	if strings.HasPrefix(line, "data: ") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
	}
	return line
}

func extractDetailAnswer(content string) (string, error) {
	var parsed struct {
		DetailAnswer string `json:"detailAnswer"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return "", err
	}
	return parsed.DetailAnswer, nil
}

func contentExt(contentType string) string {
	if contentType == "" {
		return "bin"
	}
	if idx := strings.Index(contentType, ";"); idx > 0 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)
	if !strings.Contains(contentType, "/") {
		return "bin"
	}
	ext := strings.TrimSpace(strings.TrimPrefix(path.Ext(contentType), "."))
	if ext != "" {
		return ext
	}
	parts := strings.Split(contentType, "/")
	if len(parts) == 2 && parts[1] != "" {
		return parts[1]
	}
	return "bin"
}

func shouldSendField(fieldName string, hideReasoning bool) bool {
	if fieldName == "session_state.answer" || fieldName == "session_state.streaming_markmap" || strings.Contains(fieldName, "session_state.streaming_detail_answer") {
		return true
	}
	if hideReasoning {
		return false
	}
	return fieldName == "session_state.answerthink_is_started" || fieldName == "session_state.answerthink" || fieldName == "session_state.answerthink_is_finished"
}
