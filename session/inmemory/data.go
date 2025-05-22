package inmemory

import (
	"context"
	"sync"
)

// sessionData is the implementation of the Session interface.
type sessionData struct {
	Id       string
	Data     map[string]string
	FlashOld map[string]string
	FlashNew map[string]string
	mu       sync.RWMutex
}

func (s *sessionData) Authenticate(userId string) {
	s.Put("user_id", userId)
}

func (s *sessionData) Authenticated() (string, bool) {
	v, ok := s.Get("user_id")
	return v, ok
}

func (s *sessionData) IsAuthenticated() bool {
	_, ok := s.Get("user_id")
	return ok
}

// Get retrieves a value from the sessions by key.
func (s *sessionData) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.Data[key]
	return val, ok
}

// Put stores a value in the sessions by key.
func (s *sessionData) Put(key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Data[key] = value
}

// Forget removes a key from the sessions.
func (s *sessionData) Forget(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Data, key)
}

// ID returns the sessions ID.
func (s *sessionData) ID() string {
	return s.Id
}

// Flash adds a flash message to the sessions.
func (s *sessionData) Flash(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.FlashOld[key] = value
	s.FlashNew[key] = value
}

// FlashMessages returns all flash messages from the previous request.
func (s *sessionData) FlashMessages() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.FlashOld
}

// Save persists the current sessions state to the underlying store.
func (s *sessionData) Save(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.FlashOld = s.FlashNew
	s.FlashNew = make(map[string]string)
	return nil
}

func (s *sessionData) Invalidate(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Id = generateSessionID()
	s.Data = make(map[string]string)
	s.FlashOld = make(map[string]string)
	s.FlashNew = make(map[string]string)
}
