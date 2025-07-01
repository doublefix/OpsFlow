package app

import (
	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/handler"
)

type Container struct {
	NodeHandler *handler.NodeHandler
}

func NewContainer(c core.Client) *Container {
	nodeClient := c.Core().CoreV1().Nodes()

	return &Container{
		NodeHandler: handler.NewNodeHandler(nodeClient),
	}
}
