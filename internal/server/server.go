package server

import (
	"bigbat/internal/config"
	"bigbat/internal/genspark"
	"bigbat/internal/recaptcha"
	"bigbat/internal/state"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"
)

func New(cfg *config.Config) (http.Handler, error) {
	gs, err := genspark.NewClient(cfg.UpstreamBaseURL, cfg.ProxyURL, cfg.RequestTimeout)
	if err != nil {
		return nil, err
	}

	app := &App{
		Config:      cfg,
		Genspark:    gs,
		Recaptcha:   recaptcha.New(cfg.RecaptchaProxyURL, cfg.RequestTimeout),
		CookiePool:  state.NewCookiePool(cfg.GSCookies),
		SessionPool: state.NewSessionManager(),
		RateLimiter: state.NewRateLimiter(),
		Logger:      log.Default(),
		StartedAt:   time.Now(),
	}

	if err = app.loadRuntimeState(); err != nil {
		return nil, fmt.Errorf("load runtime state: %w", err)
	}
	if strings.TrimSpace(cfg.AdminStateFile) != "" {
		if _, statErr := os.Stat(cfg.AdminStateFile); errors.Is(statErr, os.ErrNotExist) {
			_ = app.saveRuntimeState()
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleHealth)

	base := cfg.RoutePrefix + "/v1"
	mux.Handle(base+"/chat/completions", app.withMiddlewares(http.HandlerFunc(app.handleChatCompletions)))
	mux.Handle(base+"/chat/health", app.withMiddlewares(http.HandlerFunc(app.handleOpenAIChatHealth)))
	mux.Handle(base+"/images/generations", app.withMiddlewares(http.HandlerFunc(app.handleImagesGenerations)))
	mux.Handle(base+"/videos/generations", app.withMiddlewares(http.HandlerFunc(app.handleVideosGenerations)))
	mux.Handle(base+"/models", app.withMiddlewares(http.HandlerFunc(app.handleModels)))
	mux.Handle(base+"/messages", app.withMiddlewares(http.HandlerFunc(app.handleAnthropicMessages)))
	mux.Handle(base+"/messages/health", app.withMiddlewares(http.HandlerFunc(app.handleAnthropicMessagesHealth)))

	adminBase := cfg.RoutePrefix + "/admin"
	mux.Handle(adminBase, http.RedirectHandler(adminBase+"/ui", http.StatusFound))
	mux.Handle(adminBase+"/", http.RedirectHandler(adminBase+"/ui", http.StatusFound))
	mux.Handle(adminBase+"/state", app.withMiddlewares(http.HandlerFunc(app.handleAdminState)))
	mux.Handle(adminBase+"/cookies", app.withMiddlewares(http.HandlerFunc(app.handleAdminCookies)))
	mux.Handle(adminBase+"/cookies/health", app.withMiddlewares(http.HandlerFunc(app.handleAdminCookiesHealth)))
	mux.Handle(adminBase+"/config", app.withMiddlewares(http.HandlerFunc(app.handleAdminConfig)))
	mux.Handle(adminBase+"/models", app.withMiddlewares(http.HandlerFunc(app.handleAdminModels)))
	// Keep UI page publicly reachable so users can input API key in browser.
	mux.Handle(adminBase+"/ui", app.withRateLimit(http.HandlerFunc(app.handleAdminUI)))

	return app.withCORS(mux), nil
}

func (a *App) withMiddlewares(next http.Handler) http.Handler {
	return a.withRateLimit(a.withAuth(next))
}

func (a *App) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, proxy-secret")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.configMu.RLock()
		limit := a.Config.RequestRateLimitPerMinute
		a.configMu.RUnlock()
		if limit <= 0 {
			next.ServeHTTP(w, r)
			return
		}
		ip := clientIP(r)
		if !a.RateLimiter.Allow("REQUEST_RATE_LIMIT:"+ip, limit, time.Minute) {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"success": false,
				"message": "请求过于频繁,请稍后再试",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.configMu.RLock()
		secrets := make([]string, len(a.Config.APISecrets))
		copy(secrets, a.Config.APISecrets)
		a.configMu.RUnlock()
		if len(secrets) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" {
			token = strings.TrimSpace(r.Header.Get("x-api-key"))
		}
		if token == "" || !slices.Contains(secrets, token) {
			writeOpenAIError(w, http.StatusUnauthorized, "authorization(api-secret)校验失败", "invalid_request_error", "invalid_authorization")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":     "Big Bat",
		"upstream": a.Genspark.BaseURL(),
		"status":   "ok",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeOpenAIError(w http.ResponseWriter, status int, message, typ, code string) {
	if typ == "" {
		typ = "request_error"
	}
	if code == "" {
		code = fmt.Sprintf("%d", status)
	}
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    typ,
			"code":    code,
		},
	})
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}
