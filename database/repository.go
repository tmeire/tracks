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

// AtomicOp represents an atomic update operation on a specific field
type AtomicOp struct {
	Field string
	Delta any // int, float64, etc.
}

// AtomicUpdate performs atomic updates on specific fields using increments/decrements
func (r *Repository[S, T]) AtomicUpdate(ctx context.Context, id any, ops ...AtomicOp) error {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.atomicupdate")
	defer span.End()

	if len(ops) == 0 {
		return nil
	}

	// Validate fields against the model definition to prevent SQL injection
	// We use the zero value of T to access the Fields() method
	allowedFields := make(map[string]bool)
	for _, f := range r.zero.Fields() {
		allowedFields[f] = true
	}

	// Build the SET clause
	var setClause []string
	var args []any

	for _, op := range ops {
		if !allowedFields[op.Field] {
			return fmt.Errorf("invalid field: %s", op.Field)
		}
		setClause = append(setClause, fmt.Sprintf("%s = %s + ?", op.Field, op.Field))
		args = append(args, op.Delta)
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?",
		r.zero.TableName(),
		strings.Join(setClause, ", "))

	// Domain-aware scoping
	if IsDomainFilteringEnabled(ctx) && !shouldSkipDomainScope(ctx) {
		if _, ok := any(r.zero).(DomainScoped); ok {
			domain := DomainFromContext(ctx)
			if domain != "" {
				query += " AND domain = ?"
				args = append(args, id, domain)
			} else {
				args = append(args, id)
			}
		} else {
			args = append(args, id)
		}
	} else {
		args = append(args, id)
	}

	_, err := FromContext(ctx).ExecContext(ctx, query, args...)
	return err
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

	if !shouldSkipHooks(ctx) {
		if h, ok := any(model).(BeforeCreateHook); ok {
			if err := h.BeforeCreate(ctx); err != nil {
				return r.zero, err
			}
		}
	}

	// Domain-aware scoping
	if IsDomainFilteringEnabled(ctx) && !shouldSkipDomainScope(ctx) {
		if ds, ok := any(model).(DomainScoped); ok {
			domain := DomainFromContext(ctx)
			if domain != "" && ds.GetDomain() == "" {
				ds.SetDomain(domain)
			}
		}
	}

	// GetFunc all fields and values
	fields := model.Fields()
	values := model.Values()

	if !model.HasAutoIncrementID() {
		fields = append([]string{"id"}, fields...)
		values = append([]any{model.GetID()}, values...)
	}

	// Domain-aware scoping
	if IsDomainFilteringEnabled(ctx) && !shouldSkipDomainScope(ctx) {
		if _, ok := any(model).(DomainScoped); ok {
			domain := DomainFromContext(ctx)
			if domain != "" {
				// Inject domain if it's missing and we're dealing with a pointer that can be modified
				// This is tricky with generics and interfaces.
				// For now, we assume models that are DomainScoped will have their domain field set
				// or we can try to set it via reflection if it's a pointer.
				// But simpler is to just include it in the query if it's part of Fields()
			}
		}
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

	var created T
	// For auto-increment IDs, retrieve the ID from the database
	if !model.HasAutoIncrementID() {
		// For app-provided IDs, use the ID from the model
		created, err = r.FindByID(ctx, model.GetID())
	} else {
		id, err2 := res.LastInsertId()
		if err2 != nil {
			return r.zero, err2
		}
		created, err = r.FindByID(ctx, id)
	}

	if err != nil {
		return r.zero, err
	}

	if !shouldSkipHooks(ctx) {
		if h, ok := any(created).(AfterCreateHook); ok {
			if err := h.AfterCreate(ctx); err != nil {
				return r.zero, err
			}
		}
	}

	return created, nil
}

// Update updates an existing record in the database
func (r *Repository[S, T]) Update(ctx context.Context, model T) error {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.update")
	defer span.End()

	if !shouldSkipHooks(ctx) {
		if h, ok := any(model).(BeforeUpdateHook); ok {
			if err := h.BeforeUpdate(ctx); err != nil {
				return err
			}
		}
	}

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

	// Domain-aware scoping
	if IsDomainFilteringEnabled(ctx) && !shouldSkipDomainScope(ctx) {
		if _, ok := any(model).(DomainScoped); ok {
			domain := DomainFromContext(ctx)
			if domain != "" {
				query += " AND domain = ?"
				args = append(args, domain)
			}
		}
	}

	_, err := FromContext(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	if !shouldSkipHooks(ctx) {
		if h, ok := any(model).(AfterUpdateHook); ok {
			if err := h.AfterUpdate(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// Delete removes a record from the database
func (r *Repository[S, T]) Delete(ctx context.Context, model T) error {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "repository.delete")
	defer span.End()

	if !shouldSkipHooks(ctx) {
		if h, ok := any(model).(BeforeDeleteHook); ok {
			if err := h.BeforeDelete(ctx); err != nil {
				return err
			}
		}
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", model.TableName())
	args := []any{model.GetID()}

	// Domain-aware scoping
	if IsDomainFilteringEnabled(ctx) && !shouldSkipDomainScope(ctx) {
		if _, ok := any(model).(DomainScoped); ok {
			domain := DomainFromContext(ctx)
			if domain != "" {
				query += " AND domain = ?"
				args = append(args, domain)
			}
		}
	}

	_, err := FromContext(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	if !shouldSkipHooks(ctx) {
		if h, ok := any(model).(AfterDeleteHook); ok {
			if err := h.AfterDelete(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}
