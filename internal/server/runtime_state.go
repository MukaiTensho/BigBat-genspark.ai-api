package server

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type runtimeState struct {
	Cookies             []string          `json:"cookies"`
	APISecrets          []string          `json:"api_secrets"`
	RequestRateLimit    int               `json:"request_rate_limit_per_minute"`
	AutoDeleteChat      bool              `json:"auto_delete_chat"`
	AutoModelChatMap    bool              `json:"auto_model_chat_map"`
	ModelChatMap        map[string]string `json:"model_chat_map"`
	SessionImageChatMap map[string]string `json:"session_image_chat_map"`
	ReasoningHide       bool              `json:"reasoning_hide"`
	PreMessagesJSON     string            `json:"pre_messages_json"`
}

func (a *App) loadRuntimeState() error {
	path := strings.TrimSpace(a.Config.AdminStateFile)
	if path == "" {
		return nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var st runtimeState
	if err = json.Unmarshal(body, &st); err != nil {
		return err
	}

	a.configMu.Lock()
	defer a.configMu.Unlock()

	if len(st.Cookies) > 0 {
		a.CookiePool.SetAll(st.Cookies)
		a.Config.GSCookies = a.CookiePool.All()
	}
	if st.APISecrets != nil {
		a.Config.APISecrets = st.APISecrets
	}
	if st.RequestRateLimit >= 0 {
		a.Config.RequestRateLimitPerMinute = st.RequestRateLimit
	}
	a.Config.AutoDeleteChat = st.AutoDeleteChat
	a.Config.AutoModelChatMap = st.AutoModelChatMap
	if st.ModelChatMap != nil {
		a.Config.ModelChatMap = st.ModelChatMap
	}
	if st.SessionImageChatMap != nil {
		a.Config.SessionImageChatMap = st.SessionImageChatMap
	}
	a.Config.ReasoningHide = st.ReasoningHide
	a.Config.PreMessagesJSON = st.PreMessagesJSON

	return nil
}

func (a *App) saveRuntimeState() error {
	path := strings.TrimSpace(a.Config.AdminStateFile)
	if path == "" {
		return nil
	}
	a.configMu.RLock()
	st := runtimeState{
		Cookies:             a.CookiePool.All(),
		APISecrets:          append([]string(nil), a.Config.APISecrets...),
		RequestRateLimit:    a.Config.RequestRateLimitPerMinute,
		AutoDeleteChat:      a.Config.AutoDeleteChat,
		AutoModelChatMap:    a.Config.AutoModelChatMap,
		ModelChatMap:        cloneMap(a.Config.ModelChatMap),
		SessionImageChatMap: cloneMap(a.Config.SessionImageChatMap),
		ReasoningHide:       a.Config.ReasoningHide,
		PreMessagesJSON:     a.Config.PreMessagesJSON,
	}
	a.configMu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600)
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
