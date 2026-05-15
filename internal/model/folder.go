package model

import "time"

type Folder struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parent_id"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FolderTreeNode struct {
	Folder
	Children []*FolderTreeNode `json:"children"`
}
