package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host                        string
	Port                        int
	Debug                       bool
	RoutePrefix                 string
	APISecrets                  []string
	GSCookies                   []string
	AdminStateFile              string
	AutoDeleteChat              bool
	RequestRateLimitPerMinute   int
	ProxyURL                    string
	RecaptchaProxyURL           string
	AutoModelChatMap            bool
	ModelChatMap                map[string]string
	SessionImageChatMap         map[string]string
	RateLimitCookieLockDuration time.Duration
	ReasoningHide               bool
	PreMessagesJSON             string
	UpstreamBaseURL             string
	RequestTimeout              time.Duration
}

func Load() (*Config, error) {
	port := getInt("PORT", 7055)
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid PORT: %d", port)
	}

	cfg := &Config{
		Host:                        getString("HOST", "0.0.0.0"),
		Port:                        port,
		Debug:                       getBool("DEBUG", false),
		RoutePrefix:                 normalizeRoutePrefix(os.Getenv("ROUTE_PREFIX")),
		APISecrets:                  splitCSV(os.Getenv("API_SECRET")),
		GSCookies:                   parseCookieEnv(os.Getenv("GS_COOKIE")),
		AdminStateFile:              strings.TrimSpace(getString("ADMIN_STATE_FILE", "./data/runtime-state.json")),
		AutoDeleteChat:              getInt("AUTO_DEL_CHAT", 0) == 1,
		RequestRateLimitPerMinute:   getInt("REQUEST_RATE_LIMIT", 60),
		ProxyURL:                    strings.TrimSpace(os.Getenv("PROXY_URL")),
		RecaptchaProxyURL:           normalizeRecaptchaURL(os.Getenv("RECAPTCHA_PROXY_URL")),
		AutoModelChatMap:            getInt("AUTO_MODEL_CHAT_MAP_TYPE", 1) == 1,
		ModelChatMap:                parseKVMap(os.Getenv("MODEL_CHAT_MAP"), false),
		SessionImageChatMap:         parseKVMap(os.Getenv("SESSION_IMAGE_CHAT_MAP"), true),
		RateLimitCookieLockDuration: time.Duration(getInt("RATE_LIMIT_COOKIE_LOCK_DURATION", 600)) * time.Second,
		ReasoningHide:               getInt("REASONING_HIDE", 0) == 1,
		PreMessagesJSON:             strings.TrimSpace(os.Getenv("PRE_MESSAGES_JSON")),
		UpstreamBaseURL:             getString("UPSTREAM_BASE_URL", "https://www.genspark.ai"),
		RequestTimeout:              time.Duration(getInt("REQUEST_TIMEOUT_SECONDS", 120)) * time.Second,
	}

	if len(cfg.GSCookies) == 0 {
		return nil, errors.New("GS_COOKIE is required")
	}
	if len(cfg.ModelChatMap) > 0 && cfg.AutoModelChatMap {
		return nil, errors.New("AUTO_MODEL_CHAT_MAP_TYPE cannot be 1 when MODEL_CHAT_MAP is configured")
	}
	if cfg.RequestRateLimitPerMinute < 0 {
		return nil, errors.New("REQUEST_RATE_LIMIT must be >= 0")
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 120 * time.Second
	}
	if strings.TrimSpace(cfg.Host) == "" {
		cfg.Host = "0.0.0.0"
	}

	return cfg, nil
}

func getInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getString(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func splitCSV(in string) []string {
	in = strings.TrimSpace(in)
	if in == "" {
		return nil
	}
	parts := strings.Split(in, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func normalizeCookies(cookies []string) []string {
	if len(cookies) == 0 {
		return nil
	}
	out := make([]string, 0, len(cookies))
	for _, c := range cookies {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !strings.Contains(c, "session_id=") {
			c = "session_id=" + c
		}
		out = append(out, c)
	}
	return out
}

func parseCookieEnv(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if len(raw) >= 2 {
		if (raw[0] == '"' && raw[len(raw)-1] == '"') || (raw[0] == '\'' && raw[len(raw)-1] == '\'') {
			raw = raw[1 : len(raw)-1]
		}
	}
	raw = strings.ReplaceAll(raw, "\\n", "\n")
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	var parts []string
	switch {
	case strings.Contains(raw, "\n"):
		parts = strings.Split(raw, "\n")
	case strings.Contains(raw, "||"):
		parts = strings.Split(raw, "||")
	case strings.Contains(raw, ",session_id="):
		segments := strings.Split(raw, ",session_id=")
		parts = make([]string, 0, len(segments))
		for i, seg := range segments {
			if i == 0 {
				parts = append(parts, seg)
				continue
			}
			parts = append(parts, "session_id="+seg)
		}
	default:
		parts = []string{raw}
	}

	return normalizeCookies(parts)
}

func parseKVMap(raw string, keyAsCookie bool) map[string]string {
	out := make(map[string]string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" || val == "" {
			continue
		}
		if keyAsCookie && !strings.Contains(key, "session_id=") {
			key = "session_id=" + key
		}
		out[key] = val
	}
	return out
}

func normalizeRecaptchaURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return ""
	}
	return strings.TrimRight(raw, "/")
}

func normalizeRoutePrefix(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	for strings.HasSuffix(raw, "/") {
		raw = strings.TrimSuffix(raw, "/")
	}
	if raw == "/" {
		return ""
	}
	return raw
}
