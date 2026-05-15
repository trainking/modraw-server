package ws

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/trainking/modraw-server/internal/config"
	jwtpkg "github.com/trainking/modraw-server/pkg/jwt"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type AccessChecker func(canvasID, userID, shareToken string) (permission string, err error)

type Handler struct {
	hub           *Hub
	accessChecker AccessChecker
	saveHandler   SaveFunc
	cfg           *config.Config
}

func NewHandler(hub *Hub, cfg *config.Config, accessChecker AccessChecker, saveHandler SaveFunc) *Handler {
	return &Handler{hub: hub, cfg: cfg, accessChecker: accessChecker, saveHandler: saveHandler}
}

func (h *Handler) Upgrade(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"ok":      false,
			"error":   "MISSING_TOKEN",
			"message": "token query parameter required",
		})
		return
	}

	claims, err := jwtpkg.ValidateToken(token, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"ok":      false,
			"error":   "INVALID_TOKEN",
			"message": "invalid or expired token",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	client := NewClient(h.hub, conn, claims.UserID, claims.Email, claims.Nickname, "")
	client.SetSaveHandler(h.saveHandler)
	checker := h.accessChecker

	go client.WritePump()
	go client.ReadPump(func(canvasID, userID, shareToken string) (string, error) {
		return checker(canvasID, userID, shareToken)
	})
}
