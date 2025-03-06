package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/job"
	"github.com/modcoco/OpsFlow/pkg/model"
	"github.com/modcoco/OpsFlow/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateRayJobHandle(c *gin.Context) {
	var clusterConfig model.ClusterConfig
	if err := c.ShouldBindJSON(&clusterConfig); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	utils.MarshalToJSON(clusterConfig)

	appCtx := core.GetAppContext(c)
	existingJob, err := appCtx.Client().Ray().RayV1().RayJobs(clusterConfig.Namespace).Get(appCtx.Ctx(), clusterConfig.Job.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if existingJob.Name == clusterConfig.Job.Name {
		fmt.Println(existingJob.Name)
		c.JSON(400, gin.H{"message": "Cluster already exists"})
		return
	}

	rayJob := job.CreateRayJob(clusterConfig)
	utils.MarshalToJSON(rayJob)
	res, err := appCtx.Client().Ray().RayV1().RayJobs(clusterConfig.Namespace).Create(appCtx.Ctx(), rayJob, metav1.CreateOptions{})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(res.Name)

	c.JSON(200, gin.H{
		"message": fmt.Sprintf("Ray Cluster %s is created", res.Name),
	})
}
