package db

import (
	"encoding/json"
	"github.com/tmeire/tracks/database"
	"time"
)

// SessionModel represents a session stored in the database
type SessionModel struct {
	database.Model[*SessionModel] `tracks:"sessions"`
	ID                            string `tracks:",primarykey"`
	Data                          string // JSON-encoded session data
	Flash                         string // JSON-encoded flash data
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

// TableName returns the name of the database table for this model
func (s *SessionModel) TableName() string {
	return "sessions"
}

// Fields returns the list of field names for this model
func (s *SessionModel) Fields() []string {
	return []string{"data", "flash", "created_at", "updated_at"}
}

// Values returns the values of the fields in the same order as Fields()
func (s *SessionModel) Values() []any {
	return []any{s.Data, s.Flash, s.CreatedAt, s.UpdatedAt}
}

// Scan scans the values from a row into this model
func (s *SessionModel) Scan(row database.Scanner) (*SessionModel, error) {
	var model SessionModel
	err := row.Scan(&model.ID, &model.Data, &model.Flash, &model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// UnmarshalData unmarshals the JSON-encoded data into a map
func (s *SessionModel) UnmarshalData() (map[string]string, error) {
	var data map[string]string
	if s.Data == "" {
		return make(map[string]string), nil
	}
	err := json.Unmarshal([]byte(s.Data), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MarshalData marshals a map into JSON-encoded data
func (s *SessionModel) MarshalData(data map[string]string) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	s.Data = string(jsonData)
	return nil
}

// UnmarshalFlash unmarshals the JSON-encoded flash data into a map
func (s *SessionModel) UnmarshalFlash() (map[string]string, error) {
	var flash map[string]string
	if s.Flash == "" {
		return make(map[string]string), nil
	}
	err := json.Unmarshal([]byte(s.Flash), &flash)
	if err != nil {
		return nil, err
	}
	return flash, nil
}

// MarshalFlash marshals a map into JSON-encoded data
func (s *SessionModel) MarshalFlash(flash map[string]string) error {
	jsonData, err := json.Marshal(flash)
	if err != nil {
		return err
	}
	s.Flash = string(jsonData)
	return nil
}

// HasAutoIncrementID returns true if the ID is auto-incremented by the database
func (*SessionModel) HasAutoIncrementID() bool {
	return false
}

// GetID returns the ID of the model
func (s *SessionModel) GetID() any {
	return s.ID
}
