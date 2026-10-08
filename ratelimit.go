package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type loginAttempt struct {
	Count int
	WindowStart time.Time
}

var(
	loginAttempts = make(map[string]loginAttempt)
	loginAttemptsMu sync.Mutex
)

const(
	maxLoginAttempts = 5
	loginWindow 	 = time.Minute
)

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func LoginRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		ip := getClientIP(r)

		loginAttemptsMu.Lock()

		attempt, exists := loginAttempts[ip]

		now := time.Now()

		if !exists || now.Sub(attempt.WindowStart) >= loginWindow {
			loginAttempts[ip] = loginAttempt{
				Count:		1,
				WindowStart: now,
			}

			loginAttemptsMu.Unlock()

			next.ServeHTTP(w, r)
			return
		}

		if attempt.Count >= maxLoginAttempts {
			loginAttemptsMu.Unlock()

			http.Error(
				w,
				"too many login attempts",
				http.StatusTooManyRequests,
			)
			return
		}

		attempt.Count++
		loginAttempts[ip] = attempt

		loginAttemptsMu.Unlock()

		next.ServeHTTP(w, r)
	})
}