package main 

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterPage(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

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

	body := bytes.NewBufferString(`{
		"username": "alice",
		"password": "secret123"
	}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		body,
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler := RegisterPage(db)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			recorder.Code,
		)
	}
}

func TestRegisterDuplicateUsername(t *testing.T){
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

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

	//first
	body := bytes.NewBufferString(`{
		"username": "alice",
		"password": "secret123"
	}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		body,
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler := RegisterPage(db)

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated{
		t.Fatalf(
			"first registration: expected %d, got %d",
			http.StatusCreated,
			recorder.Code,
		)
	}

	//duplicate
	body := bytes.NewBufferString(`{
		"username": "alice",
		"password": "secret123"
	}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		body,
	)

	req.Header.Set("Content-Type", "application/json")

	recorder = httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"duplicate registration: expected %d, got %d",
			http.StatusConflict,
			recorder.Code,
		)
	}
}

func TestRegistrationValidation(t *testing.T){
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

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

	handler := RegisterPage(db)

	tests := []struct {
		name 	string
		body 	string
		expectedCode int
	}{
		{
			name:		"missing username",
			body:		`{"password":"secret123"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:		"missing password",
			body:		`{"username":"alice"}`,
			expectedCode: http.StatusBadRequest,
		}
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := bytes.NewBufferString(test.body)

			req := httptest.NewRequest(
				http.MethodPost,
				"/register",
				body,
			)

			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if recorder.Code != test.expectedCode {
				t.Fatalf(
					"expected status %d, got %d",
					test.expectedCode,
					recorder.Code,
				)
			}
		})
	}
}

