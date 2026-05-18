package main

import (
	"atgc/src/methods/handlers"
	"atgc/src/types"
	"context"
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

func main() {
	r := gin.Default()

	tmpl := template.Must(template.New("").ParseFS(templateFS, "templates/*"))
	r.SetHTMLTemplate(tmpl)

	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	r.StaticFS("/static", http.FS(staticRoot))
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
