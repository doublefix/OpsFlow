package job

import (
	"context"
	"time"

	rayclient "github.com/ray-project/kuberay/ray-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RayJobWatcherConfig struct {
	Clientset  rayclient.Interface
	Namespace  string
	JobName    string
	Timeout    time.Duration
	Context    context.Context
	ResultChan chan<- string
}

type RayJobWatcher struct {
	Clientset  rayclient.Interface
	Namespace  string
	JobName    string
	Timeout    time.Duration
	Context    context.Context
	ResultChan chan<- string
}

func NewRayJobWatcher(config RayJobWatcherConfig) *RayJobWatcher {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Minute
	}

	if config.Context == nil {
		config.Context = context.Background()
	}

	return &RayJobWatcher{
		Clientset:  config.Clientset,
		Namespace:  config.Namespace,
		JobName:    config.JobName,
		Timeout:    config.Timeout,
		Context:    config.Context,
		ResultChan: config.ResultChan,
	}
}

func (rjw *RayJobWatcher) WaitForRayClusterName() {
	endTime := time.Now().Add(rjw.Timeout)

	for time.Now().Before(endTime) {
		rayJob, err := rjw.Clientset.RayV1().RayJobs(rjw.Namespace).Get(rjw.Context, rjw.JobName, metav1.GetOptions{})
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		if rayJob.Status.RayClusterName != "" {
			rjw.ResultChan <- rayJob.Status.RayClusterName
			return
		}

		time.Sleep(10 * time.Second)
	}

	rjw.ResultChan <- "timeout waiting for RayClusterName"
}
