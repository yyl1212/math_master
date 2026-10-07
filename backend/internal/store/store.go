package store

import (
	"database/sql"
	"errors"
)

type Store struct {
	db      *sql.DB
	codeSHA string
}

func New(db *sql.DB) *Store { return &Store{db: db} }

var ErrImmutableConflict = errors.New("immutable version conflict")
var ErrInvalidPackage = errors.New("invalid sealed package")
var ErrNotFound = errors.New("not found")
var ErrUnavailable = errors.New("service unavailable")

// Trusted version is supplied by immutable build metadata, never by an HTTP request.
func NewWithTrustedCodeSHA(db *sql.DB, codeSHA string) *Store {
	return &Store{db: db, codeSHA: codeSHA}
}
