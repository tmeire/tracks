package featureflags

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tmeire/tracks/database"
)

type PrincipalType string

const (
	PrincipalGlobal PrincipalType = "global"
	PrincipalTenant PrincipalType = "tenant"
	PrincipalRole   PrincipalType = "role"
	PrincipalUser   PrincipalType = "user"
)

type Principal struct {
	Type PrincipalType
	ID   string // empty for global
}

type Principals struct {
	UserID   *string
	TenantID *string
	RoleIDs  []string
}

type Override struct {
	FlagKey       string
	PrincipalType PrincipalType
	PrincipalID   sql.NullString
	Value         bool
}

type repository struct {
	db database.Database // central DB
}

func newRepository(db database.Database) *repository {
	return &repository{db: db}
}

func (r *repository) UpsertFlags(ctx context.Context, flags map[string]Flag) error {
	// Insert if not exists; update description/default_value if present
	for _, f := range flags {
		_, err := r.db.ExecContext(ctx, `INSERT INTO feature_flags(key, description, default_value) VALUES(?,?,?)
            ON CONFLICT(key) DO UPDATE SET description=excluded.description, default_value=excluded.default_value, updated_at=CURRENT_TIMESTAMP`,
			f.Key, f.Description, f.Default,
		)
		if err != nil {
			return fmt.Errorf("upsert flag %s: %w", f.Key, err)
		}
	}
	return nil
}

// ListOverrides returns all overrides for the given keys and principals.
func (r *repository) ListOverrides(ctx context.Context, keys []string, p Principals) ([]Override, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	// Build query with IN clauses. For sqlite, prepare placeholders.
	q := `SELECT flag_key, principal_type, principal_id, value FROM feature_flag_overrides WHERE flag_key IN (`
	args := make([]any, 0, len(keys)+4)
	for i, k := range keys {
		if i > 0 {
			q += ","
		}
		q += "?"
		args = append(args, k)
	}
	q += ") AND (principal_type='global'"

	if p.TenantID != nil {
		q += " OR (principal_type='tenant' AND principal_id=?)"
		args = append(args, *p.TenantID)
	}
	if len(p.RoleIDs) > 0 {
		q += " OR (principal_type='role' AND principal_id IN ("
		for i, rid := range p.RoleIDs {
			if i > 0 {
				q += ","
			}
			q += "?"
			args = append(args, rid)
		}
		q += "))"
	}
	if p.UserID != nil {
		q += " OR (principal_type='user' AND principal_id=?)"
		args = append(args, *p.UserID)
	}
	q += ")"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := []Override{}
	for rows.Next() {
		var o Override
		if err := rows.Scan(&o.FlagKey, &o.PrincipalType, &o.PrincipalID, &o.Value); err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

func (r *repository) SetOverride(ctx context.Context, flagKey string, principal Principal, value bool) error {
	// First try update existing (handles NULL principal_id via IFNULL match)
	res, err := r.db.ExecContext(ctx, `UPDATE feature_flag_overrides
        SET value=?, updated_at=CURRENT_TIMESTAMP
        WHERE flag_key=? AND principal_type=? AND IFNULL(principal_id,'')=IFNULL(?, '')`,
		value, flagKey, string(principal.Type), nullIfEmpty(principal.ID),
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	// Insert new row
	_, err = r.db.ExecContext(ctx, `INSERT INTO feature_flag_overrides(flag_key, principal_type, principal_id, value) VALUES(?,?,?,?)`,
		flagKey, string(principal.Type), nullIfEmpty(principal.ID), value,
	)
	return err
}

func (r *repository) DeleteOverride(ctx context.Context, flagKey string, principal Principal) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM feature_flag_overrides WHERE flag_key=? AND principal_type=? AND IFNULL(principal_id,'')=IFNULL(?, '')`,
		flagKey, string(principal.Type), nullIfEmpty(principal.ID),
	)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
