package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trainking/modraw-server/internal/middleware"
	"github.com/trainking/modraw-server/internal/model"
	"github.com/trainking/modraw-server/internal/service"
	"github.com/trainking/modraw-server/pkg/httputil"
)

type FolderHandler struct {
	folderService *service.FolderService
}

func NewFolderHandler(folderService *service.FolderService) *FolderHandler {
	return &FolderHandler{folderService: folderService}
}

func (h *FolderHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if c.Query("tree") == "true" {
		tree, err := h.folderService.ListTree(c.Request.Context(), userID)
		if err != nil {
			httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		httputil.Success(c, tree)
		return
	}

	folders, err := h.folderService.List(c.Request.Context(), userID)
	if err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httputil.Success(c, folders)
}

type createFolderInput struct {
	Name      string  `json:"name" binding:"required"`
	ParentID  *string `json:"parent_id"`
	SortOrder int     `json:"sort_order"`
}

func (h *FolderHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input createFolderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	folder := &model.Folder{
		UserID:    userID,
		Name:      input.Name,
		ParentID:  input.ParentID,
		SortOrder: input.SortOrder,
	}

	if err := h.folderService.Create(c.Request.Context(), folder); err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.Created(c, folder)
}

func (h *FolderHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	folder, err := h.folderService.Get(c.Request.Context(), id, userID)
	if err != nil {
		httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "folder not found")
		return
	}

	httputil.Success(c, folder)
}

type updateFolderInput struct {
	Name      string `json:"name"`
	SortOrder *int   `json:"sort_order"`
}

func (h *FolderHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	var input updateFolderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	folder, err := h.folderService.Get(c.Request.Context(), id, userID)
	if err != nil {
		httputil.Error(c, http.StatusNotFound, "NOT_FOUND", "folder not found")
		return
	}

	if input.Name != "" {
		folder.Name = input.Name
	}
	if input.SortOrder != nil {
		folder.SortOrder = *input.SortOrder
	}

	if err := h.folderService.Update(c.Request.Context(), folder); err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.Success(c, folder)
}

func (h *FolderHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	if err := h.folderService.Delete(c.Request.Context(), id, userID); err != nil {
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.NoContent(c)
}

type moveFolderInput struct {
	ParentID string `json:"parent_id"`
}

func (h *FolderHandler) Move(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	var input moveFolderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.Error(c, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}

	if err := h.folderService.Move(c.Request.Context(), id, input.ParentID, userID); err != nil {
		if errors.Is(err, service.ErrCircularReference) {
			httputil.Error(c, http.StatusBadRequest, "CIRCULAR_REFERENCE", err.Error())
			return
		}
		httputil.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httputil.Success(c, gin.H{"message": "moved"})
}
