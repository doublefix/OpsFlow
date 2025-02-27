package internal

import (
	"fmt"

	rayclient "github.com/ray-project/kuberay/ray-operator/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client interface {
	Core() kubernetes.Interface
	Ray() rayclient.Interface
	Dynamic() dynamic.Interface
	Config() rest.Config
}

type clientImpl struct {
	core    kubernetes.Interface
	ray     rayclient.Interface
	dynamic dynamic.Interface
	config  rest.Config
}

func (c *clientImpl) Core() kubernetes.Interface { return c.core }
func (c *clientImpl) Ray() rayclient.Interface   { return c.ray }
func (c *clientImpl) Dynamic() dynamic.Interface { return c.dynamic }
func (c *clientImpl) Config() rest.Config        { return c.config }

func NewClient() (Client, error) {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	rayClient, err := rayclient.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ray client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &clientImpl{
		core:    kubeClient,
		ray:     rayClient,
		dynamic: dynamicClient,
		config:  *cfg,
	}, nil
}
