package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/trainking/modraw-server/internal/config"
	"github.com/trainking/modraw-server/internal/model"
	"github.com/trainking/modraw-server/internal/repository"
	"github.com/trainking/modraw-server/pkg/jwt"
)

var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrInvalidCreds = errors.New("invalid email or password")
	ErrWeakPassword = errors.New("password must be 8-72 characters")
	ErrInvalidToken = errors.New("invalid or expired refresh token")
	ErrTokenRevoked = errors.New("refresh token revoked")
)

type AuthService struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

type RegisterInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthTokens struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         *model.User `json:"user"`
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthTokens, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if len(input.Password) < 8 || len(input.Password) > 72 {
		return nil, ErrWeakPassword
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Email:        input.Email,
		PasswordHash: string(passwordHash),
		Nickname:     input.Nickname,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.generateTokens(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthTokens, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCreds
	}

	return s.generateTokens(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, refreshTokenStr string) (*AuthTokens, error) {
	claims, err := jwt.ValidateRefreshToken(refreshTokenStr, s.cfg.JWTSecret)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// claims.TokenID is the jti claim — matches the jti column in refresh_tokens.
	valid, err := s.userRepo.IsRefreshTokenValid(ctx, claims.UserID, claims.TokenID)
	if err != nil || !valid {
		return nil, ErrTokenRevoked
	}

	if err := s.userRepo.RevokeRefreshToken(ctx, claims.TokenID); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return s.generateTokens(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshTokenStr string) error {
	claims, err := jwt.ValidateRefreshToken(refreshTokenStr, s.cfg.JWTSecret)
	if err != nil {
		return nil // token already invalid, consider logout successful
	}

	_ = s.userRepo.RevokeRefreshToken(ctx, claims.TokenID)
	return nil
}

func (s *AuthService) generateTokens(ctx context.Context, user *model.User) (*AuthTokens, error) {
	access, err := jwt.GenerateAccessToken(user.ID, user.Email, user.Nickname, s.cfg.JWTSecret, s.cfg.AccessTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refresh, tokenID, err := jwt.GenerateRefreshToken(user.ID, s.cfg.JWTSecret, s.cfg.RefreshTTL)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// tokenID is the JWT jti claim — store it in the refresh_tokens.jti column
	// so that RevokeRefreshToken and IsRefreshTokenValid can look it up.
	tokenHash := sha256Hex(refresh)
	expiresAt := refreshTokenExpiry(s.cfg.RefreshTTL)

	if err := s.userRepo.SaveRefreshToken(ctx, user.ID, tokenID, tokenHash, expiresAt); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &AuthTokens{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         user,
	}, nil
}

func sha256Hex(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

func refreshTokenExpiry(ttl time.Duration) time.Time {
	return time.Now().Add(ttl)
}
