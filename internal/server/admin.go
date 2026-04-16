package server

import (
	"bigbat/internal/config"
	"bigbat/internal/genspark"
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type adminConfigRequest struct {
	APISecrets                []string          `json:"api_secrets"`
	RequestRateLimitPerMinute *int              `json:"request_rate_limit_per_minute"`
	AutoDeleteChat            *bool             `json:"auto_delete_chat"`
	AutoModelChatMap          *bool             `json:"auto_model_chat_map"`
	ModelChatMap              map[string]string `json:"model_chat_map"`
	SessionImageChatMap       map[string]string `json:"session_image_chat_map"`
	ReasoningHide             *bool             `json:"reasoning_hide"`
	PreMessagesJSON           *string           `json:"pre_messages_json"`
}

type cookieHealthItem struct {
	Index      int    `json:"index"`
	Cookie     string `json:"cookie"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Source     string `json:"source,omitempty"`
	Debug      any    `json:"debug,omitempty"`
}

type cookieProbeDebug struct {
	Stage      string `json:"stage"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Decision   string `json:"decision,omitempty"`
	Message    string `json:"message,omitempty"`
	BodySample string `json:"body_sample,omitempty"`
}

func (a *App) handleAdminState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	allCookies := a.CookiePool.All()
	a.configMu.RLock()
	cfg := *a.Config
	a.configMu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"app": map[string]any{
			"name":    "Big Bat",
			"uptime":  time.Since(a.StartedAt).Round(time.Second).String(),
			"started": a.StartedAt.Format(time.RFC3339),
		},
		"cookies": map[string]any{
			"total":  len(allCookies),
			"active": len(a.CookiePool.Snapshot()),
			"all":    allCookies,
		},
		"config": map[string]any{
			"request_rate_limit_per_minute": cfg.RequestRateLimitPerMinute,
			"auto_delete_chat":              cfg.AutoDeleteChat,
			"auto_model_chat_map":           cfg.AutoModelChatMap,
			"model_chat_map":                cfg.ModelChatMap,
			"session_image_chat_map":        cfg.SessionImageChatMap,
			"reasoning_hide":                cfg.ReasoningHide,
			"pre_messages_json":             cfg.PreMessagesJSON,
			"api_secret_count":              len(cfg.APISecrets),
		},
		"recommended_models": map[string]string{
			"chat_default":  "gpt-5-pro",
			"chat_opus46":   "claude-opus-4-6",
			"chat_sonnet46": "claude-sonnet-4-6",
		},
	})
}

