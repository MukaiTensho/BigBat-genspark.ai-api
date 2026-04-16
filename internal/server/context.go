package server

import (
	"bigbat/internal/config"
	"bigbat/internal/genspark"
	"bigbat/internal/recaptcha"
	"bigbat/internal/state"
	"log"
	"sync"
	"time"
)

type App struct {
	Config      *config.Config
	Genspark    *genspark.Client
	Recaptcha   *recaptcha.Client
	CookiePool  *state.CookiePool
	SessionPool *state.SessionManager
	RateLimiter *state.RateLimiter
	Logger      *log.Logger
	configMu    sync.RWMutex
	StartedAt   time.Time
}
