package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/trainking/modraw-server/internal/model"
)

type ShareLinkRepository struct {
	db *sql.DB
}

func NewShareLinkRepository(db *sql.DB) *ShareLinkRepository {
	return &ShareLinkRepository{db: db}
}

func (r *ShareLinkRepository) Create(ctx context.Context, sl *model.ShareLink) error {
	query := `INSERT INTO share_links (canvas_id, created_by, permission, password_hash, code, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query, sl.CanvasID, sl.CreatedBy, sl.Permission, sl.PasswordHash, sl.Code, sl.ExpiresAt).
		Scan(&sl.ID, &sl.CreatedAt)
}

func (r *ShareLinkRepository) GetByCode(ctx context.Context, code string) (*model.ShareLink, error) {
	query := `SELECT id, canvas_id, created_by, permission, password_hash, code, expires_at, created_at
		FROM share_links WHERE code = $1`
	sl := &model.ShareLink{}
	var passwordHash sql.NullString
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&sl.ID, &sl.CanvasID, &sl.CreatedBy, &sl.Permission, &passwordHash, &sl.Code, &sl.ExpiresAt, &sl.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get share link by code: %w", err)
	}
	if passwordHash.Valid {
		sl.PasswordHash = passwordHash.String
	}
	return sl, nil
}

func (r *ShareLinkRepository) ListByCanvas(ctx context.Context, canvasID string) ([]*model.ShareLinkInfo, error) {
	query := `SELECT code, permission, password_hash, expires_at, created_at
		FROM share_links WHERE canvas_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, canvasID)
	if err != nil {
		return nil, fmt.Errorf("list share links: %w", err)
	}
	defer rows.Close()

	var result []*model.ShareLinkInfo
	for rows.Next() {
		info := &model.ShareLinkInfo{}
		var passwordHash string
		if err := rows.Scan(&info.Code, &info.Permission, &passwordHash, &info.ExpiresAt, &info.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan share link info: %w", err)
		}
		info.HasPassword = passwordHash != ""
		result = append(result, info)
	}
	return result, rows.Err()
}

func (r *ShareLinkRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM share_links WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete share link: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ShareLinkRepository) GetCanvasOwnerInfo(ctx context.Context, canvasID string) (canvasName, thumbnail, ownerName string, err error) {
	query := `SELECT c.name, c.thumbnail, u.nickname
		FROM canvases c JOIN users u ON u.id = c.owner_id WHERE c.id = $1`
	err = r.db.QueryRowContext(ctx, query, canvasID).Scan(&canvasName, &thumbnail, &ownerName)
	return
}

func (r *ShareLinkRepository) IsExpired(sl *model.ShareLink) bool {
	if sl.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*sl.ExpiresAt)
}
