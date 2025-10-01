package database

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Repository provides CRUD operations for a specific model type
type Repository[S Schema, T Model[S, T]] struct {
	schema S
	zero   T
}

// NewRepository creates a new repository for the given model type
func NewRepository[S Schema, T Model[S, T]](schema S) *Repository[S, T] {
	return &Repository[S, T]{schema: schema}
}

// Select creates a new QueryBuilder with the specified fields
func (r *Repository[S, T]) Select(fields ...string) WhereableQuery[S, T] {
	var zero T
	if len(fields) == 0 {
		fields = zero.Fields()
	}
	return &QueryBuilder[S, T]{
		repo:      r,
		fields:    fields,
		tableName: zero.TableName(),
	}
}

// FindBy retrieves records that match the given search criteria
func (r *Repository[S, T]) FindBy(ctx context.Context, criteria map[string]any) ([]T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.findby")
	defer span.End()

	s := r.Select()
	for field, value := range criteria {
		s = s.Where(fmt.Sprintf("%s = ?", field), value)
	}
	return s.Execute(ctx)
}

// FindAll retrieves all records of the model type from the database
func (r *Repository[S, T]) FindAll(ctx context.Context) ([]T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.findall")
	defer span.End()

	return r.Select().Execute(ctx)
}

// FindByID retrieves a record by its ID
func (r *Repository[S, T]) FindByID(ctx context.Context, id any) (T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.findbyid")
	defer span.End()

	return r.Select().Where("id = ?", id).First(ctx)
}

// Count returns the total number of records in the table
func (r *Repository[S, T]) Count(ctx context.Context) (int, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.count")
	defer span.End()

	return r.Select().Count(ctx)
}

// Create inserts a new record into the database
func (r *Repository[S, T]) Create(ctx context.Context, model T) (T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.create", trace.WithAttributes(attribute.String("table", r.zero.TableName())))
	defer span.End()

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

	res, err := FromContext(ctx).ExecContext(ctx, query, values...)
	if err != nil {
		return r.zero, err
	}

	// For auto-increment IDs, retrieve the ID from the database
	if !model.HasAutoIncrementID() {
		// For app-provided IDs, use the ID from the model
		return r.FindByID(ctx, model.GetID())
	}

	id, err := res.LastInsertId()
	if err != nil {
		return r.zero, err
	}
	return r.FindByID(ctx, id)
}

// Update updates an existing record in the database
func (r *Repository[S, T]) Update(ctx context.Context, model T) error {
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

	_, err := FromContext(ctx).ExecContext(ctx, query, args...)
	return err
}

// Delete removes a record from the database
func (r *Repository[S, T]) Delete(ctx context.Context, model T) error {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.delete")
	defer span.End()

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", model.TableName())

	_, err := FromContext(ctx).ExecContext(ctx, query, model.GetID())
	return err
}