func (a *App) handleAdminCookies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		all := a.CookiePool.All()
		writeJSON(w, http.StatusOK, map[string]any{
			"cookies": all,
		})
	case http.MethodPost:
		var req struct {
			Cookies []string `json:"cookies"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeOpenAIError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_request")
			return
		}
		a.CookiePool.SetAll(req.Cookies)
		a.configMu.Lock()
		a.Config.GSCookies = a.CookiePool.All()
		a.configMu.Unlock()
		_ = a.saveRuntimeState()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "total": len(a.CookiePool.All())})
	case http.MethodDelete:
		cookie := strings.TrimSpace(r.URL.Query().Get("cookie"))
		if cookie == "" {
			writeOpenAIError(w, http.StatusBadRequest, "cookie query is required", "invalid_request_error", "bad_request")
			return
		}
		a.CookiePool.Remove(cookie)
		a.configMu.Lock()
		a.Config.GSCookies = a.CookiePool.All()
		a.configMu.Unlock()
		_ = a.saveRuntimeState()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "total": len(a.CookiePool.All())})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *App) handleAdminCookiesHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	debugMode := r.URL.Query().Get("debug") == "1"
	cookies := a.CookiePool.All()
	items := make([]cookieHealthItem, len(cookies))
	if len(cookies) > 0 {
		var wg sync.WaitGroup
		// Keep health probes conservative to avoid triggering upstream risk controls.
		sem := make(chan struct{}, 1)
		for i, cookie := range cookies {
			wg.Add(1)
			go func(idx int, ck string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				items[idx] = a.probeCookieHealth(idx, ck, debugMode)
			}(i, cookie)
		}
		wg.Wait()
	}

	summary := map[string]int{
		"total":   len(items),
		"healthy": 0,
		"expired": 0,
		"limited": 0,
		"blocked": 0,
		"error":   0,
	}
	for _, item := range items {
		switch item.Status {
		case "healthy":
			summary["healthy"]++
		case "expired":
			summary["expired"]++
		case "limited":
			summary["limited"]++
		case "blocked":
			summary["blocked"]++
		default:
			summary["error"]++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"summary":       summary,
		"debug_enabled": debugMode,
		"items":         items,
	})
}

func (a *App) probeCookieHealth(index int, cookie string, debugMode bool) cookieHealthItem {
	item := cookieHealthItem{
		Index:  index + 1,
		Cookie: cookie,
		Status: "error",
	}
	var debugList []cookieProbeDebug

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	raw, status, err := a.Genspark.ModelsConfigRaw(ctx, cookie)
	item.HTTPStatus = status
	if err != nil {
		item.Message = err.Error()
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "models_config", HTTPStatus: status, Decision: "error", Message: err.Error()})
			item.Debug = debugList
		}
		return item
	}

	body := strings.TrimSpace(raw)
	if debugMode {
		debugList = append(debugList, cookieProbeDebug{Stage: "models_config", HTTPStatus: status, BodySample: shortBody(body)})
	}
	if status == http.StatusUnauthorized || status == http.StatusProxyAuthRequired || status == http.StatusUnavailableForLegalReasons {
		item.Status = "expired"
		item.Message = "登录态失效或无权限"
		item.Source = "models_config"
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "models_config", HTTPStatus: status, Decision: "expired", Message: item.Message})
			item.Debug = debugList
		}
		return item
	}
	if status == http.StatusForbidden {
		msgLower := strings.ToLower(body)
		if genspark.IsNotLogin(body) || strings.Contains(strings.ToLower(body), "not login") {
			item.Status = "expired"
			item.Message = "cookie 已失效"
			item.Source = "models_config"
			if debugMode {
				debugList = append(debugList, cookieProbeDebug{Stage: "models_config", HTTPStatus: status, Decision: "expired", Message: item.Message})
				item.Debug = debugList
			}
			return item
		}
		item.Status = "blocked"
		item.Message = "models_config 返回 403，将使用 chat 探针复核"
		item.Source = "models_config"
		if debugMode {
			reason := "403 unknown"
			if strings.Contains(msgLower, "permission") || strings.Contains(msgLower, "denied") || strings.Contains(msgLower, "forbidden") {
				reason = "403 permission/denied/forbidden"
			}
			if genspark.IsCloudflareBlocked(body) || genspark.IsCloudflareChallenge(body) || genspark.IsServiceUnavailablePage(body) {
				reason = "403 risk/challenge page"
			}
			debugList = append(debugList, cookieProbeDebug{Stage: "models_config", HTTPStatus: status, Decision: "blocked", Message: reason + ", fallback to chat_probe"})
		}

		verified := a.probeCookieByChat(cookie, debugMode)
		verified.Index = item.Index
		verified.Cookie = item.Cookie
		verified.Source = "chat_probe"
		if debugMode {
			verified.Debug = mergeDebug(debugList, verified.Debug)
		}
		if verified.Status == "healthy" {
			verified.Message = "chat 探针可用（models_config:403 不作为失效依据）"
			return verified
		}
		if verified.Status == "error" {
			if debugMode {
				item.Debug = mergeDebug(debugList, verified.Debug)
			}
			return item
		}
		return verified
	}
	if status == http.StatusTooManyRequests || genspark.IsRateLimit(body) || genspark.IsFreeLimit(body) {
		item.Status = "limited"
		item.Message = "触发频率限制或额度限制"
		item.Source = "models_config"
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "models_config", HTTPStatus: status, Decision: "limited", Message: item.Message})
			item.Debug = debugList
		}
		return item
	}
	if genspark.IsCloudflareBlocked(body) || genspark.IsCloudflareChallenge(body) || genspark.IsServiceUnavailablePage(body) {
		item.Status = "blocked"
		item.Message = "触发风控或上游阻断"
		item.Source = "models_config"
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "models_config", HTTPStatus: status, Decision: "blocked", Message: item.Message})
			item.Debug = debugList
		}
		return item
	}
	if status >= 500 || genspark.IsServerOverloaded(body) || genspark.IsServerError(body) {
		item.Status = "error"
		item.Message = "上游服务异常"
		item.Source = "models_config"
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "models_config", HTTPStatus: status, Decision: "error", Message: item.Message})
			item.Debug = debugList
		}
		return item
	}
	if genspark.IsNotLogin(body) {
		item.Status = "expired"
		item.Message = "cookie 已失效"
		item.Source = "models_config"
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "models_config", HTTPStatus: status, Decision: "expired", Message: item.Message})
			item.Debug = debugList
		}
		return item
	}

	upstreamStatus, upstreamMessage, hasStatus := parseUpstreamStatus(body)
	if hasStatus && upstreamStatus < 0 {
		msgLower := strings.ToLower(strings.TrimSpace(upstreamMessage + " " + body))
		switch {
		case strings.Contains(msgLower, "not login"),
			strings.Contains(msgLower, "login required"),
			strings.Contains(msgLower, "please login"),
			strings.Contains(msgLower, "status:-5"),
			strings.Contains(msgLower, "cookie") && strings.Contains(msgLower, "expired"):
			item.Status = "expired"
			if upstreamMessage != "" {
				item.Message = upstreamMessage
			} else {
				item.Message = "cookie 已失效"
			}
		case strings.Contains(msgLower, "forbidden"),
			strings.Contains(msgLower, "permission"),
			strings.Contains(msgLower, "denied"),
			strings.Contains(msgLower, "blocked"),
			strings.Contains(msgLower, "challenge"):
			item.Status = "blocked"
			if upstreamMessage != "" {
				item.Message = upstreamMessage
			} else {
				item.Message = "上游访问受限（非登录失效）"
			}
		default:
			item.Status = "error"
			if upstreamMessage != "" {
				item.Message = upstreamMessage
			} else {
				item.Message = "upstream status < 0"
			}
		}
		return item
	}

	if status >= 200 && status < 300 {
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "models_config", HTTPStatus: status, Decision: "healthy", Message: "models_config 可用，继续 chat_probe 复核"})
		}
		verified := a.probeCookieByChat(cookie, debugMode)
		verified.Index = item.Index
		verified.Cookie = item.Cookie
		verified.Source = "chat_probe"
		if debugMode {
			verified.Debug = mergeDebug(debugList, verified.Debug)
		}
		if verified.Status == "healthy" {
			verified.Message = "可用（models_config + chat_probe）"
		}
		return verified
	}

	item.Status = "error"
	item.Message = "未知状态"

	// Second opinion: models_config can be stricter than real chat path.
	verified := a.probeCookieByChat(cookie, debugMode)
	if verified.Status == "healthy" {
		verified.Index = item.Index
		verified.Cookie = item.Cookie
		verified.Source = "chat_probe"
		if item.Status != "healthy" {
			verified.Message = "chat 探针可用（models_config 结果不作为失效依据）"
		}
		if debugMode {
			verified.Debug = mergeDebug(debugList, verified.Debug)
		}
		return verified
	}
	if verified.Status == "limited" || verified.Status == "blocked" || verified.Status == "expired" {
		verified.Index = item.Index
		verified.Cookie = item.Cookie
		verified.Source = "chat_probe"
		if debugMode {
			verified.Debug = mergeDebug(debugList, verified.Debug)
		}
		return verified
	}
	if debugMode {
		item.Debug = debugList
	}
	return item
}

func (a *App) probeCookieByChat(cookie string, debugMode bool) cookieHealthItem {
	item := cookieHealthItem{Status: "error", Cookie: cookie}
	var debugList []cookieProbeDebug
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	body := map[string]any{
		"type":         "ai_chat",
		"user_s_input": "hi",
		"messages": []map[string]any{
			{"role": "user", "content": "hi"},
		},
		"action_params": map[string]any{
			"model":                 "gpt-5-pro",
			"request_web_knowledge": false,
		},
		"extra_data": map[string]any{
			"models":                 []string{"gpt-5-pro"},
			"run_with_another_model": false,
			"writingContent":         nil,
			"request_web_knowledge":  false,
		},
	}

	if err := a.attachRecaptchaToken(ctx, cookie, body); err != nil {
		item.Status = "error"
		item.Message = "recaptcha token 获取失败: " + err.Error()
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "chat_probe", Decision: "error", Message: item.Message})
			item.Debug = debugList
		}
		return item
	}

	raw, status, err := a.Genspark.AskAgentBody(ctx, cookie, body, "application/json")
	for attempt := 1; attempt <= 3; attempt++ {
		raw, status, err = a.Genspark.AskAgentBody(ctx, cookie, body, "application/json")
		if err == nil {
			break
		}
		if !isTransientProbeError(err.Error()) || attempt == 3 {
			break
		}
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "chat_probe", HTTPStatus: status, Decision: "retry", Message: "transient error, retrying: " + err.Error()})
		}
		time.Sleep(300 * time.Millisecond)
	}

	item.HTTPStatus = status
	if err != nil {
		item.Message = err.Error()
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "chat_probe", HTTPStatus: status, Decision: "error", Message: err.Error()})
			item.Debug = debugList
		}
		return item
	}
	if debugMode {
		debugList = append(debugList, cookieProbeDebug{Stage: "chat_probe", HTTPStatus: status, BodySample: shortBody(strings.TrimSpace(raw))})
	}

	rr := a.classifyFailure(status, raw)
	switch rr.Decision {
	case retryWithNextCookie:
		msg := strings.ToLower(rr.Message + " " + raw)
		switch {
		case strings.Contains(msg, "not login"),
			strings.Contains(msg, "login required"),
			strings.Contains(msg, "cookie not login"),
			(strings.Contains(msg, "cookie") && strings.Contains(msg, "expired")),
			status == http.StatusUnauthorized:
			item.Status = "expired"
			item.Message = "cookie 已失效"
		case status == http.StatusTooManyRequests,
			strings.Contains(msg, "rate limit"),
			strings.Contains(msg, "quota"),
			strings.Contains(msg, "free usage limit"):
			item.Status = "limited"
			item.Message = "触发频率限制或额度限制"
		default:
			item.Status = "blocked"
			item.Message = "上游拒绝请求（可能风控/权限策略）"
		}
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "chat_probe", HTTPStatus: status, Decision: item.Status, Message: item.Message})
			item.Debug = debugList
		}
		return item
	case retryFatal:
		msg := strings.ToLower(rr.Message + " " + raw)
		switch {
		case strings.Contains(msg, "not login"), strings.Contains(msg, "login required"), strings.Contains(msg, "cookie") && strings.Contains(msg, "expired"):
			item.Status = "expired"
			item.Message = "cookie 已失效"
		case strings.Contains(msg, "forbidden"), strings.Contains(msg, "permission"), strings.Contains(msg, "denied"), strings.Contains(msg, "blocked"), strings.Contains(msg, "challenge"):
			item.Status = "blocked"
			item.Message = "风控/访问策略限制"
		default:
			item.Status = "error"
			if rr.Message != "" {
				item.Message = rr.Message
			} else {
				item.Message = "chat 探针失败"
			}
		}
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "chat_probe", HTTPStatus: status, Decision: item.Status, Message: item.Message})
			item.Debug = debugList
		}
		return item
	}

	if status >= 200 && status < 300 {
		item.Status = "healthy"
		item.Message = "可用"
		if debugMode {
			debugList = append(debugList, cookieProbeDebug{Stage: "chat_probe", HTTPStatus: status, Decision: "healthy", Message: item.Message})
			item.Debug = debugList
		}
		return item
	}

	item.Status = "error"
	item.Message = "chat 探针未知状态"
	if debugMode {
		debugList = append(debugList, cookieProbeDebug{Stage: "chat_probe", HTTPStatus: status, Decision: "error", Message: item.Message})
		item.Debug = debugList
	}
	return item
}

func isTransientProbeError(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	if m == "" {
		return false
	}
	if strings.Contains(m, "eof") || strings.Contains(m, "timeout") || strings.Contains(m, "connection reset") || strings.Contains(m, "tls handshake") || strings.Contains(m, "temporary") {
		return true
	}
	return false
}

func mergeDebug(base []cookieProbeDebug, extra any) any {
	out := make([]cookieProbeDebug, 0, len(base)+2)
	out = append(out, base...)
	switch v := extra.(type) {
	case []cookieProbeDebug:
		out = append(out, v...)
	case []any:
		for _, x := range v {
			m, ok := x.(map[string]any)
			if !ok {
				continue
			}
			item := cookieProbeDebug{}
			if s, ok := m["stage"].(string); ok {
				item.Stage = s
			}
			if f, ok := m["http_status"].(float64); ok {
				item.HTTPStatus = int(f)
			}
			if s, ok := m["decision"].(string); ok {
				item.Decision = s
			}
			if s, ok := m["message"].(string); ok {
				item.Message = s
			}
			if s, ok := m["body_sample"].(string); ok {
				item.BodySample = s
			}
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func shortBody(raw string) string {
	raw = strings.ReplaceAll(raw, "\n", " ")
	raw = strings.TrimSpace(raw)
	if len(raw) <= 280 {
		return raw
	}
	return raw[:280] + "..."
}

func parseUpstreamStatus(raw string) (status int, message string, ok bool) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return 0, "", false
	}
	message, _ = payload["message"].(string)
	v, exists := payload["status"]
	if !exists {
		return 0, message, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), message, true
	case int:
		return n, message, true
	case int64:
		return int(n), message, true
	}
	return 0, message, false
}

func (a *App) handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req adminConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_request")
		return
	}

	a.configMu.Lock()
	defer a.configMu.Unlock()

	if req.APISecrets != nil {
		clean := make([]string, 0, len(req.APISecrets))
		for _, secret := range req.APISecrets {
			secret = strings.TrimSpace(secret)
			if secret != "" {
				clean = append(clean, secret)
			}
		}
		a.Config.APISecrets = clean
	}
	if req.RequestRateLimitPerMinute != nil {
		a.Config.RequestRateLimitPerMinute = max(*req.RequestRateLimitPerMinute, 0)
	}
	if req.AutoDeleteChat != nil {
		a.Config.AutoDeleteChat = *req.AutoDeleteChat
	}
	if req.AutoModelChatMap != nil {
		a.Config.AutoModelChatMap = *req.AutoModelChatMap
	}
	if req.ModelChatMap != nil {
		a.Config.ModelChatMap = req.ModelChatMap
	}
	if req.SessionImageChatMap != nil {
		norm := make(map[string]string, len(req.SessionImageChatMap))
		for k, v := range req.SessionImageChatMap {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k == "" || v == "" {
				continue
			}
			if !strings.Contains(k, "session_id=") {
				k = "session_id=" + k
			}
			norm[k] = v
		}
		a.Config.SessionImageChatMap = norm
	}
	if req.ReasoningHide != nil {
		a.Config.ReasoningHide = *req.ReasoningHide
	}
	if req.PreMessagesJSON != nil {
		a.Config.PreMessagesJSON = strings.TrimSpace(*req.PreMessagesJSON)
	}
	_ = a.saveRuntimeState()

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleAdminModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	all := modelListWithUsage(a)
	writeJSON(w, http.StatusOK, map[string]any{
		"models": all,
	})
}

func modelListWithUsage(a *App) []map[string]any {
	out := make([]map[string]any, 0, len(config.DefaultModelList)+16)
	seen := make(map[string]struct{}, len(config.DefaultModelList)+16)

	for _, model := range config.DefaultModelList {
		out = append(out, modelUsageRow(model, false))
		seen[model] = struct{}{}
	}

	for alias, target := range config.ModelAliasMap {
		if _, ok := seen[alias]; ok {
			continue
		}
		row := modelUsageRow(alias, true)
		row["maps_to"] = target
		out = append(out, row)
		seen[alias] = struct{}{}
	}

	if a != nil {
		if cookie, ok := a.CookiePool.Random(); ok == nil && strings.TrimSpace(cookie) != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			remote, status, err := a.Genspark.ModelsConfig(ctx, cookie)
			if err == nil && status >= 200 && status < 300 {
				for _, id := range extractModelNamesFromModelsConfig(remote) {
					if _, exists := seen[id]; exists {
						continue
					}
					out = append(out, map[string]any{
						"id":       id,
						"type":     modelType(id),
						"routes":   modelRoutes(id),
						"upstream": "models_config",
					})
					seen[id] = struct{}{}
				}
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i]["id"].(string) < out[j]["id"].(string)
	})
	return out
}

func modelUsageRow(model string, alias bool) map[string]any {
	routes := modelRoutes(model)
	row := map[string]any{
		"id":      model,
		"type":    modelType(model),
		"routes":  routes,
		"method":  "POST",
		"example": modelExample(model, routes),
	}
	if alias {
		row["alias"] = true
	}
	return row
}

func modelRoutes(model string) []string {
	if config.IsImageModel(model) {
		return []string{"/v1/images/generations", "/v1/chat/completions (image markdown wrapper)"}
	}
	if config.IsVideoModel(model) {
		return []string{"/v1/videos/generations"}
	}
	return []string{"/v1/chat/completions"}
}

func modelExample(model string, routes []string) map[string]any {
	if len(routes) == 0 {
		return map[string]any{}
	}
	primary := routes[0]
	switch primary {
	case "/v1/images/generations":
		return map[string]any{
			"path": primary,
			"body": map[string]any{
				"model":  model,
				"prompt": "a futuristic city in mist",
			},
		}
	case "/v1/videos/generations":
		return map[string]any{
			"path": primary,
			"body": map[string]any{
				"model":        model,
				"prompt":       "a panda riding a bike",
				"aspect_ratio": "16:9",
				"duration":     5,
				"auto_prompt":  false,
			},
		}
	default:
		return map[string]any{
			"path": "/v1/chat/completions",
			"body": map[string]any{
				"model":  model,
				"stream": false,
				"messages": []map[string]any{
					{"role": "user", "content": "hello"},
				},
			},
		}
	}
}

func extractModelNamesFromModelsConfig(payload map[string]any) []string {
	data, ok := payload["data"].(map[string]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, 64)
	seen := make(map[string]struct{}, 64)

	for _, key := range []string{"image_models", "video_models", "audio_models"} {
		items, ok := data[key].([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["name"].(string)
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}

	return out
}

func modelType(model string) string {
	if config.IsImageModel(model) {
		return "image"
	}
	if config.IsVideoModel(model) {
		return "video"
	}
	return "text"
}

func (a *App) handleAdminUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminHTML))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
