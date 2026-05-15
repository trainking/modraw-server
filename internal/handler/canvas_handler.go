package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/trainking/modraw-server/internal/middleware"
	"github.com/trainking/modraw-server/internal/model"
	"github.com/trainking/modraw-server/internal/repository"
	"github.com/trainking/modraw-server/internal/service"
	"github.com/trainking/modraw-server/pkg/httputil"
)

type CanvasHandler struct {
	canvasService *service.CanvasService
}

func NewCanvasHandler(canvasService *service.CanvasService) *CanvasHandler {
	return &CanvasHandler{canvasService: canvasService}
}

func (h *CanvasHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}

	filter := repository.CanvasFilter{
		Page:  page,
		Limit: limit,
	}

	if folderID := c.Query("folder_id"); folderID != "" {
		filter.FolderID = &folderID
	}
	if search := c.Query("search"); search != "" {
		filter.Search = search
	}

	canvases, total, err := h.canvasService.List(c.Request.Context(), userID, filter)
	if err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.Paginated(c, canvases, page, limit, total)
}

type createCanvasInput struct {
	Name     string          `json:"name"`
	FolderID *string         `json:"folder_id"`
	Data     json.RawMessage `json:"data"`
}

func (h *CanvasHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input createCanvasInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	canvas := &model.Canvas{
		OwnerID:  userID,
		Name:     input.Name,
		FolderID: input.FolderID,
		Data:     input.Data,
	}

	if err := h.canvasService.Create(c.Request.Context(), canvas); err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.Created(c, canvas)
}

func (h *CanvasHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	canvas, err := h.canvasService.Get(c.Request.Context(), id, userID)
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

	httputil.Success(c, canvas)
}

type updateCanvasInput struct {
	Name     string  `json:"name"`
	FolderID *string `json:"folder_id"`
}

func (h *CanvasHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	var input updateCanvasInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	canvas, err := h.canvasService.Update(c.Request.Context(), id, userID, input.Name, input.FolderID)
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

	httputil.Success(c, canvas)
}

type saveCanvasDataInput struct {
	Data     json.RawMessage `json:"data" binding:"required"`
	FileSize int64           `json:"file_size"`
}

func (h *CanvasHandler) SaveData(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	var input saveCanvasDataInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	if input.FileSize == 0 {
		input.FileSize = int64(len(input.Data))
	}

	if err := h.canvasService.SaveData(c.Request.Context(), id, userID, input.Data, input.FileSize); err != nil {
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

	httputil.Success(c, gin.H{"message": "saved"})
}

func (h *CanvasHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	if err := h.canvasService.Delete(c.Request.Context(), id, userID); err != nil {
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

	httputil.NoContent(c)
}

type moveCanvasInput struct {
	FolderID string `json:"folder_id"`
}

func (h *CanvasHandler) Move(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	var input moveCanvasInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	if err := h.canvasService.Move(c.Request.Context(), id, userID, input.FolderID); err != nil {
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

	httputil.Success(c, gin.H{"message": "moved"})
}
