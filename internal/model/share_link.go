package model

import "time"

type ShareLink struct {
	ID           string     `json:"id"`
	CanvasID     string     `json:"canvas_id"`
	CreatedBy    string     `json:"created_by"`
	Permission   string     `json:"permission"`
	PasswordHash string     `json:"-"`
	Code         string     `json:"code"`
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ShareLinkInfo struct {
	Code       string     `json:"code"`
	Permission string     `json:"permission"`
	HasPassword bool      `json:"has_password"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CanvasCollaborator struct {
	ID         string    `json:"id"`
	CanvasID   string    `json:"canvas_id"`
	UserID     string    `json:"user_id"`
	Permission string    `json:"permission"`
	AddedAt    time.Time `json:"added_at"`
	Email      string    `json:"email,omitempty"`
	Nickname   string    `json:"nickname,omitempty"`
}

type ValidateShareRequest struct {
	Password string `json:"password"`
}

type ValidateShareResponse struct {
	CanvasID   string `json:"canvas_id"`
	CanvasName string `json:"canvas_name"`
	Thumbnail  string `json:"thumbnail"`
	OwnerName  string `json:"owner_name"`
	Permission string `json:"permission"`
	ShareToken string `json:"share_token"`
}

type CreateShareLinkRequest struct {
	Permission string  `json:"permission" binding:"required"`
	Password   string  `json:"password"`
	ExpiresAt  *string `json:"expires_at"`
}
