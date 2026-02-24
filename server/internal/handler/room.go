package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackvidyu/witchhunt/server/internal/room"
)

type createRoomReq struct {
	MaxPlayers int `json:"max_players" binding:"required,min=2,max=20"`
}

type joinRoomReq struct {
	Code string `json:"code" binding:"required,len=6"`
}

func CreateRoom(rm *room.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createRoomReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		uid := uint(c.GetFloat64("user_id"))
		username, _ := c.Get("username")

		r := rm.Create(uid, username.(string), req.MaxPlayers)
		c.JSON(http.StatusCreated, r)
	}
}

func JoinRoom(rm *room.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req joinRoomReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		uid := uint(c.GetFloat64("user_id"))
		username, _ := c.Get("username")

		r, err := rm.Join(req.Code, uid, username.(string))
		if err != nil {
			status := http.StatusNotFound
			if err == room.ErrRoomFull {
				status = http.StatusConflict
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, r)
	}
}
