package router

import (
	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/app"
)

func RegisterRoutes(c *app.Container) *gin.Engine {

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	api := engine.Group("/api/v1")
	{
		api.GET("/node", c.NodeHandler.GetNodesHandle)

		api.POST("/pod", c.PodHandler.CreatePod)
		api.GET("/pod", c.PodHandler.GetPods)
		api.DELETE("/pod/:namespace/:name", c.PodHandler.DeletePod)

		api.POST("/deployments", c.DeploymentHandler.CreateDeployment)
		api.DELETE("/deployments/:namespace/:name", c.DeploymentHandler.DeleteDeployment)

		api.POST("/services", c.ServiceHandler.CreateService)
		api.DELETE("/services/:namespace/:name", c.ServiceHandler.DeleteService)

	}

	return engine
}
