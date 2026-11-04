package auth

import (
	"context"
	"sync"
	"time"
)

type memSession struct {
	userID  int64
	expires time.Time
}

// MemoryRepository garde les comptes en mémoire. Les tests l'utilisent à la place de PostgreSQL.
type MemoryRepository struct {
	mu       sync.Mutex
	nextID   int64
	byEmail  map[string]Account
	byID     map[int64]string
	sessions map[string]memSession
}

// NewMemoryRepository retourne un dépôt vide.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byEmail:  map[string]Account{},
		byID:     map[int64]string{},
		sessions: map[string]memSession{},
	}
}

func (m *MemoryRepository) CreateAccount(_ context.Context, account Account) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byEmail[account.Email]; ok {
		return Account{}, ErrEmailTaken
	}
	m.nextID++
	account.ID = m.nextID
	m.byEmail[account.Email] = account
	m.byID[account.ID] = account.Email
	return account, nil
}

func (m *MemoryRepository) FindByEmail(_ context.Context, email string) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	account, ok := m.byEmail[email]
	if !ok {
		return Account{}, ErrNotFound
	}
	return account, nil
}

func (m *MemoryRepository) UpdateAccount(_ context.Context, account Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	previous, ok := m.byID[account.ID]
	if !ok {
		return ErrNotFound
	}
	if previous != account.Email {
		if _, taken := m.byEmail[account.Email]; taken {
			return ErrEmailTaken
		}
		delete(m.byEmail, previous)
	}
	m.byEmail[account.Email] = account
	m.byID[account.ID] = account.Email
	return nil
}

func (m *MemoryRepository) CreateSession(_ context.Context, userID int64, token string, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[userID]; !ok {
		return ErrNotFound
	}
	m.sessions[token] = memSession{userID: userID, expires: expires}
	return nil
}

func (m *MemoryRepository) FindBySession(_ context.Context, token string, now time.Time) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[token]
	if !ok || !session.expires.After(now) {
		return Account{}, ErrNotFound
	}
	email, ok := m.byID[session.userID]
	if !ok {
		return Account{}, ErrNotFound
	}
	return m.byEmail[email], nil
}

func (m *MemoryRepository) DeleteSession(_ context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
	return nil
}

func (m *MemoryRepository) FindByID(_ context.Context, id int64) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	email, ok := m.byID[id]
	if !ok {
		return Account{}, ErrNotFound
	}
	return m.byEmail[email], nil
}

func (m *MemoryRepository) ListAccounts(_ context.Context) ([]Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	accounts := make([]Account, 0, len(m.byEmail))
	for _, account := range m.byEmail {
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (m *MemoryRepository) DeleteAccount(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	email, ok := m.byID[id]
	if !ok {
		return ErrNotFound
	}
	delete(m.byID, id)
	delete(m.byEmail, email)
	return nil
}

func (m *MemoryRepository) DeleteUserSessions(_ context.Context, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for token, session := range m.sessions {
		if session.userID == userID {
			delete(m.sessions, token)
		}
	}
	return nil
}
