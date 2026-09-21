package auth

import (
	"sync"

	"golang.org/x/time/rate"
)

type LimiterManager struct {
	mu           sync.RWMutex
	limiters     map[string]*rate.Limiter
	defaultRPS   float64
	defaultBurst int
}

func NewLimiterManager(defaultRPS float64, defaultBurst int) *LimiterManager {
	return &LimiterManager{
		limiters:     make(map[string]*rate.Limiter),
		defaultRPS:   defaultRPS,
		defaultBurst: defaultBurst,
	}
}

func (m *LimiterManager) Allow(username string) bool {
	m.mu.RLock()
	limiter, exists := m.limiters[username]
	m.mu.RUnlock()

	// Lazy allocation: Create the limiter if the user is seen for the first time
	if !exists {
		m.mu.Lock()
		// Double-check locking to prevent race conditions
		limiter, exists = m.limiters[username]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(m.defaultRPS), m.defaultBurst)
			m.limiters[username] = limiter
		}
		m.mu.Unlock()
	}

	return limiter.Allow()
}

// SetOverride allows pre-configuring specific limits for certain users.
func (m *LimiterManager) SetOverride(username string, rps float64, burst int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limiters[username] = rate.NewLimiter(rate.Limit(rps), burst)
}