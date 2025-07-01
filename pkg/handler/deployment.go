package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/pkg/core"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
