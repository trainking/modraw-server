package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/trainking/modraw-server/internal/middleware"
	"github.com/trainking/modraw-server/internal/service"
	"github.com/trainking/modraw-server/pkg/httputil"
)

type ShareLinkHandler struct {
	shareLinkService *service.ShareLinkService
}

func NewShareLinkHandler(shareLinkService *service.ShareLinkService) *ShareLinkHandler {
	return &ShareLinkHandler{shareLinkService: shareLinkService}
}

func (h *ShareLinkHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	canvasID := c.Param("id")

	links, err := h.shareLinkService.List(c.Request.Context(), canvasID, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCanvasNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "canvas not found")
		case errors.Is(err, service.ErrCanvasAccessDenied):
			httputil.Error(c, http.StatusForbidden, "FORBIDDEN", "access denied")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	httputil.Success(c, links)
}

type createShareLinkInput struct {
	Permission string  `json:"permission" binding:"required"`
	Password   string  `json:"password"`
	ExpiresAt  *string `json:"expires_at"`
}

func (h *ShareLinkHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	canvasID := c.Param("id")

	var input createShareLinkInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	if input.Permission != "readonly" && input.Permission != "collaborate" {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", "permission must be 'readonly' or 'collaborate'")
		return
	}

	var expiresAt *time.Time
	if input.ExpiresAt != nil && *input.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *input.ExpiresAt)
		if err != nil {
			httputil.Error(c, http.StatusBadRequest, "VALIDATION", "invalid expires_at format, use RFC3339")
			return
		}
		expiresAt = &t
	}

	svcInput := service.CreateShareLinkInput{
		Permission: input.Permission,
		Password:   input.Password,
		ExpiresAt:  expiresAt,
	}

	result, err := h.shareLinkService.Create(c.Request.Context(), canvasID, userID, svcInput)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCanvasNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "canvas not found")
		case errors.Is(err, service.ErrCanvasAccessDenied):
			httputil.Error(c, http.StatusForbidden, "FORBIDDEN", "access denied")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	httputil.Created(c, result)
}

func (h *ShareLinkHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	canvasID := c.Param("id")
	shareID := c.Param("share_id")

	if err := h.shareLinkService.Delete(c.Request.Context(), shareID, canvasID, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrCanvasNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "canvas not found")
		case errors.Is(err, service.ErrCanvasAccessDenied):
			httputil.Error(c, http.StatusForbidden, "FORBIDDEN", "access denied")
		case errors.Is(err, service.ErrShareLinkNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "share link not found")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	httputil.NoContent(c)
}

func (h *ShareLinkHandler) GetByCode(c *gin.Context) {
	code := c.Param("code")

	info, canvasName, thumbnail, ownerName, err := h.shareLinkService.GetByCode(c.Request.Context(), code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShareLinkNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "share link not found")
		case errors.Is(err, service.ErrShareLinkExpired):
			httputil.Error(c, http.StatusGone, "EXPIRED", "share link has expired")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	httputil.Success(c, gin.H{
		"code":        info.Code,
		"permission":  info.Permission,
		"has_password": info.HasPassword,
		"expires_at":   info.ExpiresAt,
		"created_at":   info.CreatedAt,
		"canvas_name":  canvasName,
		"thumbnail":    thumbnail,
		"owner_name":   ownerName,
	})
}

type validateShareInput struct {
	Password string `json:"password"`
}

func (h *ShareLinkHandler) Validate(c *gin.Context) {
	code := c.Param("code")

	var input validateShareInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	resp, err := h.shareLinkService.Validate(c.Request.Context(), code, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShareLinkNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "share link not found")
		case errors.Is(err, service.ErrShareLinkExpired):
			httputil.Error(c, http.StatusGone, "EXPIRED", "share link has expired")
		case errors.Is(err, service.ErrInvalidPassword):
			httputil.Error(c, http.StatusForbidden, "INVALID_PASSWORD", "invalid password")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	httputil.Success(c, resp)
}
