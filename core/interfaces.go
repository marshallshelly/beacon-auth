package core

import (
	"context"
	"net/http"
	"time"
)

// Endpoint defines an API endpoint
type Endpoint struct {
	Method  string
	Handler http.HandlerFunc
}

// Adapter defines the interface for database adapters
type Adapter interface {
	// CRUD operations
	Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error)
	FindOne(ctx context.Context, query *Query) (map[string]interface{}, error)
	FindMany(ctx context.Context, query *Query) ([]map[string]interface{}, error)
	Update(ctx context.Context, query *Query, data map[string]interface{}) (map[string]interface{}, error)
	UpdateMany(ctx context.Context, query *Query, data map[string]interface{}) (int64, error)
	Delete(ctx context.Context, query *Query) error
	DeleteMany(ctx context.Context, query *Query) (int64, error)
	Count(ctx context.Context, query *Query) (int64, error)

	// Transaction support
	Transaction(ctx context.Context, fn func(Adapter) error) error

	// Connection management
	Ping(ctx context.Context) error
	Close() error

	// Metadata
	ID() string
}

// Operator type
type Operator string

const (
	OpEqual          Operator = "="
	OpNotEqual       Operator = "!="
	OpGreaterThan    Operator = ">"
	OpGreaterOrEqual Operator = ">="
	OpLessThan       Operator = "<"
	OpLessOrEqual    Operator = "<="
	OpLike           Operator = "LIKE"
	OpIn             Operator = "IN"
	OpNotIn          Operator = "NOT IN"
	OpIsNull         Operator = "IS NULL"
	OpIsNotNull      Operator = "IS NOT NULL"
)

// JoinType represents join type
type JoinType string

const (
	InnerJoin JoinType = "INNER"
	LeftJoin  JoinType = "LEFT"
	RightJoin JoinType = "RIGHT"
)

// Query represents a database query
type Query struct {
	Model   string
	Where   []WhereClause
	Joins   []Join
	Limit   int
	Offset  int
	OrderBy []OrderBy
}

// WhereClause represents a where condition
type WhereClause struct {
	Field    string
	Operator Operator
	Value    interface{}
	Or       bool
}

// Join represents a table join
type Join struct {
	Model string
	Type  JoinType
	On    JoinCondition
}

// JoinCondition represents join condition
type JoinCondition struct {
	Left  string
	Right string
}

// OrderBy represents order by clause
type OrderBy struct {
	Field string
	Desc  bool
}

// Plugin defines the interface for BeaconAuth plugins
type Plugin interface {
	// ID returns the unique plugin identifier
	ID() string

	// Init initializes the plugin with the auth context
	Init(ctx *AuthContext) error

	// Endpoints returns the plugin endpoints
	Endpoints() map[string]Endpoint
}

// OAuthProvider defines the interface for OAuth providers
type OAuthProvider interface {
	ID() string
	Name() string
}

// Mailer defines the interface for sending emails
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SecondaryStorage defines the interface for session secondary storage
type SecondaryStorage interface {
	Get(ctx context.Context, key string) (*Session, error)
	Set(ctx context.Context, key string, session *Session, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Close() error
}

// RateLimitStorage defines the interface for rate limit storage
type RateLimitStorage interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
	Reset(ctx context.Context, key string) error
}

// Hook is a function that runs before or after a request
type Hook interface {
	Execute(ctx context.Context, data interface{}) error
}

// Logger defines the logging interface
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// SessionManager defines the interface for session management
type SessionManager interface {
	Create(ctx context.Context, userID string, opts *SessionOptions) (*Session, *User, string, error)
	Get(ctx context.Context, token string) (*Session, *User, error)
	Update(ctx context.Context, session *Session) error
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID string) error
}

// PasswordHasher defines the interface for password hashing
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) (bool, error)
}

// DataManager defines high-level database operations
type DataManager interface {
	FindAccountByProvider(ctx context.Context, provider, accountID string) (*Account, error)
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, email, name string) (*User, error)
	CreateOAuthAccount(ctx context.Context, userID, provider, accountID, accessToken, refreshToken string, expiresAt *time.Time) (*Account, error)
	CreateCredentialAccount(ctx context.Context, userID, identifier, passwordHash string) (*Account, error)
}
