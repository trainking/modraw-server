package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/trainking/modraw-server/internal/model"
	"github.com/trainking/modraw-server/internal/repository"
)

var (
	ErrCollaboratorNotFound = errors.New("collaborator not found")
	ErrNotOwner             = errors.New("only the canvas owner can manage collaborators")
)

type CollaboratorService struct {
	collaboratorRepo *repository.CollaboratorRepository
	canvasRepo       *repository.CanvasRepository
	userRepo         *repository.UserRepository
}

func NewCollaboratorService(
	collaboratorRepo *repository.CollaboratorRepository,
	canvasRepo *repository.CanvasRepository,
	userRepo *repository.UserRepository,
) *CollaboratorService {
	return &CollaboratorService{
		collaboratorRepo: collaboratorRepo,
		canvasRepo:       canvasRepo,
		userRepo:         userRepo,
	}
}

func (s *CollaboratorService) List(ctx context.Context, canvasID, userID string) ([]*model.CanvasCollaborator, error) {
	if !s.canAccess(ctx, canvasID, userID) {
		return nil, ErrCanvasAccessDenied
	}
	return s.collaboratorRepo.List(ctx, canvasID)
}

func (s *CollaboratorService) Add(ctx context.Context, canvasID, ownerID, email, permission string) (*model.CanvasCollaborator, error) {
	if !s.isOwner(ctx, canvasID, ownerID) {
		return nil, ErrNotOwner
	}

	targetUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	c := &model.CanvasCollaborator{
		CanvasID:   canvasID,
		UserID:     targetUser.ID,
		Permission: permission,
	}
	if err := s.collaboratorRepo.Add(ctx, c); err != nil {
		return nil, fmt.Errorf("add collaborator: %w", err)
	}

	c.Email = targetUser.Email
	c.Nickname = targetUser.Nickname
	return c, nil
}

func (s *CollaboratorService) Update(ctx context.Context, canvasID, ownerID, userID, permission string) error {
	if !s.isOwner(ctx, canvasID, ownerID) {
		return ErrNotOwner
	}
	if err := s.collaboratorRepo.Update(ctx, canvasID, userID, permission); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCollaboratorNotFound
		}
		return fmt.Errorf("update collaborator: %w", err)
	}
	return nil
}

func (s *CollaboratorService) Remove(ctx context.Context, canvasID, ownerID, userID string) error {
	if !s.isOwner(ctx, canvasID, ownerID) {
		return ErrNotOwner
	}
	if err := s.collaboratorRepo.Remove(ctx, canvasID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCollaboratorNotFound
		}
		return fmt.Errorf("remove collaborator: %w", err)
	}
	return nil
}

func (s *CollaboratorService) isOwner(ctx context.Context, canvasID, userID string) bool {
	canvas, err := s.canvasRepo.GetByID(ctx, canvasID)
	if err != nil {
		return false
	}
	return canvas.OwnerID == userID
}

func (s *CollaboratorService) canAccess(ctx context.Context, canvasID, userID string) bool {
	if s.isOwner(ctx, canvasID, userID) {
		return true
	}
	_, err := s.collaboratorRepo.Get(ctx, canvasID, userID)
	return err == nil
}
