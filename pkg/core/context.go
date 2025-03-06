package core

import (
	"context"

	"github.com/gin-gonic/gin"
)

type AppContext interface {
	Ctx() context.Context
	Client() Client
}

type appContextImpl struct {
	ctx    context.Context
	client Client
}

func (a *appContextImpl) Ctx() context.Context { return a.ctx }
func (a *appContextImpl) Client() Client       { return a.client }

func GetAppContext(c *gin.Context) AppContext {
	return c.MustGet("appCtx").(AppContext)
}
