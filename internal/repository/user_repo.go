package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/trainking/modraw-server/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT id, email, password_hash, nickname, avatar_url, created_at, updated_at FROM users WHERE id = $1`
	u := &model.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Nickname, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id, email, password_hash, nickname, avatar_url, created_at, updated_at FROM users WHERE email = $1`
	u := &model.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Nickname, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	query := `INSERT INTO users (email, password_hash, nickname, avatar_url) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, u.Email, u.PasswordHash, u.Nickname, u.AvatarURL).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepository) Update(ctx context.Context, u *model.User) error {
	query := `UPDATE users SET nickname = $1, avatar_url = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, u.Nickname, u.AvatarURL, u.ID)
	return err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, passwordHash, id)
	return err
}

func (r *UserRepository) SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (string, error) {
	var tokenID string
	query := `INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, userID, tokenHash, expiresAt).Scan(&tokenID)
	return tokenID, err
}

func (r *UserRepository) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	query := `UPDATE refresh_tokens SET revoked = TRUE WHERE id = $1 AND revoked = FALSE`
	_, err := r.db.ExecContext(ctx, query, tokenID)
	return err
}

func (r *UserRepository) IsRefreshTokenValid(ctx context.Context, userID, tokenID string) (bool, error) {
	var revoked bool
	var expiresAt time.Time
	query := `SELECT revoked, expires_at FROM refresh_tokens WHERE id = $1 AND user_id = $2`
	err := r.db.QueryRowContext(ctx, query, tokenID, userID).Scan(&revoked, &expiresAt)
	if err != nil {
		return false, err
	}
	return !revoked && time.Now().Before(expiresAt), nil
}
