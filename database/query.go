package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
)

type WhereableQuery[S Schema, T Model[S, T]] interface {
	OrderableQuery[S, T]

	Where(string, ...any) WhereableQuery[S, T]
}

type OrderDirection byte

func (d OrderDirection) String() string {
	switch d {
	case ASC:
		return "ASC"
	case DESC:
		return "DESC"
	default:
		panic("unknown order direction")
	}
}

const (
	ASC OrderDirection = iota
	DESC
)

type OrderableQuery[S Schema, T Model[S, T]] interface {
	LimitableQuery[S, T]

	Order(string, OrderDirection) OrderableQuery[S, T]
}

type LimitableQuery[S Schema, T Model[S, T]] interface {
	ExecutableQuery[S, T]

	Limit(limit int) OffsetableQuery[S, T]
}

type OffsetableQuery[S Schema, T Model[S, T]] interface {
	ExecutableQuery[S, T]

	Offset(offset int) ExecutableQuery[S, T]
}

// ExecutableQuery is an interface for queries that can be executed
type ExecutableQuery[S Schema, T Model[S, T]] interface {
	Query
	// Execute runs the query and returns the results
	Execute(ctx context.Context) ([]T, error)
	First(ctx context.Context) (T, error)
}

// Query is the base interface for all query types
type Query interface {
	// Build constructs the SQL query string and arguments
	Build() (string, []any)
}

// QueryBuilder represents a SELECT query that can be further refined
type QueryBuilder[S Schema, T Model[S, T]] struct {
	repo       *Repository[S, T]
	fields     []string
	tableName  string
	conditions []string
	args       []any
	orderBy    []string
	limit      int
	offset     int
	hasLimit   bool
	hasOffset  bool
}

// Where adds WHERE conditions to the query
func (q *QueryBuilder[S, T]) Where(condition string, args ...any) WhereableQuery[S, T] {
	q.conditions = append(q.conditions, condition)
	q.args = append(q.args, args...)
	return q
}

// Order adds ORDER BY clause to the query
func (q *QueryBuilder[S, T]) Order(orderBy string, direction OrderDirection) OrderableQuery[S, T] {
	q.orderBy = append(q.orderBy, orderBy+" "+direction.String())
	return q
}

// Limit adds LIMIT clause to the query
// TODO: figure out how to handle limit = 0 case
func (q *QueryBuilder[S, T]) Limit(limit int) OffsetableQuery[S, T] {
	q.limit = limit
	q.hasLimit = true
	return q
}

// Offset adds OFFSET clause to the query
// TODO: does it make a difference to have "OFFSET 0" or no OFFSET clause at all?
func (q *QueryBuilder[S, T]) Offset(offset int) ExecutableQuery[S, T] {
	q.offset = offset
	q.hasOffset = true
	return q
}

// Build constructs the SQL query string and arguments
func (q *QueryBuilder[S, T]) Build() (string, []any) {
	var fields string
	if len(q.fields) > 0 {
		fields = "id, " + strings.Join(q.fields, ", ")
	} else {
		fields = "*"
	}

	query := "SELECT " + fields + " FROM " + q.tableName

	if len(q.conditions) > 0 {
		query += " WHERE " + strings.Join(q.conditions, " AND ")
	}

	if len(q.orderBy) > 0 {
		query += " ORDER BY " + strings.Join(q.orderBy, ", ")
	}

	if q.hasLimit {
		query += " LIMIT " + fmt.Sprintf("%d", q.limit)
	}

	if q.hasOffset {
		query += " OFFSET " + fmt.Sprintf("%d", q.offset)
	}

	return query, q.args
}

// Execute runs the query and returns the results
func (q *QueryBuilder[S, T]) Execute(ctx context.Context) ([]T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "querybuilder.execute")
	defer span.End()

	query, args := q.Build()

	rows, err := FromContext(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zero T
	var results []T
	for rows.Next() {
		model, err := zero.Scan(ctx, q.repo.schema, rows)
		if err != nil {
			return nil, err
		}
		results = append(results, model)
	}

	return results, nil
}

// First runs the query and returns the first result
func (q *QueryBuilder[S, T]) First(ctx context.Context) (T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "querybuilder.first")
	defer span.End()

	if !q.hasLimit {
		q.Limit(1)
	}

	query, args := q.Build()

	row := FromContext(ctx).QueryRowContext(ctx, query, args...)

	var zero T
	res, err := zero.Scan(ctx, q.repo.schema, row)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return zero, err
	}
	return res, nil
}
