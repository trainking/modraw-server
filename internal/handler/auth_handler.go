package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trainking/modraw-server/internal/service"
	"github.com/trainking/modraw-server/pkg/httputil"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input service.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	tokens, err := h.authService.Register(c.Request.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			httputil.Error(c, http.StatusConflict, "EMAIL_TAKEN", "this email is already registered")
		case errors.Is(err, service.ErrWeakPassword):
			httputil.Error(c, http.StatusBadRequest, "WEAK_PASSWORD", "password must be 8-72 characters")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "registration failed")
		}
		return
	}

	httputil.Created(c, tokens)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input service.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	tokens, err := h.authService.Login(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCreds) {
			httputil.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
			return
		}
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "login failed")
		return
	}

	httputil.Success(c, tokens)
}

type refreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var input refreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	tokens, err := h.authService.Refresh(c.Request.Context(), input.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			httputil.Error(c, http.StatusUnauthorized, "INVALID_TOKEN", "invalid refresh token")
		case errors.Is(err, service.ErrTokenRevoked):
			httputil.Error(c, http.StatusUnauthorized, "TOKEN_REVOKED", "refresh token has been revoked")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "token refresh failed")
		}
		return
	}

	httputil.Success(c, tokens)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var input refreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	_ = h.authService.Logout(c.Request.Context(), input.RefreshToken)
	httputil.Success(c, gin.H{"message": "logged out"})
}
