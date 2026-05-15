package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trainking/modraw-server/internal/middleware"
	"github.com/trainking/modraw-server/internal/service"
	"github.com/trainking/modraw-server/pkg/httputil"
)

type CollaboratorHandler struct {
	collaboratorService *service.CollaboratorService
}

func NewCollaboratorHandler(collaboratorService *service.CollaboratorService) *CollaboratorHandler {
	return &CollaboratorHandler{collaboratorService: collaboratorService}
}

func (h *CollaboratorHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	canvasID := c.Param("id")

	collaborators, err := h.collaboratorService.List(c.Request.Context(), canvasID, userID)
	if err != nil {
		if errors.Is(err, service.ErrCanvasAccessDenied) {
			httputil.Error(c, http.StatusForbidden, "FORBIDDEN", "access denied")
			return
		}
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.Success(c, collaborators)
}

type addCollaboratorInput struct {
	Email      string `json:"email" binding:"required"`
	Permission string `json:"permission" binding:"required"`
}

func (h *CollaboratorHandler) Add(c *gin.Context) {
	userID := middleware.GetUserID(c)
	canvasID := c.Param("id")

	var input addCollaboratorInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	if input.Permission != "readonly" && input.Permission != "collaborate" {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", "permission must be 'readonly' or 'collaborate'")
		return
	}

	collaborator, err := h.collaboratorService.Add(c.Request.Context(), canvasID, userID, input.Email, input.Permission)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotOwner):
			httputil.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			httputil.Error(c, http.StatusBadRequest, "INVALID", "user not found")
		}
		return
	}

	httputil.Created(c, collaborator)
}

type updateCollaboratorInput struct {
	Permission string `json:"permission" binding:"required"`
}

func (h *CollaboratorHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	canvasID := c.Param("id")
	targetUserID := c.Param("user_id")

	var input updateCollaboratorInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	if input.Permission != "readonly" && input.Permission != "collaborate" {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", "permission must be 'readonly' or 'collaborate'")
		return
	}

	if err := h.collaboratorService.Update(c.Request.Context(), canvasID, userID, targetUserID, input.Permission); err != nil {
		switch {
		case errors.Is(err, service.ErrNotOwner):
			httputil.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		case errors.Is(err, service.ErrCollaboratorNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "collaborator not found")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	httputil.Success(c, gin.H{"message": "updated"})
}

func (h *CollaboratorHandler) Remove(c *gin.Context) {
	userID := middleware.GetUserID(c)
	canvasID := c.Param("id")
	targetUserID := c.Param("user_id")

	if err := h.collaboratorService.Remove(c.Request.Context(), canvasID, userID, targetUserID); err != nil {
		switch {
		case errors.Is(err, service.ErrNotOwner):
			httputil.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		case errors.Is(err, service.ErrCollaboratorNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "collaborator not found")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	httputil.NoContent(c)
}
