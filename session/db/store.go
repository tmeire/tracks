package db

import (
	"context"
	"embed"
	"fmt"
	"go.opentelemetry.io/otel/trace"
	"time"

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
	}
	return sess, true
}

func (s *Store) update(ctx context.Context, d *sessionData) error {
	model := &SessionModel{
		ID:        d.Id,
		CreatedAt: d.createdAt,
		UpdatedAt: time.Now(),
	}

	// Marshal the data
	err := model.MarshalData(d.Data)
	if err != nil {
		return err
	}

	// Marshal the data
	err = model.MarshalFlash(d.FlashNew)
	if err != nil {
		return err
	}

	ctx = database.WithDB(ctx, s.database)

	// Update in database
	err = s.repository.Update(ctx, model)
	if err != nil {
		return err
	}

	// Move flash messages
	d.FlashOld = d.FlashNew
	d.FlashNew = make(map[string]string)
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
		// Log the error but continue with in-memory session
		// This ensures the application doesn't break if the database is unavailable
		// TODO: Add proper logging
		fmt.Println("Error creating session in database:", err)
	}

	return &sessionData{
		store:     s,
		Id:        id,
		createdAt: model.CreatedAt,
		updatedAt: model.UpdatedAt,
		Data:      make(map[string]string),
		FlashOld:  make(map[string]string),
		FlashNew:  make(map[string]string),
	}
}

func (s *Store) invalidate(ctx context.Context, d *sessionData) {
	ctx = database.WithDB(ctx, s.database)

	// DeleteFunc from database
	model := &SessionModel{
		ID: d.Id,
	}
	err := s.repository.Delete(ctx, model)
	if err != nil {
		span := trace.SpanFromContext(ctx)
		span.RecordError(err)
	}

	// Generate a new session ID
	d.Id = generateSessionID()
	d.Data = make(map[string]string)
	d.FlashOld = make(map[string]string)
	d.FlashNew = make(map[string]string)

	// Create a new session in the database
	now := time.Now()
	newModel := &SessionModel{
		ID:        d.Id,
		Data:      "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = s.repository.Create(ctx, newModel)
	if err != nil {
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

// Get retrieves a value from the session by key
func (s *sessionData) Get(key string) (string, bool) {
	val, ok := s.Data[key]
	return val, ok
}

// Put stores a value in the session by key
func (s *sessionData) Put(key string, value string) {
	s.Data[key] = value
}

// Forget removes a key from the session
func (s *sessionData) Forget(key string) {
	delete(s.Data, key)
}

// ID returns the session ID
func (s *sessionData) ID() string {
	return s.Id
}

// Flash adds a flash message to the session
func (s *sessionData) Flash(key, value string) {
	s.FlashOld[key] = value
	s.FlashNew[key] = value
}

// FlashMessages returns all flash messages from the previous request
func (s *sessionData) FlashMessages() map[string]string {
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
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond) // Ensure uniqueness
	}
	return string(result)
}
