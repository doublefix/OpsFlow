package core

import "github.com/gin-gonic/gin"

func AppContextMiddleware(client Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		appCtx := &appContextImpl{
			ctx:    c.Request.Context(),
			client: client,
		}
		c.Set("appCtx", appCtx)
		c.Next()
	}
}
