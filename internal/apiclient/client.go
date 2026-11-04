// Package apiclient appelle l'API HTTP depuis les interfaces terminal et SSH.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// User est le compte renvoyé par l'API, sans secret.
type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Error est une erreur métier renvoyée par l'API.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("erreur HTTP %d", e.Status)
}

// Client parle au serveur net/http de la plateforme.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New prépare un client. baseURL est l'adresse de l'API, par exemple http://localhost:8080.
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// SetToken remplace le jeton de session envoyé dans Authorization.
func (c *Client) SetToken(token string) {
	c.token = token
}

// Register crée un compte et retourne le code de confirmation.
func (c *Client) Register(ctx context.Context, email, password string) (string, error) {
	var out struct {
		ConfirmationCode string `json:"confirmation_code"`
	}
	err := c.do(ctx, http.MethodPost, "/auth/register", map[string]string{
		"email": email, "password": password,
	}, &out, false)
	return out.ConfirmationCode, err
}

// Confirm valide le code d'inscription.
func (c *Client) Confirm(ctx context.Context, email, code string) error {
	return c.do(ctx, http.MethodPost, "/auth/confirm", map[string]string{
		"email": email, "code": code,
	}, nil, false)
}

// RenewConfirmation demande un nouveau code pour un compte non confirmé.
func (c *Client) RenewConfirmation(ctx context.Context, email string) (string, error) {
	var out struct {
		ConfirmationCode string `json:"confirmation_code"`
	}
	err := c.do(ctx, http.MethodPost, "/auth/confirmation-code", map[string]string{
		"email": email,
	}, &out, false)
	return out.ConfirmationCode, err
}

// Login ouvre une session et mémorise le jeton.
func (c *Client) Login(ctx context.Context, email, password string) (User, error) {
	var out struct {
		Token string `json:"token"`
		User  User   `json:"user"`
	}
	err := c.do(ctx, http.MethodPost, "/auth/login", map[string]string{
		"email": email, "password": password,
	}, &out, false)
	if err != nil {
		return User{}, err
	}
	c.token = out.Token
	return out.User, nil
}

// ForgotPassword demande un code de réinitialisation. Le code est vide si le compte n'existe pas.
func (c *Client) ForgotPassword(ctx context.Context, email string) (string, error) {
	var out struct {
		ResetCode string `json:"reset_code"`
	}
	err := c.do(ctx, http.MethodPost, "/auth/forgot-password", map[string]string{
		"email": email,
	}, &out, false)
	return out.ResetCode, err
}

// ResetPassword change le mot de passe avec le code reçu.
func (c *Client) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	return c.do(ctx, http.MethodPost, "/auth/reset-password", map[string]string{
		"email": email, "code": code, "new_password": newPassword,
	}, nil, false)
}

// Logout ferme la session côté serveur puis oublie le jeton local.
func (c *Client) Logout(ctx context.Context) error {
	err := c.do(ctx, http.MethodPost, "/auth/logout", nil, nil, true)
	c.token = ""
	return err
}

func (c *Client) do(ctx context.Context, method, path string, in, out any, withAuth bool) error {
	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if withAuth {
		if c.token == "" {
			return &Error{Status: http.StatusUnauthorized, Message: "authentification requise"}
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("connexion au serveur impossible : %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		var apiErr struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(payload, &apiErr)
		return &Error{Status: resp.StatusCode, Message: apiErr.Error}
	}
	if out != nil && len(bytes.TrimSpace(payload)) > 0 {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("réponse JSON illisible : %w", err)
		}
	}
	return nil
}
