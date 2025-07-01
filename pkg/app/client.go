package app

import (
	"fmt"

	rayclient "github.com/ray-project/kuberay/ray-operator/pkg/client/clientset/versioned"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client interface {
	Core() kubernetes.Interface
	Ray() rayclient.Interface
	// Volcano() versioned.Interface
	Istio() istioclient.Interface
	Dynamic() dynamic.Interface
	DynamicNRI() dynamic.NamespaceableResourceInterface
	Config() rest.Config
}

type clientImpl struct {
	core kubernetes.Interface
	ray  rayclient.Interface
	// volcano versioned.Interface
	istio   istioclient.Interface
	dynamic dynamic.Interface
	nri     dynamic.NamespaceableResourceInterface
	config  rest.Config
}

func (c *clientImpl) Core() kubernetes.Interface { return c.core }
func (c *clientImpl) Ray() rayclient.Interface   { return c.ray }

// func (c *clientImpl) Volcano() versioned.Interface { return c.volcano }
func (c *clientImpl) Istio() istioclient.Interface                       { return c.istio }
func (c *clientImpl) Dynamic() dynamic.Interface                         { return c.dynamic }
func (c *clientImpl) DynamicNRI() dynamic.NamespaceableResourceInterface { return c.nri }
func (c *clientImpl) Config() rest.Config                                { return c.config }

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

	// volcanoClient, err := versioned.NewForConfig(cfg)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create volcano client: %w", err)
	// }

	istioClient, err := istioclient.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create istio client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	crdClient := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "opsflow.io",
		Version:  "v1alpha1",
		Resource: "noderesourceinfos",
	})

	return &clientImpl{
		core: kubeClient,
		ray:  rayClient,
		// volcano: volcanoClient,
		istio:   istioClient,
		dynamic: dynamicClient,
		nri:     crdClient,
		config:  *cfg,
	}, nil
}
