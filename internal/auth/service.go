package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service applique les règles d'authentification.
// Le dépôt ne décide rien : il lit et écrit ce que le service lui donne.
type Service struct {
	repo      Repository
	now       func() time.Time
	dummyHash []byte
}

// NewService prépare une empreinte factice. Elle sert quand l'email est inconnu,
// pour que le temps de réponse reste proche de celui d'un vrai bcrypt.
func NewService(repo Repository) (*Service, error) {
	dummy, err := dummyPasswordHash()
	if err != nil {
		return nil, err
	}
	return &Service{repo: repo, now: time.Now, dummyHash: dummy}, nil
}

var (
	dummyOnce sync.Once
	dummyHash []byte
	dummyErr  error
)

func dummyPasswordHash() ([]byte, error) {
	dummyOnce.Do(func() {
		dummyHash, dummyErr = bcrypt.GenerateFromPassword([]byte("placeholder-not-a-user"), bcrypt.DefaultCost)
	})
	if dummyErr != nil {
		return nil, dummyErr
	}
	return dummyHash, nil
}

// Register crée un compte non confirmé et retourne le code à transmettre à l'utilisateur.
// Le sujet ne prévoit pas d'envoi d'email : l'API rend le code pour que le client l'affiche.
func (s *Service) Register(ctx context.Context, email, password string) (string, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return "", err
	}
	if err := validatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	code, err := newCode()
	if err != nil {
		return "", err
	}
	_, err = s.repo.CreateAccount(ctx, Account{
		Email:               email,
		PasswordHash:        string(hash),
		Role:                "client",
		ConfirmationCode:    code,
		ConfirmationExpires: s.now().Add(codeTTL),
	})
	if err != nil {
		return "", err
	}
	return code, nil
}

// Confirm valide le code d'inscription et ouvre le droit de connexion.
func (s *Service) Confirm(ctx context.Context, email, code string) error {
	acc, err := s.accountByEmail(ctx, email)
	if err != nil {
		return err
	}
	if acc.Confirmed {
		return ErrAlreadyConfirmed
	}
	if !codesMatch(normalizeCode(code), acc.ConfirmationCode) {
		return ErrInvalidCode
	}
	if !acc.ConfirmationExpires.After(s.now()) {
		return ErrExpiredCode
	}
	acc.Confirmed = true
	acc.ConfirmationCode = ""
	acc.ConfirmationExpires = time.Time{}
	return s.repo.UpdateAccount(ctx, acc)
}

// RenewConfirmation remplace le code d'un compte encore en attente.
// Un email inconnu ne produit pas d'erreur : le code renvoyé est vide.
func (s *Service) RenewConfirmation(ctx context.Context, email string) (string, error) {
	acc, err := s.lookupOptional(ctx, email)
	if err != nil || acc.ID == 0 {
		return "", err
	}
	if acc.Confirmed {
		return "", ErrAlreadyConfirmed
	}
	return s.saveCode(ctx, acc, true)
}

// Login vérifie le mot de passe et ouvre une session.
func (s *Service) Login(ctx context.Context, email, password string) (string, User, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return "", User{}, err
	}
	acc, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return "", User{}, ErrInvalidCredentials
		}
		return "", User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password)) != nil {
		return "", User{}, ErrInvalidCredentials
	}
	if !acc.Confirmed {
		return "", User{}, ErrNotConfirmed
	}
	token, err := newToken()
	if err != nil {
		return "", User{}, err
	}
	if err := s.repo.CreateSession(ctx, acc.ID, token, s.now().Add(sessionTTL)); err != nil {
		return "", User{}, err
	}
	return token, acc.Public(), nil
}

// RequestPasswordReset prépare un code de réinitialisation sans exiger de session.
// L'absence de compte renvoie un code vide et aucune erreur.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	acc, err := s.lookupOptional(ctx, email)
	if err != nil || acc.ID == 0 {
		return "", err
	}
	return s.saveCode(ctx, acc, false)
}

// ResetPassword remplace l'empreinte si le code est encore valable, puis coupe les sessions.
func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	acc, err := s.accountByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return ErrInvalidCode
	}
	if err != nil {
		return err
	}
	if !codesMatch(normalizeCode(code), acc.ResetCode) {
		return ErrInvalidCode
	}
	if !acc.ResetExpires.After(s.now()) {
		return ErrExpiredCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	acc.PasswordHash = string(hash)
	acc.ResetCode = ""
	acc.ResetExpires = time.Time{}
	if err := s.repo.UpdateAccount(ctx, acc); err != nil {
		return err
	}
	return s.repo.DeleteUserSessions(ctx, acc.ID)
}

// CurrentUser retrouve le compte lié à un jeton de session encore valide.
func (s *Service) CurrentUser(ctx context.Context, token string) (User, error) {
	if strings.TrimSpace(token) == "" {
		return User{}, ErrUnauthenticated
	}
	acc, err := s.repo.FindBySession(ctx, token, s.now())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, ErrUnauthenticated
		}
		return User{}, err
	}
	return acc.Public(), nil
}

// Logout supprime la session. Un jeton absent est une erreur, un jeton déjà
// supprimé est accepté par le dépôt.
func (s *Service) Logout(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrUnauthenticated
	}
	return s.repo.DeleteSession(ctx, token)
}

func (s *Service) accountByEmail(ctx context.Context, email string) (Account, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return Account{}, err
	}
	return s.repo.FindByEmail(ctx, email)
}

// lookupOptional normalise l'email. Un compte absent donne un Account vide et aucune erreur.
func (s *Service) lookupOptional(ctx context.Context, email string) (Account, error) {
	acc, err := s.accountByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return Account{}, nil
	}
	return acc, err
}

func (s *Service) saveCode(ctx context.Context, acc Account, confirmation bool) (string, error) {
	code, err := newCode()
	if err != nil {
		return "", err
	}
	expires := s.now().Add(codeTTL)
	if confirmation {
		acc.ConfirmationCode = code
		acc.ConfirmationExpires = expires
	} else {
		acc.ResetCode = code
		acc.ResetExpires = expires
	}
	if err := s.repo.UpdateAccount(ctx, acc); err != nil {
		return "", err
	}
	return code, nil
}
