package model

import (
	"encoding/json"
	"time"
)

type Library struct {
	ID          string          `json:"id"`
	OwnerID     string          `json:"owner_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Data        json.RawMessage `json:"data"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type LibraryExport struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Data        json.RawMessage `json:"data"`
}
