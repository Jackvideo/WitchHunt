package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackvidyu/witchhunt/server/internal/middleware"
	"github.com/jackvidyu/witchhunt/server/internal/room"
	"github.com/jackvidyu/witchhunt/server/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func ServeWS(hub *ws.Hub, rm *room.Manager, secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		roomCode := c.Query("room")

		if tokenStr == "" || roomCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token and room required"})
			return
		}

		claims, err := middleware.ParseToken(tokenStr, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		if _, ok := rm.Get(roomCode); !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}

		userID := uint(claims["user_id"].(float64))
		username := claims["username"].(string)

		client := &ws.Client{
			Hub:      hub,
			Conn:     conn,
			Send:     make(chan []byte, 256),
			UserID:   userID,
			Username: username,
			RoomCode: roomCode,
		}

		hub.Register <- client
		go client.WritePump()
		go client.ReadPump()
	}
}
