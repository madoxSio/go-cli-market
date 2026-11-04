package auth

import (
	"context"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// EnsureAdmin crée un administrateur confirmé s'il n'en existe pas encore.
// Les appels suivants ne changent pas le mot de passe déjà en base.
func (s *Service) EnsureAdmin(ctx context.Context, email, password string) error {
	email, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	accounts, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if account.Role == "admin" {
			return nil
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.repo.CreateAccount(ctx, Account{
		Email:        email,
		PasswordHash: string(hash),
		Role:         "admin",
		Confirmed:    true,
	})
	return err
}

// ListUsers retourne les comptes sans leur empreinte, triés par email.
func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	accounts, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]User, 0, len(accounts))
	for _, account := range accounts {
		users = append(users, account.Public())
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Email < users[j].Email })
	return users, nil
}

// CreateUser crée un compte depuis l'administration. confirmed évite le code d'inscription.
func (s *Service) CreateUser(ctx context.Context, email, password, role string, confirmed bool) (User, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return User{}, err
	}
	if err := validatePassword(password); err != nil {
		return User{}, err
	}
	role, err = normalizeRole(role)
	if err != nil {
		return User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	account, err := s.repo.CreateAccount(ctx, Account{
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		Confirmed:    confirmed,
	})
	if err != nil {
		return User{}, err
	}
	return account.Public(), nil
}

// UpdateUser modifie l'email, le rôle et, si password n'est pas vide, le mot de passe.
func (s *Service) UpdateUser(ctx context.Context, id int64, email, password, role string) (User, error) {
	account, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	email, err = normalizeEmail(email)
	if err != nil {
		return User{}, err
	}
	role, err = normalizeRole(role)
	if err != nil {
		return User{}, err
	}
	if account.Role == "admin" && role != "admin" {
		if err := s.keepAnAdmin(ctx, account.ID); err != nil {
			return User{}, err
		}
	}
	if password != "" {
		if err := validatePassword(password); err != nil {
			return User{}, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return User{}, err
		}
		account.PasswordHash = string(hash)
		if err := s.repo.DeleteUserSessions(ctx, account.ID); err != nil {
			return User{}, err
		}
	}
	account.Email = email
	account.Role = role
	if err := s.repo.UpdateAccount(ctx, account); err != nil {
		return User{}, err
	}
	return account.Public(), nil
}

// ConfirmUser marque un compte comme confirmé, sans code.
func (s *Service) ConfirmUser(ctx context.Context, id int64) (User, error) {
	account, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	if account.Confirmed {
		return User{}, ErrAlreadyConfirmed
	}
	account.Confirmed = true
	account.ConfirmationCode = ""
	account.ConfirmationExpires = time.Time{}
	if err := s.repo.UpdateAccount(ctx, account); err != nil {
		return User{}, err
	}
	return account.Public(), nil
}

// DeleteUser supprime un compte et ses sessions.
// Un administrateur ne peut pas se supprimer, ni retirer le dernier admin.
func (s *Service) DeleteUser(ctx context.Context, actorID, targetID int64) error {
	if actorID == targetID {
		return ErrSelfDelete
	}
	account, err := s.repo.FindByID(ctx, targetID)
	if err != nil {
		return err
	}
	if account.Role == "admin" {
		if err := s.keepAnAdmin(ctx, account.ID); err != nil {
			return err
		}
	}
	if err := s.repo.DeleteUserSessions(ctx, targetID); err != nil {
		return err
	}
	return s.repo.DeleteAccount(ctx, targetID)
}

func (s *Service) keepAnAdmin(ctx context.Context, exceptID int64) error {
	accounts, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if account.Role == "admin" && account.ID != exceptID {
			return nil
		}
	}
	return ErrLastAdmin
}

func normalizeRole(role string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "", "client":
		return "client", nil
	case "admin":
		return "admin", nil
	default:
		return "", ValidationError{Message: "rôle invalide"}
	}
}
