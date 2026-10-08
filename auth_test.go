package main 

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthMiddlewareNoCookie(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := authMiddleware(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/profile",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Falalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}
}

func TestAuthMiddlewareValidSession(t *testing.T) {
	userID := 42

	sessionID, _, err := createSession(userID)
	if err != nil {
		t.Fatal(err)
	}

	defer deleteSession(sessionID)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, ok := getUserID(r)

		if !ok {
			t.Fatal("expected user in context")
		}

		if gotUserID != userID {
			t.Fatalf(
				"expected user ID %d, got %d",
				userID,
				gotUserID,
			)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := authMiddleware(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/profile",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:	"session_id",
		Value:	sessionID,
	})

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expecteed status %d, got %d",
			http.StatusOK, 
			recorder.Code,
		)
	}
}

func TestAuthMiddlewareUnknownSession(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := authMiddleware(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/profile",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name: 	"session_id",
		Value:  "this-session-does-not-exist",
	})

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}
}

func TestAuthMiddlewareExpiredSession(t *testing.T){
	sessionID := "expired-session"

	sessionsMu.Lock()
	sessions[sessionID] = Session{
		UserID: 	42,
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
	}
	sessionsMu.Unlock()

	defer deleteSession(sessionID)

	next := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := authMiddleware(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/profile",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:	"session_id",
		Value: 	sessionID,
	})

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}
}

