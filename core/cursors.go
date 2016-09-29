package core

import (
	"context"
	"fmt"
	"math"

	"chain/core/cursor"
)

// POST /create-cursor
func (a *api) createCursor(ctx context.Context, in struct {
	Alias  string
	Filter string

	// ClientToken is the application's unique token for the cursor. Every cursor
	// should have a unique client token. The client token is used to ensure
	// idempotency of create cursor requests. Duplicate create cursor requests
	// with the same client_token will only create one cursor.
	ClientToken *string `json:"client_token"`
}) (*cursor.Cursor, error) {
	after := fmt.Sprintf("%x:%x-%x", a.c.Height(), math.MaxInt32, uint64(math.MaxInt64))
	return cursor.Create(ctx, in.Alias, in.Filter, after, in.ClientToken)
}

// POST /get-cursor
func getCursor(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}) (*cursor.Cursor, error) {
	return cursor.Find(ctx, in.ID, in.Alias)
}

// POST /delete-cursor
func deleteCursor(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}) error {
	return cursor.Delete(ctx, in.ID, in.Alias)
}

// POST /update-cursor
func updateCursor(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
	Prev  string `json:"prev"`
	After string `json:"after"`
}) (*cursor.Cursor, error) {
	return cursor.Update(ctx, in.ID, in.Alias, in.Prev, in.After)
}
