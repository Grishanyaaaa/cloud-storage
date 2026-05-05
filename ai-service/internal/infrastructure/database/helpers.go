package database

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// isNoRows reports whether err signals an empty result set.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
