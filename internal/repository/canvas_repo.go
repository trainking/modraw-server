package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/trainking/modraw-server/internal/model"
)

type CanvasFilter struct {
	FolderID *string
	Search   string
	Page     int
	Limit    int
}

func (f CanvasFilter) Offset() int {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	return (f.Page - 1) * f.Limit
}

type CanvasRepository struct {
	db *sql.DB
}

func NewCanvasRepository(db *sql.DB) *CanvasRepository {
	return &CanvasRepository{db: db}
}

func (r *CanvasRepository) ListByOwner(ctx context.Context, ownerID string, filter CanvasFilter) ([]*model.CanvasListItem, int, error) {
	where := "WHERE owner_id = $1"
	args := []interface{}{ownerID}
	argIdx := 2

	if filter.FolderID != nil {
		where += fmt.Sprintf(" AND folder_id = $%d", argIdx)
		args = append(args, *filter.FolderID)
		argIdx++
	}
	if filter.Search != "" {
		where += fmt.Sprintf(" AND name ILIKE $%d", argIdx)
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM canvases " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count canvases: %w", err)
	}

	query := fmt.Sprintf(`SELECT id, owner_id, folder_id, name, thumbnail, file_size, created_at, updated_at
		FROM canvases %s ORDER BY updated_at DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset())

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list canvases: %w", err)
	}
	defer rows.Close()

	var result []*model.CanvasListItem
	for rows.Next() {
		c := &model.CanvasListItem{}
		var folderID sql.NullString
		if err := rows.Scan(&c.ID, &c.OwnerID, &folderID, &c.Name, &c.Thumbnail, &c.FileSize, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan canvas: %w", err)
		}
		if folderID.Valid {
			c.FolderID = &folderID.String
		}
		result = append(result, c)
	}
	return result, total, rows.Err()
}

func (r *CanvasRepository) GetByID(ctx context.Context, id string) (*model.Canvas, error) {
	query := `SELECT id, owner_id, folder_id, name, data, thumbnail, file_size, created_at, updated_at
		FROM canvases WHERE id = $1`
	c := &model.Canvas{}
	var folderID sql.NullString
	var dataBytes []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.OwnerID, &folderID, &c.Name, &dataBytes, &c.Thumbnail, &c.FileSize, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get canvas: %w", err)
	}
	if folderID.Valid {
		c.FolderID = &folderID.String
	}
	c.Data = json.RawMessage(dataBytes)
	return c, nil
}

func (r *CanvasRepository) Create(ctx context.Context, c *model.Canvas) error {
	query := `INSERT INTO canvases (owner_id, folder_id, name, data, thumbnail, file_size)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6) RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, c.OwnerID, c.FolderID, c.Name, string(c.Data), c.Thumbnail, c.FileSize).
		Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *CanvasRepository) Update(ctx context.Context, c *model.Canvas) error {
	query := `UPDATE canvases SET name = $1, folder_id = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, c.Name, c.FolderID, c.ID)
	return err
}

func (r *CanvasRepository) UpdateData(ctx context.Context, id string, data json.RawMessage, fileSize int64) error {
	query := `UPDATE canvases SET data = $1::jsonb, file_size = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, string(data), fileSize, id)
	return err
}

func (r *CanvasRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM canvases WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *CanvasRepository) Move(ctx context.Context, id, folderID string) error {
	query := `UPDATE canvases SET folder_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, folderID, id)
	return err
}
