package main

import (
	"log"

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
	hub := ws.NewHub(engine, rm)
	go hub.Run()

	r := gin.Default()
	r.Use(middleware.CORS())

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", handler.Register(db))
			auth.POST("/login", handler.Login(db, cfg.JWTSecret))
		}

		protected := api.Group("")
		protected.Use(middleware.JWTAuth(cfg.JWTSecret))
		{
			protected.GET("/cards", handler.GetCardDescriptions(db))

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
