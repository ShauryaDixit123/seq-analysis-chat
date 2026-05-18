package main

import (
	"atgc/src/methods/handlers"
	"atgc/src/types"
	"context"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.Static("/uploads", "./uploads")

	ctx := types.Context{
		Ctx: context.Background(),
		R:   r,
	}
	h := handlers.Handler{Ctx: ctx}

	health(r)
	r.GET("/", h.ServeChat)
	r.GET("/api/chat", h.ListChatMessages)
	r.POST("/api/chat", h.PostChatMessage)
	r.POST("/api/chat/session", h.InitChatSession)
	r.GET("/api/chat/session/:session_id", h.GetChatMessages)
	r.POST("/dynamic", h.DynamicProgrammingMatchHandle)
	r.Run()
}

func health(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "OK"})
	})
}
