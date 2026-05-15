package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/trainking/modraw-server/internal/model"
)

type CollaboratorRepository struct {
	db *sql.DB
}

func NewCollaboratorRepository(db *sql.DB) *CollaboratorRepository {
	return &CollaboratorRepository{db: db}
}

func (r *CollaboratorRepository) List(ctx context.Context, canvasID string) ([]*model.CanvasCollaborator, error) {
	query := `SELECT cc.id, cc.canvas_id, cc.user_id, cc.permission, cc.added_at, u.email, u.nickname
		FROM canvas_collaborators cc
		JOIN users u ON u.id = cc.user_id
		WHERE cc.canvas_id = $1
		ORDER BY cc.added_at ASC`
	rows, err := r.db.QueryContext(ctx, query, canvasID)
	if err != nil {
		return nil, fmt.Errorf("list collaborators: %w", err)
	}
	defer rows.Close()

	var result []*model.CanvasCollaborator
	for rows.Next() {
		c := &model.CanvasCollaborator{}
		if err := rows.Scan(&c.ID, &c.CanvasID, &c.UserID, &c.Permission, &c.AddedAt, &c.Email, &c.Nickname); err != nil {
			return nil, fmt.Errorf("scan collaborator: %w", err)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *CollaboratorRepository) Get(ctx context.Context, canvasID, userID string) (*model.CanvasCollaborator, error) {
	query := `SELECT id, canvas_id, user_id, permission, added_at FROM canvas_collaborators
		WHERE canvas_id = $1 AND user_id = $2`
	c := &model.CanvasCollaborator{}
	err := r.db.QueryRowContext(ctx, query, canvasID, userID).Scan(&c.ID, &c.CanvasID, &c.UserID, &c.Permission, &c.AddedAt)
	if err != nil {
		return nil, fmt.Errorf("get collaborator: %w", err)
	}
	return c, nil
}

func (r *CollaboratorRepository) Add(ctx context.Context, c *model.CanvasCollaborator) error {
	query := `INSERT INTO canvas_collaborators (canvas_id, user_id, permission)
		VALUES ($1, $2, $3)
		ON CONFLICT (canvas_id, user_id) DO UPDATE SET permission = $3
		RETURNING id, added_at`
	return r.db.QueryRowContext(ctx, query, c.CanvasID, c.UserID, c.Permission).Scan(&c.ID, &c.AddedAt)
}

func (r *CollaboratorRepository) Update(ctx context.Context, canvasID, userID, permission string) error {
	query := `UPDATE canvas_collaborators SET permission = $1 WHERE canvas_id = $2 AND user_id = $3`
	result, err := r.db.ExecContext(ctx, query, permission, canvasID, userID)
	if err != nil {
		return fmt.Errorf("update collaborator: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *CollaboratorRepository) Remove(ctx context.Context, canvasID, userID string) error {
	query := `DELETE FROM canvas_collaborators WHERE canvas_id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, canvasID, userID)
	if err != nil {
		return fmt.Errorf("remove collaborator: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *CollaboratorRepository) GetPermission(ctx context.Context, canvasID, userID string) (string, error) {
	query := `SELECT permission FROM canvas_collaborators WHERE canvas_id = $1 AND user_id = $2`
	var perm string
	err := r.db.QueryRowContext(ctx, query, canvasID, userID).Scan(&perm)
	if err != nil {
		return "", fmt.Errorf("get permission: %w", err)
	}
	return perm, nil
}
