package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"sync"
)

func TestLoginRateLimit(t *testing.T) {

	loginAttemptsMu.Lock()
	loginAttempts = make(map[string]loginAttempt)
	loginAttemptsMu.Unlock()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := loginRateLimit(next)

	for i := 1, i <= maxLoginAttempts; i++ {
		req := httptest.NewRequest(
			http.MethodPost,
			"/login",
			nil,
		)

		req.RemoteAddr = "190.0.2.1:12345"

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf(
				"attempt %d: expected status %d, got %d",
				i,
				http.StatusOK,
				recorder.Code,
			)
		}
	}


	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		nil,
	)

	req.RemoteAddr = "190.0.2.1:12345"

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusTooManyRequests,
			recorder.Code,
		)
	}
}

func TestLoginRateLimitConcurrently(t *testing.T) {
	loginAttemptsMu.Lock()
	loginAttempts = make(map[string]loginAttempt)
	loginAttemptsMu.Unlock()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := loginRateLimit(next)

	const totalRequests = maxLoginAttempts + 20

	results := make(chan int, totalRequests)

	var wg sync.WaitGroup
	wg.Add(totalRequests)

	for i := 0; i < totalRequests; i++{
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(
				http.MethodPost,
				"/login",
				nil,
			)

			req.RemoteAddr = "192.0.2.1:12345"

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)
			results <- recorder.Code
		}()
	}

	wg.Wait()
	close(results)

	allowed := 0
	blocked := 0

	for status := range results {
		switch status {
		case http.StatusOK:
			allowed++
		case http.StatusTooManyRequests:
			blocked++
		default:
			t.Fatalf("unexpected status code: %d", status)
		}
	}

	if allowed != maxLoginAttempts {
		t.Fatalf(
			"expected %d allowed requests got %d",
			maxLoginAttempts,
			allowed,
		)
	}

	if blocked != totalRequests-maxLoginAttempts {
		t.Fatalf(
			"expected %d blocked requests, got %d",
			totalRequests-maxLoginAttempts,
			blocked,
		)
	}
}