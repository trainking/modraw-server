package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/trainking/modraw-server/internal/crdt"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4 * 1024 * 1024 // 4MB
)

type SaveFunc func(canvasID, userID string, data json.RawMessage) error

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	userID      string
	email       string
	nickname    string
	avatarURL   string
	canvasID    string
	permission  string
	saveHandler SaveFunc
	mu          sync.Mutex
}

func NewClient(hub *Hub, conn *websocket.Conn, userID, email, nickname, avatarURL string) *Client {
	return &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, 256),
		userID:    userID,
		email:     email,
		nickname:  nickname,
		avatarURL: avatarURL,
	}
}

func (c *Client) SetSaveHandler(handler SaveFunc) {
	c.saveHandler = handler
}

func (c *Client) ReadPump(accessChecker func(canvasID, userID, shareToken string) (string, error)) {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[ws] read error: %v", err)
			}
			break
		}

		var msg WsMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			c.sendError("PARSE_ERROR", "invalid message format")
			continue
		}

		c.handleMessage(msg, accessChecker)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg WsMessage, accessChecker func(canvasID, userID, shareToken string) (string, error)) {
	switch msg.Type {
	case MsgTypeJoin:
		var p JoinPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			c.sendError("PARSE_ERROR", "invalid join payload")
			return
		}
		c.handleJoin(p, accessChecker)

	case MsgTypeLeave:
		var p LeavePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return
		}
		c.handleLeave(p)

	case MsgTypeOp:
		var p OpPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return
		}
		c.handleOp(p)

	case MsgTypeCursor:
		var p CursorPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return
		}
		c.handleCursor(p)

	case MsgTypeAwareness:
		var p AwarenessPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return
		}
		c.handleAwareness(p)

	case MsgTypeSave:
		var p SavePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return
		}
		c.handleSave(p)

	case MsgTypePing:
		c.sendJSON(WsMessage{Type: MsgTypePong})
	}
}

func (c *Client) handleJoin(p JoinPayload, accessChecker func(canvasID, userID, shareToken string) (string, error)) {
	if c.canvasID != "" {
		// Leave current room first
		c.hub.Unregister(c)
	}

	perm, err := accessChecker(p.CanvasID, c.userID, p.ShareToken)
	if err != nil {
		c.sendError("FORBIDDEN", err.Error())
		return
	}

	c.canvasID = p.CanvasID
	c.permission = perm

	c.hub.Register(c)
}

func (c *Client) handleLeave(p LeavePayload) {
	if c.canvasID != "" {
		c.hub.Unregister(c)
		c.canvasID = ""
		c.permission = ""
	}
}

func (c *Client) handleOp(p OpPayload) {
	if c.permission != "collaborate" {
		c.sendError("FORBIDDEN", "read-only access")
		return
	}

	// Parse and apply the CRDT operation to the room state
	var op crdt.Operation
	if err := json.Unmarshal(p.Operation, &op); err != nil {
		c.sendError("PARSE_ERROR", "invalid operation format")
		return
	}

	c.hub.mu.RLock()
	room, ok := c.hub.rooms[c.canvasID]
	c.hub.mu.RUnlock()
	if !ok {
		return
	}

	if !room.ApplyOp(op) {
		// Stale or conflicting operation — still ack but don't broadcast
		c.sendJSON(WsMessage{
			Type: MsgTypeAck,
			Payload: marshalPayload(AckPayload{
				CanvasID: c.canvasID,
				Seq:      p.Seq,
			}),
		})
		return
	}

	relay := OpRelayPayload{
		CanvasID:  c.canvasID,
		UserID:    c.userID,
		Seq:       p.Seq,
		Operation: p.Operation,
	}
	room.broadcastOpExcept(WsMessage{
		Type:    MsgTypeOp,
		Payload: marshalPayload(relay),
	}, c.userID)

	c.sendJSON(WsMessage{
		Type: MsgTypeAck,
		Payload: marshalPayload(AckPayload{
			CanvasID: c.canvasID,
			Seq:      p.Seq,
		}),
	})
}

func (c *Client) handleCursor(p CursorPayload) {
	c.hub.mu.RLock()
	room, ok := c.hub.rooms[c.canvasID]
	c.hub.mu.RUnlock()
	if !ok {
		return
	}

	room.broadcastToAllExcept(WsMessage{
		Type: MsgTypeCursor,
		Payload: marshalPayload(CursorRelayPayload{
			CanvasID: c.canvasID,
			UserID:   c.userID,
			Position: p.Position,
		}),
	}, c.userID)
}

func (c *Client) handleAwareness(p AwarenessPayload) {
	c.hub.mu.RLock()
	room, ok := c.hub.rooms[c.canvasID]
	c.hub.mu.RUnlock()
	if !ok {
		return
	}

	room.broadcastToAllExcept(WsMessage{
		Type: MsgTypeAwareness,
		Payload: marshalPayload(AwarenessRelayPayload{
			CanvasID: c.canvasID,
			UserID:   c.userID,
			State:    p.State,
		}),
	}, c.userID)
}

func (c *Client) handleSave(p SavePayload) {
	if c.permission != "collaborate" {
		c.sendError("FORBIDDEN", "read-only access")
		return
	}

	c.hub.mu.RLock()
	room, ok := c.hub.rooms[c.canvasID]
	c.hub.mu.RUnlock()
	if !ok {
		return
	}

	// Persist the current CRDT state
	if c.saveHandler != nil {
		snapshot := room.Snapshot()
		if err := c.saveHandler(c.canvasID, c.userID, snapshot); err != nil {
			c.sendError("SAVE_FAILED", err.Error())
			return
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	c.sendJSON(WsMessage{
		Type: MsgTypeSaved,
		Payload: marshalPayload(SavedPayload{
			CanvasID:  c.canvasID,
			UpdatedAt: now,
		}),
	})

	room.broadcastToAll(WsMessage{
		Type: MsgTypeSaved,
		Payload: marshalPayload(SavedPayload{
			CanvasID:  c.canvasID,
			UpdatedAt: now,
		}),
	})
}

func (c *Client) sendJSON(v interface{}) {
	select {
	case c.send <- mustMarshal(v):
	default:
		// send buffer full, drop message
	}
}

func (c *Client) sendError(code, message string) {
	c.sendJSON(WsMessage{
		Type: MsgTypeError,
		Payload: marshalPayload(ErrorPayload{
			Code:    code,
			Message: message,
		}),
	})
}

func marshalPayload(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return json.RawMessage(b)
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
