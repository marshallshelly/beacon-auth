package session

import (
	"context"
	"time"

	"github.com/marshallshelly/beacon-auth/adapter"
	"github.com/marshallshelly/beacon-auth/core"
)

// DBStore implements Store using the database adapter
type DBStore struct {
	internal *adapter.InternalAdapter
}

// NewDBStore creates a new database session store
func NewDBStore(coreAdapter core.Adapter) *DBStore {
	return &DBStore{
		internal: adapter.NewInternalAdapter(coreAdapter, nil),
	}
}

// Get retrieves a session by token from the database
func (d *DBStore) Get(ctx context.Context, token string) (*core.Session, *core.User, error) {
	return d.internal.FindSessionWithUser(ctx, token)
}

// Set stores a session in the database
func (d *DBStore) Set(ctx context.Context, session *core.Session) error {
	// Check if session exists first
	existing, _, err := d.internal.FindSessionWithUser(ctx, session.Token)
	if err != nil && err != core.ErrSessionNotFound {
		return err
	}

	if existing != nil {
		// Update existing session
		query := &core.Query{
			Model: "sessions",
			Where: []core.WhereClause{
				{Field: "token", Operator: core.OpEqual, Value: session.Token},
			},
		}

		_, err := d.internal.Adapter().Update(ctx, query, map[string]interface{}{
			"expires_at": session.ExpiresAt,
			"updated_at": time.Now(),
		})
		return err
	}

	// Create new session
	_, err = d.internal.Adapter().Create(ctx, "sessions", map[string]interface{}{
		"id":         session.ID,
		"user_id":    session.UserID,
		"token":      session.Token,
		"expires_at": session.ExpiresAt,
		"ip_address": session.IPAddress,
		"user_agent": session.UserAgent,
		"created_at": session.CreatedAt,
		"updated_at": session.UpdatedAt,
	})

	return err
}

// Delete removes a session from the database
func (d *DBStore) Delete(ctx context.Context, token string) error {
	return d.internal.RevokeSession(ctx, token)
}

// DeleteByUserID removes all sessions for a user from the database
func (d *DBStore) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := d.internal.RevokeAllUserSessions(ctx, userID)
	return err
}

// Cleanup removes expired sessions from the database
func (d *DBStore) Cleanup(ctx context.Context) error {
	query := &core.Query{
		Model: "sessions",
		Where: []core.WhereClause{
			{Field: "expires_at", Operator: core.OpLessThan, Value: time.Now()},
		},
	}

	_, err := d.internal.Adapter().DeleteMany(ctx, query)
	return err
}

// Close closes the database connection
func (d *DBStore) Close() error {
	return d.internal.Adapter().Close()
}
