package middleware

import (
	"net/http"
	"sync"
	"time"
)

// LoginRateLimiter provides rate limiting specifically for login attempts
type LoginRateLimiter struct {
	attempts map[string][]time.Time
	mutex    sync.RWMutex
	maxAttempts int
	window   time.Duration
	blockDuration time.Duration
}

// NewLoginRateLimiter creates a new login rate limiter
func NewLoginRateLimiter(maxAttempts int, window time.Duration, blockDuration time.Duration) *LoginRateLimiter {
	rl := &LoginRateLimiter{
		attempts: make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		window:   window,
		blockDuration: blockDuration,
	}
	
	// Start cleanup goroutine
	go rl.cleanup()
	
	return rl
}

// IsAllowed checks if a login attempt from the given IP is allowed
func (rl *LoginRateLimiter) IsAllowed(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-rl.window)
	
	// Get attempts for this IP
	attempts := rl.attempts[ip]
	
	// Remove old attempts
	var validAttempts []time.Time
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			validAttempts = append(validAttempts, attempt)
		}
	}
	
	// Check if under limit
	if len(validAttempts) >= rl.maxAttempts {
		return false
	}
	
	// Update attempts
	rl.attempts[ip] = validAttempts
	
	return true
}

// RecordAttempt records a login attempt for the given IP
func (rl *LoginRateLimiter) RecordAttempt(ip string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	attempts := rl.attempts[ip]
	attempts = append(attempts, now)
	rl.attempts[ip] = attempts
}

// GetTimeUntilAllowed returns the time until the next login attempt is allowed
func (rl *LoginRateLimiter) GetTimeUntilAllowed(ip string) time.Duration {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	
	attempts := rl.attempts[ip]
	if len(attempts) < rl.maxAttempts {
		return 0
	}
	
	// Find the oldest attempt within the window
	now := time.Now()
	cutoff := now.Add(-rl.window)
	
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			// Time until this attempt expires
			return attempt.Add(rl.window).Sub(now)
		}
	}
	
	return 0
}

// cleanup removes old entries periodically
func (rl *LoginRateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)
		
		for ip, attempts := range rl.attempts {
			var validAttempts []time.Time
			for _, attempt := range attempts {
				if attempt.After(cutoff) {
					validAttempts = append(validAttempts, attempt)
				}
			}
			
			if len(validAttempts) == 0 {
				delete(rl.attempts, ip)
			} else {
				rl.attempts[ip] = validAttempts
			}
		}
		rl.mutex.Unlock()
	}
}

// LoginRateLimit provides rate limiting middleware for login endpoints
func LoginRateLimit(rateLimiter *LoginRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply rate limiting to POST requests (login attempts)
			if r.Method != "POST" {
				next.ServeHTTP(w, r)
				return
			}
			
			// Get client IP
			ip := getClientIP(r)
			
			// Check rate limit
			if !rateLimiter.IsAllowed(ip) {
				timeUntil := rateLimiter.GetTimeUntilAllowed(ip)
				
				if IsHTMXRequest(r) {
					w.WriteHeader(http.StatusTooManyRequests)
					w.Write([]byte(`
						<div class="bg-red-50 border border-red-200 text-red-800 p-4 rounded-lg">
							<div class="flex">
								<div class="flex-shrink-0">
									<svg class="h-5 w-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
									</svg>
								</div>
								<div class="ml-3">
									<p class="text-sm">Too many login attempts. Please try again in ` + timeUntil.String() + `.</p>
								</div>
							</div>
						</div>
					`))
				} else {
					http.Error(w, "Too many login attempts. Please try again later.", http.StatusTooManyRequests)
				}
				return
			}
			
			// Record the attempt after processing
			defer rateLimiter.RecordAttempt(ip)
			
			next.ServeHTTP(w, r)
		})
	}
}

