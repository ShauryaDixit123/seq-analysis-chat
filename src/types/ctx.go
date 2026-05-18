package types

import (
	"context"

	"github.com/gin-gonic/gin"
)

type Context struct {
	Ctx context.Context
	R   *gin.Engine
}
