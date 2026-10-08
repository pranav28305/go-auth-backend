package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Session struct {
	UserID    int
	ExpiresAt time.Time
	CSRFToken string
}

var (
	sessions   = make(map[string]Session)
	sessionsMu sync.RWMutex
)

func createSession(userID int) (string, string, error) {

	bytes := make([]byte, 32)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", "", err
	}

	sessionID := hex.EncodeToString(bytes)

	csrfToken, err := createCSRFToken()
	if err != nil {
		return "", "", err
	}

	session := Session{
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CSRFToken: csrfToken,
	}

	sessionsMu.Lock()
	sessions[sessionID] = session
	sessionsMu.Unlock()

	return sessionID, csrfToken, nil
}

func getUserIDFromSession(sessionID string) (int, bool) {

	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	session, exists := sessions[sessionID]
	if !exists {
		return 0, false
	}

	if time.Now().After(session.ExpiresAt) {
		return 0, false
	}

	return session.UserID, true
}

func deleteSession(sessionID string) {

	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	delete(sessions, sessionID)
}

func createCSRFToken() (string, error) {
	bytes := make([]byte, 32)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func cleanupExpiredSessions() {
	now := time.Now()

	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	for sessionID, session := range sessions {
		if now.After(session.ExpiresAt) {
			delete(sessions, sessionID)
		}
	}
}

func startSessionCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cleanupExpiredSessions()

		case <-ctx.Done():
			return
		}
	}
}

func getCSRFTokenFromSession(sessionID string) (string, bool) {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	session, exists := sessions[sessionID]
	if !exists {
		return "", false
	}

	if time.Now().After(session.ExpiresAt) {
		return "", false
	}

	return session.CSRFToken, true
}
