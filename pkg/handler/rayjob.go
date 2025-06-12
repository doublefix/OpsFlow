package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/configmap"
	"github.com/modcoco/OpsFlow/pkg/context"
	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/job"
	"github.com/modcoco/OpsFlow/pkg/model"
	"github.com/modcoco/OpsFlow/pkg/svc"
	"github.com/modcoco/OpsFlow/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	if err != nil && !k8serrors.IsNotFound(err) {
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

func CreateServiceHandle(c *gin.Context) {
	var svc corev1.Service
	if err := c.ShouldBindJSON(&svc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if svc.Namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service.metadata.namespace is required"})
		return
	}

	appCtx := core.GetAppContext(c)
	client := appCtx.Client().Core().CoreV1().Services(svc.Namespace)

	if _, err := client.Get(appCtx.Ctx(), svc.Name, metav1.GetOptions{}); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("service %q already exists", svc.Name),
		})
		return
	} else if !k8serrors.IsNotFound(err) {
		handleK8sError(c, err)
		return
	}

	created, err := client.Create(appCtx.Ctx(), &svc, metav1.CreateOptions{})
	if err != nil {
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name":      created.Name,
		"namespace": created.Namespace,
		"uid":       created.UID,
	})
}

func DeleteServiceHandle(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if namespace == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "namespace and name are required",
		})
		return
	}

	appCtx := core.GetAppContext(c)
	client := appCtx.Client().Core().CoreV1().Services(namespace)

	err := client.Delete(appCtx.Ctx(), name, metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "service deleted successfully",
		"name":      name,
		"namespace": namespace,
	})
}

func CreateDeploymentHandle(c *gin.Context) {
	var deploy appsv1.Deployment
	if err := c.ShouldBindJSON(&deploy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if deploy.Namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deployment.metadata.namespace is required"})
		return
	}

	appCtx := core.GetAppContext(c)
	client := appCtx.Client().Core().AppsV1().Deployments(deploy.Namespace)

	if _, err := client.Get(appCtx.Ctx(), deploy.Name, metav1.GetOptions{}); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("deployment %q already exists", deploy.Name),
		})
		return
	} else if !k8serrors.IsNotFound(err) {
		handleK8sError(c, err)
		return
	}

	created, err := client.Create(appCtx.Ctx(), &deploy, metav1.CreateOptions{})
	if err != nil {
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name":      created.Name,
		"namespace": created.Namespace,
		"uid":       created.UID,
	})
}

func CreatePodHandle(c *gin.Context) {
	var pod corev1.Pod
	if err := c.ShouldBindJSON(&pod); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if pod.Namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pod.metadata.namespace is required"})
		return
	}

	appCtx := core.GetAppContext(c)
	client := appCtx.Client().Core().CoreV1().Pods(pod.Namespace)

	if _, err := client.Get(appCtx.Ctx(), pod.Name, metav1.GetOptions{}); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("pod %q already exists", pod.Name),
		})
		return
	} else if !k8serrors.IsNotFound(err) {
		handleK8sError(c, err)
		return
	}

	created, err := client.Create(appCtx.Ctx(), &pod, metav1.CreateOptions{})
	if err != nil {
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name":      created.Name,
		"namespace": created.Namespace,
		"uid":       created.UID,
	})
}

func DeletePodHandle(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if namespace == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "namespace and name are required",
		})
		return
	}

	appCtx := core.GetAppContext(c)
	client := appCtx.Client().Core().CoreV1().Pods(namespace)

	_, err := client.Get(appCtx.Ctx(), name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "pod not found",
			})
			return
		}
		handleK8sError(c, err)
		return
	}

	err = client.Delete(appCtx.Ctx(), name, metav1.DeleteOptions{})
	if err != nil {
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "pod deleted successfully",
		"name":      name,
		"namespace": namespace,
	})
}

func handleK8sError(c *gin.Context, err error) {
	if statusErr, ok := err.(*k8serrors.StatusError); ok {
		c.JSON(int(statusErr.ErrStatus.Code), gin.H{
			"reason":  statusErr.ErrStatus.Reason,
			"message": statusErr.ErrStatus.Message,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
}

func DeleteDeploymentHandle(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if namespace == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "namespace and name are required",
		})
		return
	}

	appCtx := core.GetAppContext(c)
	client := appCtx.Client().Core().AppsV1().Deployments(namespace)

	_, err := client.Get(appCtx.Ctx(), name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "deployment not found",
			})
			return
		}
		handleK8sError(c, err)
		return
	}

	err = client.Delete(appCtx.Ctx(), name, metav1.DeleteOptions{})
	if err != nil {
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "deployment deleted successfully",
		"name":      name,
		"namespace": namespace,
	})
}

func RayJobInfoHandle(c *gin.Context) {
	namespace := c.Param("namespace")
	jobName := c.Param("name")

	appCtx := core.GetAppContext(c)
	existingJob, err := appCtx.Client().Ray().RayV1().RayJobs(namespace).Get(appCtx.Ctx(), jobName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
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
		if k8serrors.IsNotFound(err) {
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
