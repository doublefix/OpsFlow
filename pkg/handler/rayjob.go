package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/core"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// curl http://localhost:8090/api/v1/node\?limit\=1|jq .
func GetNodesHandle(c *gin.Context) {
	nodeName := c.Query("name")
	labelSelector := c.Query("labelSelector")
	limitStr := c.Query("limit")
	continueToken := c.Query("continue")

	appCtx := core.GetAppContext(c)
	client := appCtx.Client().Core().CoreV1().Nodes()

	if nodeName != "" {
		node, err := client.Get(appCtx.Ctx(), nodeName, metav1.GetOptions{})
		if err != nil {
			handleK8sError(c, err)
			return
		}
		c.JSON(http.StatusOK, node)
		return
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if limitStr != "" {
		limit, err := strconv.ParseInt(limitStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
			return
		}
		listOptions.Limit = limit
	}

	if continueToken != "" {
		listOptions.Continue = continueToken
	}

	nodeList, err := client.List(appCtx.Ctx(), listOptions)
	if err != nil {
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusOK, nodeList)
}

// curl http://localhost:8090/api/v1/pod\?limit\=1\&namespace\=kube-system
func GetPodsHandle(c *gin.Context) {
	namespace := c.DefaultQuery("namespace", "default")
	podName := c.Query("name")
	labelSelector := c.Query("labelSelector")
	limitStr := c.Query("limit")
	continueToken := c.Query("continue")

	appCtx := core.GetAppContext(c)
	client := appCtx.Client().Core().CoreV1().Pods(namespace)

	if podName != "" {
		pod, err := client.Get(appCtx.Ctx(), podName, metav1.GetOptions{})
		if err != nil {
			handleK8sError(c, err)
			return
		}
		c.JSON(http.StatusOK, pod)
		return
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if limitStr != "" {
		limit, err := strconv.ParseInt(limitStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
			return
		}
		listOptions.Limit = limit
	}

	if continueToken != "" {
		listOptions.Continue = continueToken
	}

	podList, err := client.List(appCtx.Ctx(), listOptions)
	if err != nil {
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusOK, podList)
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
