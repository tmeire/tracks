package database

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAtomicDB for testing
type MockAtomicDB struct {
	mock.Mock
}

func (m *MockAtomicDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	calledArgs := []any{ctx, query}
	calledArgs = append(calledArgs, args...)
	called := m.Called(calledArgs...)
	if rows, ok := called.Get(0).(*sql.Rows); ok {
		return rows, called.Error(1)
	}
	return nil, called.Error(1)
}

func (m *MockAtomicDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	calledArgs := []any{ctx, query}
	calledArgs = append(calledArgs, args...)
	called := m.Called(calledArgs...)
	return called.Get(0).(*sql.Row)
}

func (m *MockAtomicDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// We pass args as a slice to match how we will set up the expectation
	called := m.Called(ctx, query, args)
	if res, ok := called.Get(0).(sql.Result); ok {
		return res, called.Error(1)
	}
	return nil, called.Error(1)
}

func (m *MockAtomicDB) Close() error {
	return m.Called().Error(0)
}

// TestAtomicUpdateModel local definition to ensure independence
type TestAtomicUpdateModel struct {
	ID    int
	Count int
	Score float64
}

func (m TestAtomicUpdateModel) TableName() string { return "test_models" }
func (m TestAtomicUpdateModel) Fields() []string  { return []string{"count", "score"} }
func (m TestAtomicUpdateModel) Values() []any     { return []any{m.Count, m.Score} }
func (m TestAtomicUpdateModel) Scan(ctx context.Context, s any, r Scanner) (TestAtomicUpdateModel, error) {
	return m, nil
}
func (m TestAtomicUpdateModel) HasAutoIncrementID() bool { return true }
func (m TestAtomicUpdateModel) GetID() any               { return m.ID }

func TestAtomicUpdate(t *testing.T) {
	t.Run("Valid Single Update", func(t *testing.T) {
		mockDB := new(MockAtomicDB)
		ctx := WithDB(context.Background(), mockDB)
		repo := NewRepository[any, TestAtomicUpdateModel](nil)

		id := 1
		delta := 5

		expectedQuery := "UPDATE test_models SET count = count + ? WHERE id = ?"
		expectedArgs := []any{delta, id}

		mockDB.On("ExecContext", mock.Anything, expectedQuery, expectedArgs).Return(nil, nil)

		err := repo.AtomicUpdate(ctx, id, AtomicOp{Field: "count", Delta: delta})
		assert.NoError(t, err)
		mockDB.AssertExpectations(t)
	})

	t.Run("Valid Multiple Updates", func(t *testing.T) {
		mockDB := new(MockAtomicDB)
		ctx := WithDB(context.Background(), mockDB)
		repo := NewRepository[any, TestAtomicUpdateModel](nil)

		id := 2
		delta1 := -1
		delta2 := 1.5

		expectedQuery := "UPDATE test_models SET count = count + ?, score = score + ? WHERE id = ?"
		expectedArgs := []any{delta1, delta2, id}

		mockDB.On("ExecContext", mock.Anything, expectedQuery, expectedArgs).Return(nil, nil)

		err := repo.AtomicUpdate(ctx, id,
			AtomicOp{Field: "count", Delta: delta1},
			AtomicOp{Field: "score", Delta: delta2},
		)
		assert.NoError(t, err)
		mockDB.AssertExpectations(t)
	})

	t.Run("Invalid Field", func(t *testing.T) {
		mockDB := new(MockAtomicDB)
		ctx := WithDB(context.Background(), mockDB)
		repo := NewRepository[any, TestAtomicUpdateModel](nil)

		id := 3
		err := repo.AtomicUpdate(ctx, id, AtomicOp{Field: "invalid_field", Delta: 1})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid field")
		mockDB.AssertNotCalled(t, "ExecContext")
	})

	t.Run("No Ops", func(t *testing.T) {
		mockDB := new(MockAtomicDB)
		ctx := WithDB(context.Background(), mockDB)
		repo := NewRepository[any, TestAtomicUpdateModel](nil)

		err := repo.AtomicUpdate(ctx, 1) // No ops
		assert.NoError(t, err)
		mockDB.AssertNotCalled(t, "ExecContext")
	})
}