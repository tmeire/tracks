package db

import (
	"context"
	"crypto/rand"
	"embed"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/session"
)

// Store implements the session.Store interface using a database
type Store struct {
	database   database.Database
	repository *database.Repository[*Store, *SessionModel]
}

//go:embed migrations
var migrations embed.FS

// NewStore creates a new database-backed session store
func NewStore(ctx context.Context, db database.Database) (*Store, error) {
	err := database.MigrateUpFS(ctx, db, database.CentralDatabase, migrations)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate session database: %w", err)
	}

	s := &Store{
		database: db,
	}
	s.repository = database.NewRepository[*Store, *SessionModel](s)

	return s, nil
}

// Load retrieves a session from the database by ID
func (s *Store) Load(ctx context.Context, id string) (session.Session, bool) {
	// Load from database
	ctx = database.WithDB(ctx, s.database)
	model, err := s.repository.FindByID(ctx, id)
	if err != nil || model == nil {
		return nil, false
	}

	// Unmarshal the data
	data, err := model.UnmarshalData()
	if err != nil {
		return nil, false
	}

	// Unmarshal the flash
	flash, err := model.UnmarshalFlash()
	if err != nil {
		return nil, false
	}

	// Create a new session data object
	sess := &sessionData{
		store:     s,
		Id:        model.ID,
		createdAt: model.CreatedAt,
		updatedAt: model.UpdatedAt,
		Data:      data,
		FlashOld:  flash,
		FlashNew:  make(map[string]string),
		mu:        sync.RWMutex{},
	}
	return sess, true
}

func (s *Store) update(ctx context.Context, d *sessionData) error {
	model := &SessionModel{
		ID:        d.Id,
		CreatedAt: d.createdAt,
		UpdatedAt: time.Now(),
	}

	d.mu.RLock()
	dataCopy := make(map[string]string, len(d.Data))
	for k, v := range d.Data {
		dataCopy[k] = v
	}
	flashCopy := make(map[string]string, len(d.FlashNew))
	for k, v := range d.FlashNew {
		flashCopy[k] = v
	}
	d.mu.RUnlock()

	// Marshal the data
	err := model.MarshalData(dataCopy)
	if err != nil {
		return err
	}

	// Marshal the data
	err = model.MarshalFlash(flashCopy)
	if err != nil {
		return err
	}

	ctx = database.WithDB(ctx, s.database)

	// Update in database
	err = s.repository.Update(ctx, model)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update session in database", "session_id", d.Id, "error", err)
		return err
	}

	// Move flash messages
	d.mu.Lock()
	d.FlashOld = d.FlashNew
	d.FlashNew = make(map[string]string)
	d.mu.Unlock()
	return nil
}

// Create creates a new session in the database
func (s *Store) Create(ctx context.Context) session.Session {
	// Generate a new session ID
	id := generateSessionID()

	// Create a new session model
	now := time.Now()
	model := &SessionModel{
		ID:        id,
		Data:      "{}",
		Flash:     "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx = database.WithDB(ctx, s.database)

	// Save to database
	_, err := s.repository.Create(ctx, model)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create session in database", "session_id", id, "error", err)
	}

	return &sessionData{
		store:     s,
		Id:        id,
		createdAt: model.CreatedAt,
		updatedAt: model.UpdatedAt,
		Data:      make(map[string]string),
		FlashOld:  make(map[string]string),
		FlashNew:  make(map[string]string),
		mu:        sync.RWMutex{},
	}
}

func (s *Store) invalidate(ctx context.Context, d *sessionData) {
	ctx = database.WithDB(ctx, s.database)

	// Delete from database
	model := &SessionModel{
		ID: d.Id,
	}
	err := s.repository.Delete(ctx, model)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to delete session from database", "session_id", d.Id, "error", err)
		span := trace.SpanFromContext(ctx)
		span.RecordError(err)
	}

	// Generate a new session ID
	d.mu.Lock()
	d.Id = generateSessionID()
	d.Data = make(map[string]string)
	d.FlashOld = make(map[string]string)
	d.FlashNew = make(map[string]string)
	d.mu.Unlock()

	// Create a new session in the database
	now := time.Now()
	newModel := &SessionModel{
		ID:        d.Id,
		Data:      "{}",
		Flash:     "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = s.repository.Create(ctx, newModel)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create new session in database after invalidation", "session_id", d.Id, "error", err)
		span := trace.SpanFromContext(ctx)
		span.RecordError(err)
	}
}


// sessionData implements the session.Session interface
type sessionData struct {
	store     *Store
	Id        string
	createdAt time.Time
	updatedAt time.Time
	Data      map[string]string
	FlashOld  map[string]string
	FlashNew  map[string]string
	mu        sync.RWMutex
}

func (s *sessionData) Authenticate(userId string) {
	s.Put("user_id", userId)
}

func (s *sessionData) Authenticated() (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.Data["user_id"]
	return v, ok
}

func (s *sessionData) IsAuthenticated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.Data["user_id"]
	return ok
}

// Get retrieves a value from the session by key
func (s *sessionData) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.Data[key]
	return val, ok
}

// Put stores a value in the session by key
func (s *sessionData) Put(key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = value
}

// Forget removes a key from the session
func (s *sessionData) Forget(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Data, key)
}

// ID returns the session ID
func (s *sessionData) ID() string {
	return s.Id
}

// Flash adds a flash message to the session
func (s *sessionData) Flash(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FlashOld[key] = value
	s.FlashNew[key] = value
}

// FlashMessages returns all flash messages from the previous request
func (s *sessionData) FlashMessages() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.FlashOld
}

// Save persists the current session state to the database
func (s *sessionData) Save(ctx context.Context) error {
	return s.store.update(ctx, s)
}

// Invalidate invalidates the session
func (s *sessionData) Invalidate(ctx context.Context) {
	s.store.invalidate(ctx, s)
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	// Use the same implementation as the inmemory store
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
