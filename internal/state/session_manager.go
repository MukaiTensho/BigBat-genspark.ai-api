package state

import "sync"

type SessionKey struct {
	Cookie string
	Model  string
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[SessionKey]string
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[SessionKey]string),
	}
}

func (m *SessionManager) Add(cookie, model, projectID string) {
	if cookie == "" || model == "" || projectID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[SessionKey{Cookie: cookie, Model: model}] = projectID
}

func (m *SessionManager) Get(cookie, model string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.sessions[SessionKey{Cookie: cookie, Model: model}]
	return v, ok
}

func (m *SessionManager) Delete(cookie, model string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, SessionKey{Cookie: cookie, Model: model})
}

func (m *SessionManager) ProjectIDsByCookie(cookie string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0)
	for key, value := range m.sessions {
		if key.Cookie == cookie {
			out = append(out, value)
		}
	}
	return out
}
