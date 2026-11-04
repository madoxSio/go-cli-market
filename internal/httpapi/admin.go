package httpapi

import (
	"net/http"
	"strconv"

	"github.com/madoxSio/go-cli-market/internal/auth"
)

func (h *authHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	users, err := h.svc.ListUsers(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJson(w, http.StatusOK, users)
}

func (h *authHandler) createUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	var req struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		Role      string `json:"role"`
		Confirmed bool   `json:"confirmed"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	user, err := h.svc.CreateUser(r.Context(), req.Email, req.Password, req.Role, req.Confirmed)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJson(w, http.StatusCreated, user)
}

func (h *authHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	user, err := h.svc.UpdateUser(r.Context(), id, req.Email, req.Password, req.Role)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJson(w, http.StatusOK, user)
}

func (h *authHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteUser(r.Context(), actor.ID, id); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJson(w, http.StatusOK, codeResponse{Message: "Compte supprimé"})
}

func (h *authHandler) confirmUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	user, err := h.svc.ConfirmUser(r.Context(), id)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJson(w, http.StatusOK, user)
}

func (h *authHandler) requireAdmin(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, err := h.svc.CurrentUser(r.Context(), bearerToken(r))
	if err != nil {
		writeAuthError(w, err)
		return auth.User{}, false
	}
	if user.Role != "admin" {
		writeAuthError(w, auth.ErrForbidden)
		return auth.User{}, false
	}
	return user, true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "identifiant invalide"})
		return 0, false
	}
	return id, true
}
