package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackvidyu/witchhunt/server/internal/config"
	"github.com/jackvidyu/witchhunt/server/internal/game"
	"github.com/jackvidyu/witchhunt/server/internal/handler"
	"github.com/jackvidyu/witchhunt/server/internal/middleware"
	"github.com/jackvidyu/witchhunt/server/internal/model"
	"github.com/jackvidyu/witchhunt/server/internal/room"
	"github.com/jackvidyu/witchhunt/server/internal/ws"
)

func main() {
	cfg := config.Load()

	db, err := model.InitDB(cfg.DSN())
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	rm := room.NewManager()
	engine := game.NewEngine()
	hub := ws.NewHub(engine, rm, db)
	go hub.Run()

	r := gin.Default()
	r.Use(middleware.CORS())

	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			sqlDB, err := db.DB()
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "degraded",
					"error":  "db handle unavailable",
				})
				return
			}

			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := sqlDB.PingContext(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "degraded",
					"error":  "database unreachable",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
			})
		})

		auth := api.Group("/auth")
		{
			auth.POST("/register", handler.Register(db))
			auth.POST("/login", handler.Login(db, cfg.JWTSecret))
		}

		protected := api.Group("")
		protected.Use(middleware.JWTAuth(cfg.JWTSecret))
		{
			protected.GET("/cards", handler.GetCardDescriptions(db))
			protected.GET("/user/stats", handler.GetUserStats(db))
			protected.GET("/user/history", handler.GetUserHistory(db))

			roomGroup := protected.Group("/room")
			{
				roomGroup.POST("/create", handler.CreateRoom(rm))
				roomGroup.POST("/join", handler.JoinRoom(rm))
			}
		}

		api.GET("/ws", handler.ServeWS(hub, rm, cfg.JWTSecret))
	}

	log.Printf("server starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
