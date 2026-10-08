package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "userID"

func LoginPage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var request LoginRequest

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		err := decoder.Decode(&request)

		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if request.Username == "" {
			http.Error(w, "Username is required", http.StatusBadRequest)
			return
		}

		if request.Password == "" {
			http.Error(w, "Password is required", http.StatusBadRequest)
			return
		}

		user, err := getUserByUsername(db, request.Username)

		if err != nil {

			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Invalid username or password", http.StatusUnauthorized)
				return
			}

			log.Println("Databse error:", err)
			http.Error(w, "Failed to login", http.StatusInternalServerError)
			return
		}

		err = checkPassword(request.Password, user.PasswordHash)

		if err != nil {
			http.Error(w, "Initial username or password", http.StatusUnauthorized)
			return
		}

		sessionID, csrfToken, err := createSession(user.ID)

		if err != nil {
			log.Println("Session creation error:", err)
			http.Error(w, "Failed to login", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})

		response := LoginResponse{
			User:      user,
			CSRFToken: csrfToken,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	}
}

func LogoutPage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		deleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusNoContent)
}

func ProfilePage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, ok := getUserID(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := getUserByID(db, userID)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			log.Println("Database error:", err)
			http.Error(w, "Failed to get user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func getUserID(r *http.Request) (int, bool) {
	userID, ok := r.Context().Value(userIDKey).(int)
	return userID, ok
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		fmt.Println("Method:", r.Method)
		fmt.Println("Path:", r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("session_id")

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, exists := getUserIDFromSession(cookie.Value)

		if !exists {
			http.Error(w, "Unautorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(
			r.Context(),
			userIDKey,
			userID,
		)

		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {
		case http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete:
		default:
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unautorized", http.StatusUnauthorized)
			return
		}

		expectedToken, ok := getCSRFTokenFromSession(cookie.Value)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		actualToken := r.Header.Get("X-CSRF-Token")

		if subtle.ConstantTimeCompare(
			[]byte(expectedToken),
			[]byte(actualToken),
		) != 1 {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
