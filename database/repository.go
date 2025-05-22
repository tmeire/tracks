package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
)

// Repository provides CRUD operations for a specific model type
type Repository[T Model[T]] struct {
	db Database
}

// NewRepositoryFromContext creates a new repository for the given model type using the database from the context
func NewRepositoryFromContext[T Model[T]](ctx context.Context) *Repository[T] {
	return NewRepository[T](FromContext(ctx))
}

// NewRepository creates a new repository for the given model type
func NewRepository[T Model[T]](db Database) *Repository[T] {
	return &Repository[T]{db: db}
}

// FindAll retrieves all records of the model type from the database
func (r *Repository[T]) FindAll(ctx context.Context) ([]T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.findall")
	defer span.End()

	// GetFunc a zero value of T to access the table name and fields
	var zero T
	query := fmt.Sprintf("SELECT id, %s FROM %s",
		strings.Join(zero.Fields(), ", "),
		zero.TableName())

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		model, err := zero.Scan(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, model)
	}

	return results, nil
}

// FindByID retrieves a record by its ID
func (r *Repository[T]) FindByID(ctx context.Context, id any) (T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.findbyid")
	defer span.End()

	var zero T

	query := fmt.Sprintf("SELECT id, %s FROM %s WHERE id = ?",
		strings.Join(zero.Fields(), ", "),
		zero.TableName())

	row := r.db.QueryRowContext(ctx, query, id)

	model, err := zero.Scan(row)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		var empty T
		return empty, err
	}

	return model, nil
}

// Create inserts a new record into the database
func (r *Repository[T]) Create(ctx context.Context, model T) (T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.create")
	defer span.End()

	var zero T

	// GetFunc all fields and values
	fields := model.Fields()
	values := model.Values()

	if !model.HasAutoIncrementID() {
		fields = append([]string{"id"}, fields...)
		values = append([]any{model.GetID()}, values...)
	}

	// Create placeholders for the SQL query
	placeholders := make([]string, len(fields))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		model.TableName(),
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "))

	res, err := r.db.ExecContext(ctx, query, values...)
	if err != nil {
		return zero, err
	}

	// For auto-increment IDs, retrieve the ID from the database
	if !model.HasAutoIncrementID() {
		// For app-provided IDs, use the ID from the model
		return r.FindByID(ctx, model.GetID())
	}

	id, err := res.LastInsertId()
	if err != nil {
		return zero, err
	}
	return r.FindByID(ctx, id)
}

// Update updates an existing record in the database
func (r *Repository[T]) Update(ctx context.Context, model T) error {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.update")
	defer span.End()

	fields := model.Fields()
	values := model.Values()

	// Build SET clause for all fields
	var setClause []string
	var args []any

	for i, field := range fields {
		setClause = append(setClause, fmt.Sprintf("%s = ?", field))
		args = append(args, values[i])
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?",
		model.TableName(),
		strings.Join(setClause, ", "))

	// Add ID as the last argument for the WHERE clause
	args = append(args, model.GetID())

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// Delete removes a record from the database
func (r *Repository[T]) Delete(ctx context.Context, model T) error {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.delete")
	defer span.End()

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", model.TableName())

	_, err := r.db.ExecContext(ctx, query, model.GetID())
	return err
}

// Select creates a new QueryBuilder with the specified fields
func (r *Repository[T]) Select(fields ...string) WhereableQuery[T] {
	var zero T
	return &QueryBuilder[T]{
		repo:      r,
		fields:    fields,
		tableName: zero.TableName(),
	}
}

// FindBy retrieves records that match the given search criteria
func (r *Repository[T]) FindBy(ctx context.Context, criteria map[string]any) ([]T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.findby")
	defer span.End()

	var zero T

	// Build WHERE clause
	var whereConditions []string
	var args []any

	for field, value := range criteria {
		whereConditions = append(whereConditions, fmt.Sprintf("%s = ?", field))
		args = append(args, value)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	query := fmt.Sprintf("SELECT id, %s FROM %s %s",
		strings.Join(zero.Fields(), ", "),
		zero.TableName(),
		whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		model, err := zero.Scan(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, model)
	}

	return results, nil
}

// Count returns the total number of records in the table
func (r *Repository[T]) Count(ctx context.Context) (int, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.count")
	defer span.End()

	var zero T
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", zero.TableName())

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, fmt.Errorf("failed to get count from %s", zero.TableName())
	}

	var count int
	if err := rows.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
