package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trainking/modraw-server/internal/middleware"
	"github.com/trainking/modraw-server/internal/model"
	"github.com/trainking/modraw-server/internal/service"
	"github.com/trainking/modraw-server/pkg/httputil"
)

type LibraryHandler struct {
	libraryService *service.LibraryService
}

func NewLibraryHandler(libraryService *service.LibraryService) *LibraryHandler {
	return &LibraryHandler{libraryService: libraryService}
}

func (h *LibraryHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	libraries, err := h.libraryService.List(c.Request.Context(), userID)
	if err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.Success(c, libraries)
}

type createLibraryInput struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Data        json.RawMessage `json:"data"`
}

func (h *LibraryHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input createLibraryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	l := &model.Library{
		OwnerID:     userID,
		Name:        input.Name,
		Description: input.Description,
		Data:        input.Data,
	}

	if err := h.libraryService.Create(c.Request.Context(), l); err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.Created(c, l)
}

func (h *LibraryHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	l, err := h.libraryService.Get(c.Request.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLibraryNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "library not found")
		case errors.Is(err, service.ErrLibraryAccessDenied):
			httputil.Error(c, http.StatusForbidden, "FORBIDDEN", "access denied")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	httputil.Success(c, l)
}

type updateLibraryInput struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Data        json.RawMessage `json:"data"`
}

func (h *LibraryHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	var input updateLibraryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	l, err := h.libraryService.Update(c.Request.Context(), id, userID, input.Name, input.Description, input.Data)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLibraryNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "library not found")
		case errors.Is(err, service.ErrLibraryAccessDenied):
			httputil.Error(c, http.StatusForbidden, "FORBIDDEN", "access denied")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	httputil.Success(c, l)
}

func (h *LibraryHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	if err := h.libraryService.Delete(c.Request.Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrLibraryNotFound):
			httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "library not found")
		case errors.Is(err, service.ErrLibraryAccessDenied):
			httputil.Error(c, http.StatusForbidden, "FORBIDDEN", "access denied")
		default:
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	httputil.NoContent(c)
}
