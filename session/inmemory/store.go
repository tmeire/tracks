package inmemory

import (
	"github.com/tmeire/tracks/session"
	"sync"
	"time"
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

func (s *Store) Load(id string) (session.Session, bool) {
	s.sessMu.RLock()
	defer s.sessMu.RUnlock()

	session, ok := s.sessions[id]
	return session, ok
}

func (s *Store) Create() session.Session {
	session := &sessionData{
		Id:   generateSessionID(),
		Data: make(map[string]string),
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

// randomString generates a random string of the specified length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond) // Ensure uniqueness
	}
	return string(result)
}
