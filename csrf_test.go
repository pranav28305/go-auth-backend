package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFMiddleware(t *testing.T) {
	sessionID, csrfToken, err := createSession(42)
	if err != nil {
		t.Fatal(err)
	}
	defer deleteSession(sessionID)

	t.Run("missing token", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		})

		handler := csrfMiddleware(next)

		req := httptest.NewRequest(
			http.MethodPost,
			"/person",
			nil,
		)

		req.AddCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		})

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf(
				"expected status %d, got %d",
				http.StatusForbidden,
				recorder.Code,
			)
		}
	})

	t.Run("wrong token", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		})

		handler := csrfMiddleware(next)

		req := httptest.NewRequest(
			http.MethodPost,
			"/person",
			nil,
		)

		req.AddCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		})

		req.Header.Set("X-CSRF-Token", "wrong-token")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf(
				"expected status %d, got %d",
				http.StatusForbidden,
				recorder.Code,
			)
		}
	})

	t.Run("correct token", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := csrfMiddleware(next)

		req := httptest.NewRequest(
			http.MethodPost,
			"/person",
			nil,
		)

		req.AddCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		})

		req.Header.Set("X-CSRF-Token", csrfToken)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf(
				"expected status %d, got %d",
				http.StatusOK,
				recorder.Code,
			)
		}
	})
}
