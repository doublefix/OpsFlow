package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ServiceHandler struct {
	client kubernetes.Interface
}

func NewServiceHandler(client kubernetes.Interface) *ServiceHandler {
	return &ServiceHandler{client: client}
}

func (h *ServiceHandler) CreateService(c *gin.Context) {
	var svc corev1.Service
	if err := c.ShouldBindJSON(&svc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if svc.Namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service.metadata.namespace is required"})
		return
	}

	client := h.client.CoreV1().Services(svc.Namespace)

	if _, err := client.Get(c.Request.Context(), svc.Name, metav1.GetOptions{}); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("service %q already exists", svc.Name),
		})
		return
	} else if !k8serrors.IsNotFound(err) {
		handleK8sError(c, err)
		return
	}

	created, err := client.Create(c.Request.Context(), &svc, metav1.CreateOptions{})
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

func (h *ServiceHandler) DeleteService(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if namespace == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "namespace and name are required",
		})
		return
	}

	client := h.client.CoreV1().Services(namespace)

	err := client.Delete(c.Request.Context(), name, metav1.DeleteOptions{})
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
