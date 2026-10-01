package store

import (
	"database/sql"
	"errors"
)

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

var ErrImmutableConflict = errors.New("immutable version conflict")
var ErrInvalidPackage = errors.New("invalid sealed package")
var ErrNotFound = errors.New("not found")
var ErrUnavailable = errors.New("service unavailable")
