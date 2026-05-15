package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/trainking/modraw-server/internal/model"
)

type LibraryRepository struct {
	db *sql.DB
}

func NewLibraryRepository(db *sql.DB) *LibraryRepository {
	return &LibraryRepository{db: db}
}

func (r *LibraryRepository) ListByOwner(ctx context.Context, ownerID string) ([]*model.Library, error) {
	query := `SELECT id, owner_id, name, description, data, created_at, updated_at
		FROM libraries WHERE owner_id = $1 ORDER BY updated_at DESC`
	rows, err := r.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list libraries: %w", err)
	}
	defer rows.Close()

	var result []*model.Library
	for rows.Next() {
		l := &model.Library{}
		var dataBytes []byte
		if err := rows.Scan(&l.ID, &l.OwnerID, &l.Name, &l.Description, &dataBytes, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan library: %w", err)
		}
		l.Data = json.RawMessage(dataBytes)
		result = append(result, l)
	}
	return result, rows.Err()
}

func (r *LibraryRepository) GetByID(ctx context.Context, id string) (*model.Library, error) {
	query := `SELECT id, owner_id, name, description, data, created_at, updated_at
		FROM libraries WHERE id = $1`
	l := &model.Library{}
	var dataBytes []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&l.ID, &l.OwnerID, &l.Name, &l.Description, &dataBytes, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get library: %w", err)
	}
	l.Data = json.RawMessage(dataBytes)
	return l, nil
}

func (r *LibraryRepository) Create(ctx context.Context, l *model.Library) error {
	query := `INSERT INTO libraries (owner_id, name, description, data)
		VALUES ($1, $2, $3, $4::jsonb) RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, l.OwnerID, l.Name, l.Description, string(l.Data)).
		Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
}

func (r *LibraryRepository) Update(ctx context.Context, l *model.Library) error {
	query := `UPDATE libraries SET name = $1, description = $2, data = $3::jsonb, updated_at = NOW() WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, l.Name, l.Description, string(l.Data), l.ID)
	return err
}

func (r *LibraryRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM libraries WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
