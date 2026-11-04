package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/madoxSio/go-cli-market/internal/auth"
)

// Repository est le dépôt PostgreSQL du service d'authentification.
type Repository struct {
	db *sql.DB
}

// NewRepository branche le service sur la base ouverte par Open.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

var _ auth.Repository = (*Repository)(nil)

func (r *Repository) CreateAccount(ctx context.Context, account auth.Account) (auth.Account, error) {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO users (
			email, password_hash, role, confirmed,
			confirmation_code, confirmation_expires_at,
			reset_code, reset_expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		account.Email,
		account.PasswordHash,
		account.Role,
		account.Confirmed,
		nullString(account.ConfirmationCode),
		nullTime(account.ConfirmationExpires),
		nullString(account.ResetCode),
		nullTime(account.ResetExpires),
	).Scan(&account.ID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return auth.Account{}, auth.ErrEmailTaken
		}
		return auth.Account{}, err
	}
	return account, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (auth.Account, error) {
	row := r.db.QueryRowContext(ctx, userSelect+` WHERE email = $1`, email)
	account, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Account{}, auth.ErrNotFound
	}
	return account, err
}

func (r *Repository) UpdateAccount(ctx context.Context, account auth.Account) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE users SET
			email = $1,
			password_hash = $2,
			role = $3,
			confirmed = $4,
			confirmation_code = $5,
			confirmation_expires_at = $6,
			reset_code = $7,
			reset_expires_at = $8
		WHERE id = $9`,
		account.Email,
		account.PasswordHash,
		account.Role,
		account.Confirmed,
		nullString(account.ConfirmationCode),
		nullTime(account.ConfirmationExpires),
		nullString(account.ResetCode),
		nullTime(account.ResetExpires),
		account.ID,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return auth.ErrEmailTaken
		}
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return auth.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateSession(ctx context.Context, userID int64, token string, expires time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		token, userID, expires,
	)
	return err
}

func (r *Repository) FindBySession(ctx context.Context, token string, now time.Time) (auth.Account, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.password_hash, u.role, u.confirmed,
		       u.confirmation_code, u.confirmation_expires_at,
		       u.reset_code, u.reset_expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = $1 AND s.expires_at > $2`, token, now)
	account, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Account{}, auth.ErrNotFound
	}
	return account, err
}

func (r *Repository) DeleteSession(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (r *Repository) FindByID(ctx context.Context, id int64) (auth.Account, error) {
	row := r.db.QueryRowContext(ctx, userSelect+` WHERE id = $1`, id)
	account, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Account{}, auth.ErrNotFound
	}
	return account, err
}

func (r *Repository) ListAccounts(ctx context.Context) ([]auth.Account, error) {
	rows, err := r.db.QueryContext(ctx, userSelect+` ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var accounts []auth.Account
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (r *Repository) DeleteAccount(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return auth.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteUserSessions(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

const userSelect = `
	SELECT id, email, password_hash, role, confirmed,
	       confirmation_code, confirmation_expires_at,
	       reset_code, reset_expires_at
	FROM users`

type scanner interface {
	Scan(dest ...any) error
}

func scanAccount(row scanner) (auth.Account, error) {
	var account auth.Account
	var confirmationCode, resetCode sql.NullString
	var confirmationExpires, resetExpires sql.NullTime
	err := row.Scan(
		&account.ID,
		&account.Email,
		&account.PasswordHash,
		&account.Role,
		&account.Confirmed,
		&confirmationCode,
		&confirmationExpires,
		&resetCode,
		&resetExpires,
	)
	if err != nil {
		return auth.Account{}, err
	}
	account.ConfirmationCode = confirmationCode.String
	account.ResetCode = resetCode.String
	if confirmationExpires.Valid {
		account.ConfirmationExpires = confirmationExpires.Time
	}
	if resetExpires.Valid {
		account.ResetExpires = resetExpires.Time
	}
	return account, nil
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullTime(value time.Time) sql.NullTime {
	if value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value, Valid: true}
}
