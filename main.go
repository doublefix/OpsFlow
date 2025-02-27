package main

import (
	"log"
	"time"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/internal"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func CreateGinRouter(client internal.Client) *gin.Engine {
	r := gin.Default()
	r.Use(internal.AppContextMiddleware(client))

	r.GET("/test", GetPodInfo)
	r.GET("/ray", GetCreateRayClusterInfo)

	return r
}

func main() {
	client, err := internal.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	r := CreateGinRouter(client)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func GetPodInfo(c *gin.Context) {
	appCtx := internal.GetAppContext(c)
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
}

func GetCreateRayClusterInfo(c *gin.Context) {
	appCtx := internal.GetAppContext(c)
	aa, err := appCtx.Client().Ray().RayV1().RayClusters("chess-kuberay").Get(appCtx.Ctx(), "raycluster-kuberay", metav1.GetOptions{})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	print("=> ", aa.Name)
	// rayCluster := CreateRayCluster()
	// MarshalToJSON(rayCluster)

	c.JSON(200, gin.H{
		"message": "Ray Cluster is created",
	})
}

func CreateRayCluster() *rayv1.RayCluster {

	return &rayv1.RayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "raycluster-kuberay",
			Namespace: "chess-kuberay",
		},
		Spec: rayv1.RayClusterSpec{
			RayVersion:       "2.41.0",
			HeadGroupSpec:    CreateHeadGroupSpec("2.41.0", "rayproject/ray:2.41.0"),
			WorkerGroupSpecs: CreateWorkerGroupSpecs("rayproject/ray:2.41.0", 1),
		},
	}
}

func CreateHeadGroupSpec(rayVersion, rayImage string) rayv1.HeadGroupSpec {
	return rayv1.HeadGroupSpec{
		RayStartParams: map[string]string{},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "ray-head",
						Image: rayImage,
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								"cpu":    resource.MustParse("2"),
								"memory": resource.MustParse("4Gi"),
							},
							Requests: corev1.ResourceList{
								"cpu":    resource.MustParse("2"),
								"memory": resource.MustParse("4Gi"),
							},
						},
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 6379,
								Name:          "gcs-server",
							},
							{
								ContainerPort: 8265,
								Name:          "dashboard",
							},
							{
								ContainerPort: 10001,
								Name:          "client",
							},
						},
					},
				},
			},
		},
	}
}

func CreateWorkerGroupSpecs(rayImage string, numGroups int) []rayv1.WorkerGroupSpec {
	var workerGroupSpecs []rayv1.WorkerGroupSpec

	replicas := int32(1)
	minReplicas := int32(1)
	maxReplicas := int32(5)

	for i := range numGroups {
		workerGroupSpec := rayv1.WorkerGroupSpec{
			Replicas:       &replicas,
			MinReplicas:    &minReplicas,
			MaxReplicas:    &maxReplicas,
			GroupName:      "workergroup-" + fmt.Sprint(i+1),
			RayStartParams: map[string]string{},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: time.Now()},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "ray-worker",
							Image: rayImage,
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("1"),
									"memory": resource.MustParse("1Gi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("1"),
									"memory": resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
		}
		workerGroupSpecs = append(workerGroupSpecs, workerGroupSpec)
	}

	return workerGroupSpecs
}
