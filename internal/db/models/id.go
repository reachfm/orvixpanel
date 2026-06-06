package models

import "github.com/oklog/ulid/v2"

// newID returns a fresh ULID. Used by BeforeCreate hooks on Base.
func newID() string {
	return ulid.Make().String()
}
