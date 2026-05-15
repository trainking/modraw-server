package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/trainking/modraw-server/internal/model"
	"github.com/trainking/modraw-server/internal/repository"
)

var (
	ErrCanvasNotFound   = errors.New("canvas not found")
	ErrCanvasAccessDenied = errors.New("access denied")
)

type CanvasService struct {
	canvasRepo *repository.CanvasRepository
}

func NewCanvasService(canvasRepo *repository.CanvasRepository) *CanvasService {
	return &CanvasService{canvasRepo: canvasRepo}
}

func (s *CanvasService) List(ctx context.Context, userID string, filter repository.CanvasFilter) ([]*model.CanvasListItem, int, error) {
	return s.canvasRepo.ListByOwner(ctx, userID, filter)
}

func (s *CanvasService) Get(ctx context.Context, id, userID string) (*model.Canvas, error) {
	canvas, err := s.canvasRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrCanvasNotFound
	}
	if canvas.OwnerID != userID {
		return nil, ErrCanvasAccessDenied
	}
	return canvas, nil
}

func (s *CanvasService) Create(ctx context.Context, c *model.Canvas) error {
	if c.Data == nil {
		c.Data = json.RawMessage("{}")
	}
	if c.Name == "" {
		c.Name = "Untitled"
	}
	return s.canvasRepo.Create(ctx, c)
}

func (s *CanvasService) Update(ctx context.Context, id, userID string, name string, folderID *string) (*model.Canvas, error) {
	canvas, err := s.canvasRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrCanvasNotFound
	}
	if canvas.OwnerID != userID {
		return nil, ErrCanvasAccessDenied
	}

	canvas.Name = name
	canvas.FolderID = folderID
	if err := s.canvasRepo.Update(ctx, canvas); err != nil {
		return nil, fmt.Errorf("update canvas: %w", err)
	}
	return canvas, nil
}

func (s *CanvasService) SaveData(ctx context.Context, id, userID string, data json.RawMessage, fileSize int64) error {
	canvas, err := s.canvasRepo.GetByID(ctx, id)
	if err != nil {
		return ErrCanvasNotFound
	}
	if canvas.OwnerID != userID {
		return ErrCanvasAccessDenied
	}
	return s.canvasRepo.UpdateData(ctx, id, data, fileSize)
}

func (s *CanvasService) Delete(ctx context.Context, id, userID string) error {
	canvas, err := s.canvasRepo.GetByID(ctx, id)
	if err != nil {
		return ErrCanvasNotFound
	}
	if canvas.OwnerID != userID {
		return ErrCanvasAccessDenied
	}
	return s.canvasRepo.Delete(ctx, id)
}

func (s *CanvasService) Move(ctx context.Context, id, userID, folderID string) error {
	canvas, err := s.canvasRepo.GetByID(ctx, id)
	if err != nil {
		return ErrCanvasNotFound
	}
	if canvas.OwnerID != userID {
		return ErrCanvasAccessDenied
	}
	return s.canvasRepo.Move(ctx, id, folderID)
}
