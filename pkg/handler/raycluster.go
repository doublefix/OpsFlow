package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/job"
	"github.com/modcoco/OpsFlow/pkg/model"
	"github.com/modcoco/OpsFlow/pkg/utils"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetCreateRayClusterInfo(c *gin.Context) {
	var clusterConfig model.ClusterConfig
	if err := c.ShouldBindJSON(&clusterConfig); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	utils.MarshalToJSON(clusterConfig)

	appCtx := core.GetAppContext(c)
	existingCluster, err := appCtx.Client().Ray().RayV1().RayClusters(clusterConfig.Namespace).Get(appCtx.Ctx(), clusterConfig.ClusterName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if existingCluster.Name == clusterConfig.ClusterName {
		fmt.Println(existingCluster.Name)
		c.JSON(400, gin.H{"message": "Cluster already exists"})
		return
	}

	rayCluster := CreateRayCluster(clusterConfig)
	utils.MarshalToJSON(rayCluster)
	res, err := appCtx.Client().Ray().RayV1().RayClusters(clusterConfig.Namespace).Create(appCtx.Ctx(), rayCluster, metav1.CreateOptions{})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(res.Name)

	c.JSON(200, gin.H{
		"message": fmt.Sprintf("Ray Cluster %s is created", res.Name),
	})
}

func CreateRayCluster(config model.ClusterConfig) *rayv1.RayCluster {
	rayVersion := config.RayVersion
	if rayVersion == "" {
		rayVersion = "2.41.0"
	}
	rayImage := config.RayImage
	if rayImage == "" {
		rayImage = "rayproject/ray:" + rayVersion
	}

	headGroupSpec := job.CreateHeadGroupSpec(config.Machines, rayImage)
	workerGroupSpecs := job.CreateWorkerGroupSpecs(config.Machines, rayImage)

	return &rayv1.RayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              config.ClusterName,
			Namespace:         config.Namespace,
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: rayv1.RayClusterSpec{
			RayVersion:       rayVersion,
			HeadGroupSpec:    headGroupSpec,
			WorkerGroupSpecs: workerGroupSpecs,
		},
	}
}
