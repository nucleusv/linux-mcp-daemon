package auth

import (
	"sync"
	"golang.org/x/time/rate"
)

type LimiterManager struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

func NewLimiterManager() *LimiterManager {
	return &LimiterManager{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (m *LimiterManager) Allow(username string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	limiter, exists := m.limiters[username]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(5.0), 10)
		m.limiters[username] = limiter
	}
	return limiter.Allow()
}