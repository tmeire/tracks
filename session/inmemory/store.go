package inmemory

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/tmeire/tracks/session"
)

type Store struct {
	sessions map[string]*sessionData
	sessMu   sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		sessions: make(map[string]*sessionData),
	}
}

func (s *Store) Load(_ context.Context, id string) (session.Session, bool) {
	s.sessMu.RLock()
	defer s.sessMu.RUnlock()

	session, ok := s.sessions[id]
	return session, ok
}

func (s *Store) Create(_ context.Context) session.Session {
	session := &sessionData{
		Id:       generateSessionID(),
		Data:     make(map[string]string),
		FlashOld: make(map[string]string),
		FlashNew: make(map[string]string),
	}

	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	s.sessions[session.Id] = session

	return session
}

// generateSessionID generates a random sessions ID.
func generateSessionID() string {
	// Simple implementation for demonstration
	return time.Now().Format("20060102150405") + "-" + randomString(16)
}

// randomString generates a random string of the specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	_, err := rand.Read(result)
	if err != nil {
		// Fallback to timestamp if crypto/rand fails
		panic(fmt.Sprintf("failed to generate random string: %v", err))
	}
	for i := range result {
		result[i] = charset[int(result[i])%len(charset)]
	}
	return string(result)
}
