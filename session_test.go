package main

import(
	"sync"
	"testing"
	"time"
)

func TestCreatesessionConcurrently(t *testing.T) {
	const goroutines = 100

	var wg sync.WaitGroup 
	wg.Add(goroutines)

	sessionIDs := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(userID int) {
			defer wg.Done()

			sessionID, _, err := createSession(userID)
			if err != nil {
				t.Errorf("createSession failed: %v", err)
				return
			}

			sessionIDs <- sessionID
		}(i)
	}

	wg.Wait()
	close(sessionIDs)

	for sessionID := range sessionIDs {
		deleteSession(sessionID)
	}
}

func TestCleaupExpiredSessiosns(t *testing.T) {
	sessionsMu.Lock()

	sessions["valid-session"] = Session{
		UserID:		1,
		ExpiresAt:	time.Now().Add(-time.Hour),
	}

	sessions["expired-session"] = Session{
		UserID:		2,
		ExpiresAt:	time.Now().Add(-time.Hour0),
	}

	sessionsMu.Unlock()

	cleanupExpiredSessions()

	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	if _, exists := sessions["expired-session"]; exists {
		t.Fatal("expired session should hvae been removed")
	}

	if _, exists := sessions["valid-session"]; exists {
		t.Fatal("valid session should still exist")
	}

	delete(sessions, "valid-session")
}