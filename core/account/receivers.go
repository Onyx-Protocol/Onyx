package account

import (
	"context"
	"time"

	chainjson "chain/encoding/json"
	"chain/errors"
)

const defaultReceiverExpiry = 30 * 24 * time.Hour // 30 days

// Reciever encapsulates information about where to send assets to
// be received by an account.
type Receiver struct {
	ControlProgram chainjson.HexBytes `json:"control_program"`
	ExpiresAt      time.Time          `json:"expires_at"`
}

// CreateReceiver creates a new account receiver for an account
// with the provided expiry. If a zero time is provided for the
// expiry, a default expiry of 30 days from the current time is
// used.
func (m *Manager) CreateReceiver(ctx context.Context, accID, accAlias string, expiresAt time.Time) (*Receiver, error) {
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(defaultReceiverExpiry)
	}

	if accAlias != "" {
		s, err := m.FindByAlias(ctx, accAlias)
		if err != nil {
			return nil, err
		}
		accID = s.ID
	}

	cp, err := m.CreateControlProgram(ctx, accID, false, expiresAt)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return &Receiver{
		ControlProgram: cp,
		ExpiresAt:      expiresAt,
	}, nil
}
