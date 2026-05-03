package database

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// pgErrorCodeUniqueViolation is the SQLSTATE value PostgreSQL returns for unique_violation.
const pgErrorCodeUniqueViolation = "23505"

// isUniqueViolation reports whether err is a PostgreSQL unique_violation error.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgErrorCodeUniqueViolation
}

// isUniqueViolationOnConstraint reports whether err is a unique_violation on a specific constraint name.
func isUniqueViolationOnConstraint(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgErrorCodeUniqueViolation && pgErr.ConstraintName == constraint
}

// isNoRows reports whether err signals an empty result set.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// derefTime returns *time.Time from a time.Time value treating zero as nil.
func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// optionalString returns *string treating empty string as nil.
func optionalString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
