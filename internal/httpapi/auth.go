package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/madoxSio/go-cli-market/internal/auth"
)

type authHandler struct {
	svc *auth.Service
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type codeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type resetRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

type codeResponse struct {
	Message          string `json:"message"`
	ConfirmationCode string `json:"confirmation_code,omitempty"`
	ResetCode        string `json:"reset_code,omitempty"`
}

func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if !readJSON(w, r, &req) {
		return
	}
	code, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	// Pas de serveur mail dans le sujet : le code est journalisé et renvoyé au client.
	slog.Info("code de confirmation", "email", req.Email, "code", code)
	writeJson(w, http.StatusCreated, codeResponse{
		Message:          "Compte créé. Confirmez-le avec le code.",
		ConfirmationCode: code,
	})
}

func (h *authHandler) confirm(w http.ResponseWriter, r *http.Request) {
	var req codeRequest
	if !readJSON(w, r, &req) {
		return
	}
	if err := h.svc.Confirm(r.Context(), req.Email, req.Code); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJson(w, http.StatusOK, codeResponse{Message: "Compte confirmé"})
}

func (h *authHandler) renewConfirmation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	code, err := h.svc.RenewConfirmation(r.Context(), req.Email)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if code != "" {
		slog.Info("code de confirmation renouvelé", "email", req.Email, "code", code)
	}
	writeJson(w, http.StatusOK, codeResponse{
		Message:          "Si un compte est en attente, un nouveau code a été généré.",
		ConfirmationCode: code,
	})
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if !readJSON(w, r, &req) {
		return
	}
	token, user, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

func (h *authHandler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	code, err := h.svc.RequestPasswordReset(r.Context(), req.Email)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if code != "" {
		slog.Info("code de réinitialisation", "email", req.Email, "code", code)
	}
	writeJson(w, http.StatusOK, codeResponse{
		Message:   "Si un compte existe, un code de réinitialisation a été généré.",
		ResetCode: code,
	})
}

func (h *authHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetRequest
	if !readJSON(w, r, &req) {
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJson(w, http.StatusOK, codeResponse{Message: "Mot de passe mis à jour"})
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Logout(r.Context(), bearerToken(r)); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJson(w, http.StatusOK, codeResponse{Message: "Session fermée"})
}

func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.CurrentUser(r.Context(), bearerToken(r))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJson(w, http.StatusOK, user)
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func writeAuthError(w http.ResponseWriter, err error) {
	var validation auth.ValidationError
	switch {
	case errors.As(err, &validation):
		writeJson(w, http.StatusBadRequest, map[string]string{"error": validation.Error()})
	case errors.Is(err, auth.ErrEmailTaken), errors.Is(err, auth.ErrAlreadyConfirmed), errors.Is(err, auth.ErrLastAdmin), errors.Is(err, auth.ErrSelfDelete):
		writeJson(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrUnauthenticated):
		writeJson(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
	case errors.Is(err, auth.ErrNotConfirmed), errors.Is(err, auth.ErrForbidden):
		writeJson(w, http.StatusForbidden, map[string]string{"error": err.Error()})
	case errors.Is(err, auth.ErrInvalidCode), errors.Is(err, auth.ErrExpiredCode):
		writeJson(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, auth.ErrNotFound):
		writeJson(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	default:
		slog.Error("authentification", "err", err)
		writeJson(w, http.StatusInternalServerError, map[string]string{"error": "erreur interne"})
	}
}
