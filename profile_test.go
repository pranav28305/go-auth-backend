package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"encoding/json"
)

func TestProfilePageAuthenticated(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	result, err := db.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		"alice",
		"fake-hash-for-test",
	)
	if err != nil {
		t.Fatal(err)
	}

	userID64, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	userID := int(userID64)

	sessionID, _, err := createSession(userID)
	if err != nil {
		t.Fatal(err)
	}
	defer deleteSession(sessionID)

	req := httptest.NewRequest(
		http.MethodGet,
		"/profile",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	recorder := httptest.NewRecorder()

	handler := authMiddleware(ProfilePage(db))
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var user User

	err = json.NewDecoder(recorder.Body).Decode(&user)
	if err != nil {
		t.Fatal("failed to decode response:", err)
	}

	if user.ID != userID {
		t.Fatalf(
			"expected user ID %d, got %d",
			userID,
			user.ID,
		)
	}

	if user.Username != "alice" {
		t.Fatalf(
			"expected username alice, got %s",
			user.Username,
		)
	}
}

func TestLogoutPage(t *testing.T) {
	userID := 42

	sessionID, _, err := createSession(userID)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/logout",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	recorder := httptest.NewRecorder()

	LogoutPage(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			recorder.Code,
		)
	}

	_, exists := getUserIDFromSession(sessionID)
	if exists {
		t.Fatal("session should have been deleted")
	}

	cookies := recorder.Result().Cookies()

	if len(cookies) == 0 {
		t.Fatal("expected logout response to set a cookie")
	}

	sessionCookie := cookies[0]

	if sessionCookie.Name != "session_id" {
		t.Fatalf(
			"expected cookie name session_id, got %s",
			sessionCookie.Name,
		)
	}

	if sessionCookie.MaxAge >= 0 {
		t.Fatal("expected session cookie to be deleted")
	}
}

