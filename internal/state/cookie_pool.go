package state

import (
	"errors"
	"math/rand"
	"slices"
	"strings"
	"sync"
	"time"
)

type CookiePool struct {
	mu      sync.RWMutex
	cookies []string
	locked  map[string]time.Time
}

func NewCookiePool(cookies []string) *CookiePool {
	copyCookies := make([]string, 0, len(cookies))
	for _, c := range cookies {
		if c != "" {
			copyCookies = append(copyCookies, c)
		}
	}
	return &CookiePool{
		cookies: copyCookies,
		locked:  make(map[string]time.Time),
	}
}

func (p *CookiePool) Lock(cookie string, until time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.locked[cookie] = until
}

func (p *CookiePool) Remove(cookie string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	idx := -1
	for i, c := range p.cookies {
		if c == cookie {
			idx = i
			break
		}
	}
	if idx >= 0 {
		p.cookies = append(p.cookies[:idx], p.cookies[idx+1:]...)
	}
	delete(p.locked, cookie)
}

func (p *CookiePool) Add(cookie string) {
	cookie = normalizeCookie(cookie)
	if cookie == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.cookies {
		if c == cookie {
			return
		}
	}
	p.cookies = append(p.cookies, cookie)
}

func (p *CookiePool) SetAll(cookies []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	seen := make(map[string]struct{}, len(cookies))
	out := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		cookie = normalizeCookie(cookie)
		if cookie == "" {
			continue
		}
		if _, ok := seen[cookie]; ok {
			continue
		}
		seen[cookie] = struct{}{}
		out = append(out, cookie)
	}
	p.cookies = out
	p.locked = make(map[string]time.Time)
}

func (p *CookiePool) All() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, len(p.cookies))
	copy(out, p.cookies)
	return out
}

func normalizeCookie(cookie string) string {
	cookie = strings.TrimSpace(cookie)
	if cookie == "" {
		return ""
	}
	if !strings.Contains(cookie, "session_id=") {
		cookie = "session_id=" + cookie
	}
	return cookie
}

func (p *CookiePool) Snapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for cookie, until := range p.locked {
		if now.After(until) {
			delete(p.locked, cookie)
		}
	}
	valid := make([]string, 0, len(p.cookies))
	for _, c := range p.cookies {
		until, ok := p.locked[c]
		if ok && now.Before(until) {
			continue
		}
		valid = append(valid, c)
	}
	return valid
}

func (p *CookiePool) Candidates() []string {
	valid := p.Snapshot()
	if len(valid) <= 1 {
		return valid
	}
	shuffled := slices.Clone(valid)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func (p *CookiePool) Random() (string, error) {
	candidates := p.Candidates()
	if len(candidates) == 0 {
		return "", errors.New("no valid cookies available")
	}
	return candidates[0], nil
}
