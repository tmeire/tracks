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
	lifetime session.Lifetime
}

func NewStore() *Store {
	return &Store{
		sessions: make(map[string]*sessionData),
		lifetime: session.Lifetime{}.Resolved(),
	}
}

// SetLifetime configures the absolute and idle timeouts of the sessions in this store.
func (s *Store) SetLifetime(l session.Lifetime) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	s.lifetime = l.Resolved()
}

func (s *Store) Load(_ context.Context, id string) (session.Session, bool) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, false
	}

	sess.mu.RLock()
	expired := s.lifetime.Expired(sess.createdAt, sess.updatedAt, time.Now())
	sess.mu.RUnlock()
	if expired {
		delete(s.sessions, id)
		return nil, false
	}
	return sess, true
}

func (s *Store) Create(_ context.Context) session.Session {
	now := time.Now()
	sess := &sessionData{
		store:     s,
		Id:        generateSessionID(),
		createdAt: now,
		updatedAt: now,
		Data:      make(map[string]string),
		FlashOld:  make(map[string]string),
		FlashNew:  make(map[string]string),
	}

	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	s.sessions[sess.Id] = sess

	return sess
}

// rekey moves a session from the old ID to its new ID.
func (s *Store) rekey(oldID string, sess *sessionData, newID string) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	delete(s.sessions, oldID)
	s.sessions[newID] = sess
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
