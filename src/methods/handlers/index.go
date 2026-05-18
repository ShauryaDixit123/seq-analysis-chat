package handlers

import (
	"atgc/src/methods"
	"atgc/src/types"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Ctx types.Context
}

func (h *Handler) DynamicProgrammingMatchHandle(ctx *gin.Context) {
	log.Default().Printf("Received request for Dynamic Programming Match")
	var body types.DynamicPrgrammingRequestBody
	m := methods.NewDyanmicProgrammingMatch(
		"Dynamic Programming method",
		time.Now().Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
	)
	err := ctx.BindJSON(&body)
	if err != nil {
		log.Default().Printf("Error binding JSON: %v", err)
		ctx.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}
	m.DyanmicProgrammingMatch(h.Ctx, body)
}
