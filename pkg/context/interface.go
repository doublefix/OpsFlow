package context

import (
	"context"

	rayclient "github.com/ray-project/kuberay/ray-operator/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

type RayJobContext interface {
	Core() kubernetes.Interface
	Ray() rayclient.Interface
	Ctx() context.Context
}

type rayJobContext struct {
	coreClient kubernetes.Interface
	rayClient  rayclient.Interface
	ctx        context.Context
}

func (r *rayJobContext) Core() kubernetes.Interface {
	return r.coreClient
}

func (r *rayJobContext) Ray() rayclient.Interface {
	return r.rayClient
}

func (r *rayJobContext) Ctx() context.Context {
	return r.ctx
}

func NewRayJobContext(core kubernetes.Interface, ray rayclient.Interface, ctx context.Context) RayJobContext {
	return &rayJobContext{
		coreClient: core,
		rayClient:  ray,
		ctx:        ctx,
	}
}
