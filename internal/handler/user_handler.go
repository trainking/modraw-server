package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/trainking/modraw-server/internal/middleware"
	"github.com/trainking/modraw-server/internal/repository"
	"github.com/trainking/modraw-server/pkg/httputil"
)

type UserHandler struct {
	userRepo *repository.UserRepository
}

func NewUserHandler(userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

func (h *UserHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	httputil.Success(c, user)
}

type updateMeInput struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input updateMeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	if input.Nickname != "" {
		user.Nickname = input.Nickname
	}
	if input.AvatarURL != "" {
		user.AvatarURL = input.AvatarURL
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.Success(c, user)
}

type changePasswordInput struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input changePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	if len(input.NewPassword) < 8 || len(input.NewPassword) > 72 {
		httputil.Error(c, http.StatusBadRequest, "WEAK_PASSWORD", "password must be 8-72 characters")
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.OldPassword)); err != nil {
		httputil.Error(c, http.StatusBadRequest, "WRONG_PASSWORD", "old password is incorrect")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		return
	}

	if err := h.userRepo.UpdatePassword(c.Request.Context(), userID, string(newHash)); err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.Success(c, gin.H{"message": "password changed"})
}
