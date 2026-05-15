package ws

import (
	"encoding/json"
	"sync"

	"github.com/trainking/modraw-server/internal/crdt"
)

type StateLoader func(canvasID string) (json.RawMessage, error)

type Hub struct {
	rooms       map[string]*Room
	register    chan *Client
	unregister  chan *Client
	stateLoader StateLoader
	mu          sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
	}
}

func (h *Hub) SetStateLoader(loader StateLoader) {
	h.stateLoader = loader
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		}
	}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	room, ok := h.rooms[client.canvasID]
	if !ok {
		room = NewRoom(client.canvasID)
		// Load initial CRDT state from DB
		if h.stateLoader != nil {
			data, err := h.stateLoader(client.canvasID)
			if err == nil {
				state, err := crdt.ParseState(data)
				if err == nil {
					room.crdtState = state
				}
			}
		}
		if room.crdtState == nil {
			room.crdtState, _ = crdt.ParseState(json.RawMessage("{}"))
		}
		h.rooms[client.canvasID] = room
	}
	h.mu.Unlock()

	room.addClient(client)

	clientInfos := room.getClientInfos()
	client.sendJSON(WsMessage{
		Type: MsgTypeJoined,
		Payload: marshalPayload(JoinedPayload{
			CanvasID: client.canvasID,
			Clients:  clientInfos,
		}),
	})
}

func (h *Hub) removeClient(client *Client) {
	h.mu.RLock()
	room, ok := h.rooms[client.canvasID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	room.removeClient(client.userID)

	room.broadcastToAll(WsMessage{
		Type: MsgTypeLeft,
		Payload: marshalPayload(LeftPayload{
			CanvasID: client.canvasID,
			UserID:   client.userID,
		}),
	})

	if room.isEmpty() {
		h.mu.Lock()
		delete(h.rooms, client.canvasID)
		h.mu.Unlock()
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

type Room struct {
	id        string
	clients   map[string]*Client
	crdtState *crdt.CanvasState
	mu        sync.RWMutex
}

func NewRoom(id string) *Room {
	return &Room{
		id:      id,
		clients: make(map[string]*Client),
	}
}

func (r *Room) addClient(client *Client) {
	r.mu.Lock()
	r.clients[client.userID] = client
	r.mu.Unlock()
}

func (r *Room) removeClient(userID string) {
	r.mu.Lock()
	delete(r.clients, userID)
	r.mu.Unlock()
}

func (r *Room) isEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients) == 0
}

func (r *Room) getClientInfos() []ClientInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ClientInfo, 0, len(r.clients))
	for _, c := range r.clients {
		infos = append(infos, ClientInfo{
			UserID:    c.userID,
			Nickname:  c.nickname,
			AvatarURL: c.avatarURL,
		})
	}
	return infos
}

func (r *Room) broadcastToAll(msg WsMessage) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		c.sendJSON(msg)
	}
}

func (r *Room) broadcastToAllExcept(msg WsMessage, excludeUserID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		if c.userID != excludeUserID {
			c.sendJSON(msg)
		}
	}
}

func (r *Room) broadcastOpExcept(msg WsMessage, excludeUserID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		if c.userID != excludeUserID && c.permission == "collaborate" {
			c.sendJSON(msg)
		}
	}
}

func (r *Room) getClientPermission(userID string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[userID]
	if !ok {
		return "", false
	}
	return c.permission, true
}

func (r *Room) ApplyOp(op crdt.Operation) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.crdtState.Apply(op)
}

func (r *Room) Snapshot() json.RawMessage {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.crdtState.Serialize()
}
