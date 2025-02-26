package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	rayclient "github.com/ray-project/kuberay/ray-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
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

func newClient() (Client, error) {
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

type AppContext interface {
	Ctx() context.Context
	Client() Client
	OutputDir() string
}

type appContextImpl struct {
	ctx       context.Context
	client    Client
	outputDir string
}

func (a *appContextImpl) Ctx() context.Context { return a.ctx }
func (a *appContextImpl) Client() Client       { return a.client }
func (a *appContextImpl) OutputDir() string    { return a.outputDir }

func AppContextMiddleware(client Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		appCtx := &appContextImpl{
			ctx:       c.Request.Context(),
			client:    client,
			outputDir: "/tmp/output",
		}
		c.Set("appCtx", appCtx)
		c.Next()
	}
}

func getAppContext(c *gin.Context) AppContext {
	return c.MustGet("appCtx").(AppContext)
}

func CreateGinRouter(client Client) *gin.Engine {
	r := gin.Default()
	r.Use(AppContextMiddleware(client))

	r.GET("/test", func(c *gin.Context) {
		appCtx := getAppContext(c)
		pods, err := appCtx.Client().Core().CoreV1().Pods("default").List(
			appCtx.Ctx(),
			metav1.ListOptions{},
		)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": "Kubernetes client is working",
			"pods":    pods.Items,
		})
	})

	return r
}

func main() {
	client, err := newClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	r := CreateGinRouter(client)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
