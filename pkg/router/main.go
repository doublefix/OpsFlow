package router

import (
	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/app"
)

func RegisterRoutes(c *app.Container) *gin.Engine {
	engine := gin.New()

	engine.Use(
		gin.Recovery(),
		gin.Logger(),
	)

	api := engine.Group("/api/v1")
	{
		api.GET("/nodes", c.NodeHandler.GetNodesHandle)

		pod := api.Group("/pods")
		{
			pod.GET("", c.PodHandler.GetPods)
			pod.POST("", c.PodHandler.CreatePod)
			pod.DELETE("/:namespace/:name", c.PodHandler.DeletePod)
		}

		deployment := api.Group("/deployments")
		{
			deployment.POST("", c.DeploymentHandler.CreateDeployment)
			deployment.DELETE("/:namespace/:name", c.DeploymentHandler.DeleteDeployment)
		}

		service := api.Group("/services")
		{
			service.POST("", c.ServiceHandler.CreateService)
			service.DELETE("/:namespace/:name", c.ServiceHandler.DeleteService)
		}
	}

	return engine
}
