package job

import (
	"context"
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
	ResultChan chan<- string
}

type RayJobWatcher struct {
	Clientset  rayclient.Interface
	Namespace  string
	JobName    string
	Timeout    time.Duration
	ResultChan chan<- string
}

func NewRayJobWatcher(config RayJobWatcherConfig) *RayJobWatcher {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Minute
	}

	return &RayJobWatcher{
		Clientset:  config.Clientset,
		Namespace:  config.Namespace,
		JobName:    config.JobName,
		Timeout:    config.Timeout,
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
			log.Println("ticker to get rayjob: ", rjw.Namespace, rjw.JobName)
			rayJob, err := rjw.Clientset.RayV1().RayJobs(rjw.Namespace).Get(ctx, rjw.JobName, metav1.GetOptions{})
			if err != nil {
				log.Printf("Failed to get RayJob: %v", err)
				continue
			}

			if rayJob.Status.RayClusterName != "" {
				log.Println("get raycluster name", rayJob.Status.RayClusterName)
				rjw.ResultChan <- rayJob.Status.RayClusterName
				close(rjw.ResultChan)
				return
			}

		case <-ctx.Done():
			log.Println("Down due to:", ctx.Err())
			rjw.ResultChan <- "timeout waiting for RayClusterName"
			close(rjw.ResultChan)
			return
		}
	}
}
