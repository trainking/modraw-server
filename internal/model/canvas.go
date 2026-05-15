package model

import (
	"encoding/json"
	"time"
)

type Canvas struct {
	ID        string          `json:"id"`
	OwnerID   string          `json:"owner_id"`
	FolderID  *string         `json:"folder_id"`
	Name      string          `json:"name"`
	Data      json.RawMessage `json:"data"`
	Thumbnail string          `json:"thumbnail"`
	FileSize  int64           `json:"file_size"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CanvasListItem struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	FolderID  *string   `json:"folder_id"`
	Name      string    `json:"name"`
	Thumbnail string    `json:"thumbnail"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
