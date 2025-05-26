package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
)

type WhereableQuery[T Model[T]] interface {
	OrderableQuery[T]

	Where(string, ...any) WhereableQuery[T]
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

type OrderableQuery[T Model[T]] interface {
	LimitableQuery[T]

	Order(string, OrderDirection) OrderableQuery[T]
}

type LimitableQuery[T Model[T]] interface {
	ExecutableQuery[T]

	Limit(limit int) OffsetableQuery[T]
}

type OffsetableQuery[T Model[T]] interface {
	ExecutableQuery[T]

	Offset(offset int) ExecutableQuery[T]
}

// ExecutableQuery is an interface for queries that can be executed
type ExecutableQuery[T Model[T]] interface {
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
type QueryBuilder[T Model[T]] struct {
	repo       *Repository[T]
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
func (q *QueryBuilder[T]) Where(condition string, args ...any) WhereableQuery[T] {
	q.conditions = append(q.conditions, condition)
	q.args = append(q.args, args...)
	return q
}

// Order adds ORDER BY clause to the query
func (q *QueryBuilder[T]) Order(orderBy string, direction OrderDirection) OrderableQuery[T] {
	q.orderBy = append(q.orderBy, orderBy+" "+direction.String())
	return q
}

// Limit adds LIMIT clause to the query
// TODO: figure out how to handle limit = 0 case
func (q *QueryBuilder[T]) Limit(limit int) OffsetableQuery[T] {
	q.limit = limit
	q.hasLimit = true
	return q
}

// Offset adds OFFSET clause to the query
// TODO: does it make a difference to have "OFFSET 0" or no OFFSET clause at all?
func (q *QueryBuilder[T]) Offset(offset int) ExecutableQuery[T] {
	q.offset = offset
	q.hasOffset = true
	return q
}

// Build constructs the SQL query string and arguments
func (q *QueryBuilder[T]) Build() (string, []any) {
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
func (q *QueryBuilder[T]) Execute(ctx context.Context) ([]T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "querybuilder.execute")
	defer span.End()

	query, args := q.Build()

	rows, err := q.repo.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zero T
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

// First runs the query and returns the first result
func (q *QueryBuilder[T]) First(ctx context.Context) (T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(ctx, "querybuilder.first")
	defer span.End()

	if !q.hasLimit {
		q.Limit(1)
	}

	query, args := q.Build()

	row := q.repo.db.QueryRowContext(ctx, query, args...)

	var zero T
	res, err := zero.Scan(row)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return zero, err
	}
	return res, nil
}
