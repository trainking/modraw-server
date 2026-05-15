package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/trainking/modraw-server/internal/model"
)

type FolderRepository struct {
	db *sql.DB
}

func NewFolderRepository(db *sql.DB) *FolderRepository {
	return &FolderRepository{db: db}
}

func (r *FolderRepository) ListByUser(ctx context.Context, userID string) ([]*model.Folder, error) {
	query := `SELECT id, user_id, name, parent_id, sort_order, created_at, updated_at
		FROM folders WHERE user_id = $1 ORDER BY sort_order ASC, name ASC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	defer rows.Close()

	var folders []*model.Folder
	for rows.Next() {
		f := &model.Folder{}
		var parentID sql.NullString
		if err := rows.Scan(&f.ID, &f.UserID, &f.Name, &parentID, &f.SortOrder, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan folder: %w", err)
		}
		if parentID.Valid {
			f.ParentID = &parentID.String
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func (r *FolderRepository) GetByID(ctx context.Context, id, userID string) (*model.Folder, error) {
	query := `SELECT id, user_id, name, parent_id, sort_order, created_at, updated_at
		FROM folders WHERE id = $1 AND user_id = $2`
	f := &model.Folder{}
	var parentID sql.NullString
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&f.ID, &f.UserID, &f.Name, &parentID, &f.SortOrder, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get folder: %w", err)
	}
	if parentID.Valid {
		f.ParentID = &parentID.String
	}
	return f, nil
}

func (r *FolderRepository) Create(ctx context.Context, f *model.Folder) error {
	query := `INSERT INTO folders (user_id, name, parent_id, sort_order)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, f.UserID, f.Name, f.ParentID, f.SortOrder).
		Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *FolderRepository) Update(ctx context.Context, f *model.Folder) error {
	query := `UPDATE folders SET name = $1, sort_order = $2, updated_at = NOW() WHERE id = $3 AND user_id = $4`
	_, err := r.db.ExecContext(ctx, query, f.Name, f.SortOrder, f.ID, f.UserID)
	return err
}

func (r *FolderRepository) Delete(ctx context.Context, id, userID string) error {
	query := `DELETE FROM folders WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}

func (r *FolderRepository) Move(ctx context.Context, id, newParentID, userID string) error {
	query := `UPDATE folders SET parent_id = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`
	_, err := r.db.ExecContext(ctx, query, newParentID, id, userID)
	return err
}

func (r *FolderRepository) GetChildren(ctx context.Context, parentID string) ([]*model.Folder, error) {
	query := `SELECT id, user_id, name, parent_id, sort_order, created_at, updated_at
		FROM folders WHERE parent_id = $1`
	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("get children: %w", err)
	}
	defer rows.Close()

	var folders []*model.Folder
	for rows.Next() {
		f := &model.Folder{}
		var pID sql.NullString
		if err := rows.Scan(&f.ID, &f.UserID, &f.Name, &pID, &f.SortOrder, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan child folder: %w", err)
		}
		if pID.Valid {
			f.ParentID = &pID.String
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func (r *FolderRepository) GetDescendantIDs(ctx context.Context, parentID string) ([]string, error) {
	var ids []string
	children, err := r.GetChildren(ctx, parentID)
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		ids = append(ids, child.ID)
		desc, err := r.GetDescendantIDs(ctx, child.ID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, desc...)
	}
	return ids, nil
}
