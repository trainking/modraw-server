package ws

import "encoding/json"

const (
	MsgTypeJoin      = "join"
	MsgTypeLeave     = "leave"
	MsgTypeOp        = "op"
	MsgTypeCursor    = "cursor"
	MsgTypeAwareness = "awareness"
	MsgTypeSave      = "save"
	MsgTypePing      = "ping"

	MsgTypeJoined = "joined"
	MsgTypeLeft   = "left"
	MsgTypeAck    = "ack"
	MsgTypeSaved  = "saved"
	MsgTypePong   = "pong"
	MsgTypeError  = "error"
)

type WsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type JoinPayload struct {
	CanvasID   string `json:"canvas_id"`
	ShareToken string `json:"share_token,omitempty"`
}

type LeavePayload struct {
	CanvasID string `json:"canvas_id"`
}

type OpPayload struct {
	CanvasID  string          `json:"canvas_id"`
	Seq       int64           `json:"seq"`
	Operation json.RawMessage `json:"operation"`
}

type CursorPayload struct {
	CanvasID string          `json:"canvas_id"`
	Position json.RawMessage `json:"position"`
}

type AwarenessPayload struct {
	CanvasID string          `json:"canvas_id"`
	State    json.RawMessage `json:"state"`
}

type SavePayload struct {
	CanvasID string          `json:"canvas_id"`
	Data     json.RawMessage `json:"data"`
}

type JoinedPayload struct {
	CanvasID string       `json:"canvas_id"`
	Clients  []ClientInfo `json:"clients"`
}

type LeftPayload struct {
	CanvasID string `json:"canvas_id"`
	UserID   string `json:"user_id"`
}

type OpRelayPayload struct {
	CanvasID  string          `json:"canvas_id"`
	UserID    string          `json:"user_id"`
	Seq       int64           `json:"seq"`
	Operation json.RawMessage `json:"operation"`
}

type CursorRelayPayload struct {
	CanvasID string          `json:"canvas_id"`
	UserID   string          `json:"user_id"`
	Position json.RawMessage `json:"position"`
}

type AwarenessRelayPayload struct {
	CanvasID string          `json:"canvas_id"`
	UserID   string          `json:"user_id"`
	State    json.RawMessage `json:"state"`
}

type AckPayload struct {
	CanvasID string `json:"canvas_id"`
	Seq      int64  `json:"seq"`
}

type SavedPayload struct {
	CanvasID  string `json:"canvas_id"`
	UpdatedAt string `json:"updated_at"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ClientInfo struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}
