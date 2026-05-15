package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/trainking/modraw-server/internal/config"
	"github.com/trainking/modraw-server/internal/model"
	"github.com/trainking/modraw-server/internal/repository"
	jwtpkg "github.com/trainking/modraw-server/pkg/jwt"
)

var (
	ErrShareLinkNotFound = errors.New("share link not found")
	ErrShareLinkExpired  = errors.New("share link has expired")
	ErrInvalidPassword   = errors.New("invalid password")
)

type ShareLinkService struct {
	shareLinkRepo *repository.ShareLinkRepository
	canvasRepo    *repository.CanvasRepository
	cfg           *config.Config
}

func NewShareLinkService(
	shareLinkRepo *repository.ShareLinkRepository,
	canvasRepo *repository.CanvasRepository,
	cfg *config.Config,
) *ShareLinkService {
	return &ShareLinkService{
		shareLinkRepo: shareLinkRepo,
		canvasRepo:    canvasRepo,
		cfg:           cfg,
	}
}

func (s *ShareLinkService) List(ctx context.Context, canvasID, userID string) ([]*model.ShareLinkInfo, error) {
	canvas, err := s.canvasRepo.GetByID(ctx, canvasID)
	if err != nil {
		return nil, ErrCanvasNotFound
	}
	if canvas.OwnerID != userID {
		return nil, ErrCanvasAccessDenied
	}
	return s.shareLinkRepo.ListByCanvas(ctx, canvasID)
}

type CreateShareLinkInput struct {
	Permission string `json:"permission"`
	Password   string `json:"password"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

type CreateShareLinkOutput struct {
	Code       string     `json:"code"`
	Permission string     `json:"permission"`
	HasPassword bool      `json:"has_password"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

func (s *ShareLinkService) Create(ctx context.Context, canvasID, userID string, input CreateShareLinkInput) (*CreateShareLinkOutput, error) {
	canvas, err := s.canvasRepo.GetByID(ctx, canvasID)
	if err != nil {
		return nil, ErrCanvasNotFound
	}
	if canvas.OwnerID != userID {
		return nil, ErrCanvasAccessDenied
	}

	code, err := generateCode(8)
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	permission := input.Permission
	if permission != "readonly" && permission != "collaborate" {
		permission = "readonly"
	}

	var passwordHash string
	if input.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		passwordHash = string(h)
	}

	sl := &model.ShareLink{
		CanvasID:     canvasID,
		CreatedBy:    userID,
		Permission:   permission,
		PasswordHash: passwordHash,
		Code:         code,
		ExpiresAt:    input.ExpiresAt,
	}

	if err := s.shareLinkRepo.Create(ctx, sl); err != nil {
		return nil, fmt.Errorf("create share link: %w", err)
	}

	return &CreateShareLinkOutput{
		Code:        code,
		Permission:  permission,
		HasPassword: passwordHash != "",
		ExpiresAt:   input.ExpiresAt,
	}, nil
}

func (s *ShareLinkService) Delete(ctx context.Context, id, canvasID, userID string) error {
	canvas, err := s.canvasRepo.GetByID(ctx, canvasID)
	if err != nil {
		return ErrCanvasNotFound
	}
	if canvas.OwnerID != userID {
		return ErrCanvasAccessDenied
	}
	if err := s.shareLinkRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrShareLinkNotFound
		}
		return fmt.Errorf("delete share link: %w", err)
	}
	return nil
}

func (s *ShareLinkService) GetByCode(ctx context.Context, code string) (*model.ShareLinkInfo, string, string, string, error) {
	sl, err := s.shareLinkRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, "", "", "", ErrShareLinkNotFound
	}

	if s.shareLinkRepo.IsExpired(sl) {
		return nil, "", "", "", ErrShareLinkExpired
	}

	canvasName, thumbnail, ownerName, err := s.shareLinkRepo.GetCanvasOwnerInfo(ctx, sl.CanvasID)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("get canvas info: %w", err)
	}

	return &model.ShareLinkInfo{
		Code:        sl.Code,
		Permission:  sl.Permission,
		HasPassword: sl.PasswordHash != "",
		ExpiresAt:   sl.ExpiresAt,
		CreatedAt:   sl.CreatedAt,
	}, canvasName, thumbnail, ownerName, nil
}

func (s *ShareLinkService) Validate(ctx context.Context, code, password string) (*model.ValidateShareResponse, error) {
	sl, err := s.shareLinkRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, ErrShareLinkNotFound
	}

	if s.shareLinkRepo.IsExpired(sl) {
		return nil, ErrShareLinkExpired
	}

	if sl.PasswordHash != "" {
		if password == "" {
			return nil, ErrInvalidPassword
		}
		if err := bcrypt.CompareHashAndPassword([]byte(sl.PasswordHash), []byte(password)); err != nil {
			return nil, ErrInvalidPassword
		}
	}

	canvasName, thumbnail, ownerName, err := s.shareLinkRepo.GetCanvasOwnerInfo(ctx, sl.CanvasID)
	if err != nil {
		return nil, fmt.Errorf("get canvas info: %w", err)
	}

	shareToken, err := jwtpkg.GenerateShareToken(sl.CanvasID, sl.Permission, s.cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("generate share token: %w", err)
	}

	return &model.ValidateShareResponse{
		CanvasID:   sl.CanvasID,
		CanvasName: canvasName,
		Thumbnail:  thumbnail,
		OwnerName:  ownerName,
		Permission: sl.Permission,
		ShareToken: shareToken,
	}, nil
}

func generateCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}
