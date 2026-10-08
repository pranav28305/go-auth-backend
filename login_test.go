package main 

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginPage(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE users(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	password_hash, err := hashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		"alice",
		passwordHash,
	`)
	if err != nil {
		t.Fatal(err)
	}

	body  := bytes.NewBufferString(`{
		"username": "alice",
		"password": "secret123"
	}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		body,
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler  := LoginPage(db)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	cookies := recorder.Result().Cookies()

	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}

	sessionCookie := cookies[0]

	if sessionCookie.Name != "session_id" {
		t.Fatalf("expected cookie name session_id, got %s",
		sessionCookie.Name,
		)
	}

	if sessionCookie.Value == "" {
		t.Fatal("expected session cookie to have a value")
	}
}

func TestLoginWrongPassword(t *testing.T){
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE users(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	password_hash, err := hashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		"alice",
		passwordHash,
	`)
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBufferString(`{
		"username": "alice", 
		"password": "wrong-password"
	}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		body,
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler := LoginPage(db)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}

	if len(recorder.Result().Cookies()) != 0 {
		t.Fatal("did not expect a session cookie")
	}
}