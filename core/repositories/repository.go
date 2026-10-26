// core/repositories/repository.go

package repositories

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound is returned when a row is not present.
var ErrNotFound = errors.New("repositories: not found")

// row is the minimal interface satisfied by sql.Row and sql.Rows.
type row interface {
	Scan(dest ...any) error
}

// mapErr translates database/sql's sentinel into the repository-level not-found.
func mapErr(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// wrapErr adds an operation prefix unless the error is the not-found sentinel.
func wrapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}
