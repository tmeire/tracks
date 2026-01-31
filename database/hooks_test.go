package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type HookModel struct {
	ID  int      `json:"id"`
	Name string   `json:"name"`
	Log []string `json:"-"`
}

func (m *HookModel) TableName() string { return "hook_models" }
func (m *HookModel) Fields() []string { return []string{"name"} }
func (m *HookModel) Values() []any    { return []any{m.Name} }
func (m *HookModel) Scan(ctx context.Context, schema any, row Scanner) (*HookModel, error) {
	var res HookModel
	err := row.Scan(&res.ID, &res.Name)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
func (m *HookModel) HasAutoIncrementID() bool { return true }
func (m *HookModel) GetID() any              { return m.ID }

func (m *HookModel) BeforeCreate(ctx context.Context) error {
	m.Log = append(m.Log, "BeforeCreate")
	if m.Name == "fail_before_create" {
		return fmt.Errorf("fail_before_create")
	}
	return nil
}

func (m *HookModel) AfterCreate(ctx context.Context) error {
	m.Log = append(m.Log, "AfterCreate")
	if m.Name == "fail_after_create" {
		return fmt.Errorf("fail_after_create")
	}
	return nil
}

func (m *HookModel) BeforeUpdate(ctx context.Context) error {
	m.Log = append(m.Log, "BeforeUpdate")
	if m.Name == "fail_before_update" {
		return fmt.Errorf("fail_before_update")
	}
	return nil
}

func (m *HookModel) AfterUpdate(ctx context.Context) error {
	m.Log = append(m.Log, "AfterUpdate")
	if m.Name == "fail_after_update" {
		return fmt.Errorf("fail_after_update")
	}
	return nil
}

func (m *HookModel) BeforeDelete(ctx context.Context) error {
	m.Log = append(m.Log, "BeforeDelete")
	if m.Name == "fail_before_delete" {
		return fmt.Errorf("fail_before_delete")
	}
	return nil
}

func (m *HookModel) AfterDelete(ctx context.Context) error {
	m.Log = append(m.Log, "AfterDelete")
	if m.Name == "fail_after_delete" {
		return fmt.Errorf("fail_after_delete")
	}
	return nil
}

func TestHooks(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE hook_models (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)")
	require.NoError(t, err)

	ctx := WithDB(context.Background(), db)
	repo := NewRepository[any, *HookModel](nil)

	t.Run("Create Hooks", func(t *testing.T) {
		m := &HookModel{Name: "test"}
		created, err := repo.Create(ctx, m)
		require.NoError(t, err)
		assert.Equal(t, []string{"BeforeCreate"}, m.Log)
		assert.Equal(t, []string{"AfterCreate"}, created.Log)
		assert.NotZero(t, created.ID)
	})

	t.Run("BeforeCreate Fail", func(t *testing.T) {
		m := &HookModel{Name: "fail_before_create"}
		_, err := repo.Create(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "fail_before_create", err.Error())
	})

	t.Run("AfterCreate Fail", func(t *testing.T) {
		m := &HookModel{Name: "fail_after_create"}
		_, err := repo.Create(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "fail_after_create", err.Error())
	})

	t.Run("Update Hooks", func(t *testing.T) {
		m := &HookModel{Name: "test_update"}
		created, err := repo.Create(ctx, m)
		require.NoError(t, err)
		
		created.Name = "updated"
		err = repo.Update(ctx, created)
		require.NoError(t, err)
		assert.Contains(t, created.Log, "BeforeUpdate")
		assert.Contains(t, created.Log, "AfterUpdate")
	})

	t.Run("BeforeUpdate Fail", func(t *testing.T) {
		m := &HookModel{Name: "test_before_update_fail"}
		created, err := repo.Create(ctx, m)
		require.NoError(t, err)
		
		created.Name = "fail_before_update"
		err = repo.Update(ctx, created)
		assert.Error(t, err)
		assert.Equal(t, "fail_before_update", err.Error())
	})

	t.Run("Delete Hooks", func(t *testing.T) {
		m := &HookModel{Name: "test_delete"}
		created, err := repo.Create(ctx, m)
		require.NoError(t, err)

		err = repo.Delete(ctx, created)
		require.NoError(t, err)
		assert.Contains(t, created.Log, "BeforeDelete")
		assert.Contains(t, created.Log, "AfterDelete")
	})

	t.Run("BeforeDelete Fail", func(t *testing.T) {
		m := &HookModel{Name: "test_before_delete_fail"}
		created, err := repo.Create(ctx, m)
		require.NoError(t, err)
		
		created.Name = "fail_before_delete"
		err = repo.Delete(ctx, created)
		assert.Error(t, err)
		assert.Equal(t, "fail_before_delete", err.Error())
	})

	t.Run("SkipHooks", func(t *testing.T) {
		m := &HookModel{Name: "test_skip"}
		skipCtx := SkipHooks(ctx)
		created, err := repo.Create(skipCtx, m)
		require.NoError(t, err)
		assert.Empty(t, m.Log)
		assert.Empty(t, created.Log)
		
		created.Name = "updated_skip"
		err = repo.Update(skipCtx, created)
		require.NoError(t, err)
		assert.Empty(t, created.Log)
		
		err = repo.Delete(skipCtx, created)
		require.NoError(t, err)
		assert.Empty(t, created.Log)
	})
}
