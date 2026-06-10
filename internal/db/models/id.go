package models

import "github.com/oklog/ulid/v2"

// NewID returns a fresh ULID. Used by BeforeCreate hooks on Base.
func NewID() string {
	return ulid.Make().String()
}
