package auth

import (
	"context"
	"time"
)

// Repository persiste les comptes et les sessions.
// L'implémentation PostgreSQL est dans internal/store, une implémentation
// mémoire sert aux tests : le service ne connaît pas SQL.
type Repository interface {
	CreateAccount(ctx context.Context, account Account) (Account, error)
	FindByEmail(ctx context.Context, email string) (Account, error)
	UpdateAccount(ctx context.Context, account Account) error
	CreateSession(ctx context.Context, userID int64, token string, expires time.Time) error
	FindBySession(ctx context.Context, token string, now time.Time) (Account, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteUserSessions(ctx context.Context, userID int64) error
}
