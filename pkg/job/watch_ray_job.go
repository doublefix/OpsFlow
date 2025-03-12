package job

import (
	"context"
	"fmt"
	"log"
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
	ctx, cancel := context.WithTimeout(context.Background(), rjw.Timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Println("ticker.C")
			rayJob, err := rjw.Clientset.RayV1().RayJobs(rjw.Namespace).Get(ctx, rjw.JobName, metav1.GetOptions{})
			if err != nil {
				log.Printf("Failed to get RayJob: %v", err)
				continue
			}

			fmt.Println("Get raycluster name", rayJob.Status.RayClusterName)

			if rayJob.Status.RayClusterName != "" {
				fmt.Println("Get raycluster name", rayJob.Status.RayClusterName)
				rjw.ResultChan <- rayJob.Status.RayClusterName
				close(rjw.ResultChan)
				return
			}

		case <-ctx.Done():
			fmt.Println("Down due to:", ctx.Err()) // Log the error to identify the cause
			rjw.ResultChan <- "timeout waiting for RayClusterName"
			close(rjw.ResultChan)
			return
		}
	}
}
