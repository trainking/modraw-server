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
	ErrLibraryNotFound   = errors.New("library not found")
	ErrLibraryAccessDenied = errors.New("access denied")
)

type LibraryService struct {
	libraryRepo *repository.LibraryRepository
}

func NewLibraryService(libraryRepo *repository.LibraryRepository) *LibraryService {
	return &LibraryService{libraryRepo: libraryRepo}
}

func (s *LibraryService) List(ctx context.Context, userID string) ([]*model.Library, error) {
	return s.libraryRepo.ListByOwner(ctx, userID)
}

func (s *LibraryService) Get(ctx context.Context, id, userID string) (*model.Library, error) {
	l, err := s.libraryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrLibraryNotFound
	}
	if l.OwnerID != userID {
		return nil, ErrLibraryAccessDenied
	}
	return l, nil
}

func (s *LibraryService) Create(ctx context.Context, l *model.Library) error {
	if l.Data == nil {
		l.Data = json.RawMessage("{}")
	}
	return s.libraryRepo.Create(ctx, l)
}

func (s *LibraryService) Update(ctx context.Context, id, userID string, name, description string, data json.RawMessage) (*model.Library, error) {
	l, err := s.libraryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrLibraryNotFound
	}
	if l.OwnerID != userID {
		return nil, ErrLibraryAccessDenied
	}

	if name != "" {
		l.Name = name
	}
	if description != "" {
		l.Description = description
	}
	if data != nil {
		l.Data = data
	}

	if err := s.libraryRepo.Update(ctx, l); err != nil {
		return nil, fmt.Errorf("update library: %w", err)
	}
	return l, nil
}

func (s *LibraryService) Delete(ctx context.Context, id, userID string) error {
	l, err := s.libraryRepo.GetByID(ctx, id)
	if err != nil {
		return ErrLibraryNotFound
	}
	if l.OwnerID != userID {
		return ErrLibraryAccessDenied
	}
	return s.libraryRepo.Delete(ctx, id)
}
