package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/configmap"
	"github.com/modcoco/OpsFlow/pkg/context"
	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/job"
	"github.com/modcoco/OpsFlow/pkg/model"
	"github.com/modcoco/OpsFlow/pkg/svc"
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

	rayJobCtx := context.NewRayJobContext(appCtx.Client().Core(), appCtx.Client().Ray(), appCtx.Ctx())
	createRayJobInfo, err := job.CreateRayJob(clusterConfig, rayJobCtx)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
	}
	utils.MarshalToJSON(createRayJobInfo)

	response := model.RayJobResponse{
		Namespace: createRayJobInfo.Namespace,
		JobID:     createRayJobInfo.JobID,
	}
	c.JSON(200, response)
}

func RayJobInfoHandle(c *gin.Context) {
	namespace := c.Param("namespace")
	jobName := c.Param("name")

	appCtx := core.GetAppContext(c)
	existingJob, err := appCtx.Client().Ray().RayV1().RayJobs(namespace).Get(appCtx.Ctx(), jobName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(404, gin.H{"message": "Job not found"})
			return
		}
		c.JSON(500, gin.H{"message": "Internal server error", "error": err.Error()})
		return
	}

	labelSelector := fmt.Sprintf("model-unique-id=%s", jobName)

	svcList, err := appCtx.Client().Core().CoreV1().Services(namespace).List(appCtx.Ctx(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to list services", "error": err.Error()})
		return
	}

	configMapList, err := appCtx.Client().Core().CoreV1().ConfigMaps(namespace).List(appCtx.Ctx(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to list configmaps", "error": err.Error()})
		return
	}

	var svcNames []string
	for _, svc := range svcList.Items {
		svcNames = append(svcNames, svc.Name)
	}

	var configMapNames []string
	for _, cm := range configMapList.Items {
		configMapNames = append(configMapNames, cm.Name)
	}

	response := gin.H{
		"message":        "Job found",
		"jobName":        jobName,
		"namespace":      namespace,
		"jobStatus":      existingJob.Status.JobStatus,
		"startTime":      existingJob.Status.StartTime,
		"failed":         existingJob.Status.Failed,
		"rayClusterName": existingJob.Status.RayClusterName,
	}
	if len(svcNames) > 0 {
		response["services"] = svcNames
	}
	if len(configMapNames) > 0 {
		response["configMaps"] = configMapNames
	}
	c.JSON(200, response)
}

func RemoveRayJobHandle(c *gin.Context) {
	namespace := c.Param("namespace")
	jobName := c.Param("name")

	appCtx := core.GetAppContext(c)

	existingJob, err := appCtx.Client().Ray().RayV1().RayJobs(namespace).Get(appCtx.Ctx(), jobName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(404, gin.H{"message": "Job not found"})
			return
		}
		c.JSON(500, gin.H{"message": "Internal server error", "error": err.Error()})
		return
	}

	err = appCtx.Client().Ray().RayV1().RayJobs(namespace).Delete(appCtx.Ctx(), jobName, metav1.DeleteOptions{})
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to delete job", "error": err.Error()})
		return
	}

	labelSelector := fmt.Sprintf("model-unique-id=%s", jobName)
	_ = svc.DeleteServicesByLabel(appCtx, namespace, labelSelector)
	_ = configmap.DeleteConfigMapsByLabel(appCtx, namespace, labelSelector)

	c.JSON(200, gin.H{
		"message": "Job and associated resources deleted successfully",
		"jobName": existingJob.Name,
	})
}
