package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProtectedEndpoint(t *testing.T) {
	sessionID, csrfToken, err := createSession(42)
	if err != nil {
		t.Fatal(err)
	}
	defer deleteSession(sessionID)

	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		w.WriteHeader(http.StatusOK)
	})

	handler := authMiddleware(
		csrfMiddleware(
			protectedHandler,
		),
	)

	t.Run("no session", func(t *testing.T)) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/person",
			nil,
		)

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
}

t.Run("invalid csrf token", func(t *testing.T){
	req := httptest.NewRequest(
		http.MethodPost,
		"/person",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:	"session_id",
		Value: 	sessionID,
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

t.Run("valid session and csrf token", func(t *testing.T){
	req := httptest.NewRequest(
		http.MethodPost,
		"/person",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:	"session_id",
		Value:	sessionID,
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
