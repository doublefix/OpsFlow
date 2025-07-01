package router

import (
	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/app"
	"github.com/modcoco/OpsFlow/pkg/handler"
)

func RegisterRoutes(c *app.Container) *gin.Engine {

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	api := engine.Group("/api/v1")
	{
		api.GET("/node", c.NodeHandler.GetNodesHandle)
		api.POST("/pod", handler.CreatePodHandle)
		api.GET("/pod", handler.GetPodsHandle)
		api.POST("/deployments", handler.CreateDeploymentHandle)
		api.DELETE("/deployments/:namespace/:name", handler.DeleteDeploymentHandle)
		api.DELETE("/pod/:namespace/:name", handler.DeletePodHandle)
		api.POST("/services", handler.CreateServiceHandle)
		api.DELETE("/services/:namespace/:name", handler.DeleteServiceHandle)

	}

	return engine
}
