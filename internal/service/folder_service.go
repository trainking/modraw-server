package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/trainking/modraw-server/internal/model"
	"github.com/trainking/modraw-server/internal/repository"
)

var (
	ErrFolderNotFound     = errors.New("folder not found")
	ErrCircularReference  = errors.New("cannot move a folder into itself or its descendants")
)

type FolderService struct {
	folderRepo *repository.FolderRepository
}

func NewFolderService(folderRepo *repository.FolderRepository) *FolderService {
	return &FolderService{folderRepo: folderRepo}
}

func (s *FolderService) List(ctx context.Context, userID string) ([]*model.Folder, error) {
	return s.folderRepo.ListByUser(ctx, userID)
}

func (s *FolderService) ListTree(ctx context.Context, userID string) ([]*model.FolderTreeNode, error) {
	folders, err := s.folderRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return BuildTree(folders), nil
}

func (s *FolderService) Get(ctx context.Context, id, userID string) (*model.Folder, error) {
	return s.folderRepo.GetByID(ctx, id, userID)
}

func (s *FolderService) Create(ctx context.Context, f *model.Folder) error {
	return s.folderRepo.Create(ctx, f)
}

func (s *FolderService) Update(ctx context.Context, f *model.Folder) error {
	return s.folderRepo.Update(ctx, f)
}

func (s *FolderService) Delete(ctx context.Context, id, userID string) error {
	return s.folderRepo.Delete(ctx, id, userID)
}

func (s *FolderService) Move(ctx context.Context, id, newParentID, userID string) error {
	if id == newParentID {
		return ErrCircularReference
	}

	if newParentID != "" {
		descendants, err := s.folderRepo.GetDescendantIDs(ctx, id)
		if err != nil {
			return fmt.Errorf("check descendants: %w", err)
		}
		for _, desc := range descendants {
			if desc == newParentID {
				return ErrCircularReference
			}
		}
	}

	return s.folderRepo.Move(ctx, id, newParentID, userID)
}

func BuildTree(folders []*model.Folder) []*model.FolderTreeNode {
	byID := make(map[string]*model.FolderTreeNode, len(folders))
	for _, f := range folders {
		byID[f.ID] = &model.FolderTreeNode{Folder: *f, Children: make([]*model.FolderTreeNode, 0)}
	}

	roots := make([]*model.FolderTreeNode, 0)
	for _, node := range byID {
		if node.ParentID == nil || *node.ParentID == "" {
			roots = append(roots, node)
		} else if parent, ok := byID[*node.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}
	return roots
}
