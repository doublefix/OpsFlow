package app

import (
	"github.com/modcoco/OpsFlow/pkg/handler"
)

type Container struct {
	NodeHandler         *handler.NodeHandler
	PodHandler          *handler.PodHandler
	DeploymentHandler   *handler.DeploymentHandler
	ServiceHandler      *handler.ServiceHandler
	ModelPiplineHandler *handler.ModelPiplineHandler
}

func NewContainer(c Client) *Container {
	client := c.Core()
	ctrlClient := c.Runtime()
	nodeClient := client.CoreV1().Nodes()

	return &Container{
		NodeHandler:         handler.NewNodeHandler(nodeClient),
		PodHandler:          handler.NewPodHandler(client),
		DeploymentHandler:   handler.NewDeploymentHandler(client),
		ServiceHandler:      handler.NewServiceHandler(client),
		ModelPiplineHandler: handler.NewModelPiplineHandler(ctrlClient),
	}
}
