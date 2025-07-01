package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodHandler struct {
	client kubernetes.Interface
}

func NewPodHandler(client kubernetes.Interface) *PodHandler {
	return &PodHandler{client: client}
}

func (h *PodHandler) GetPods(c *gin.Context) {
	namespace := c.DefaultQuery("namespace", "default")
	podName := c.Query("name")
	labelSelector := c.Query("labelSelector")
	limitStr := c.Query("limit")
	continueToken := c.Query("continue")

	client := h.client.CoreV1().Pods(namespace)

	if podName != "" {
		pod, err := client.Get(c.Request.Context(), podName, metav1.GetOptions{})
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

	podList, err := client.List(c.Request.Context(), listOptions)
	if err != nil {
		handleK8sError(c, err)
		return
	}

	c.JSON(http.StatusOK, podList)
}

func (h *PodHandler) CreatePod(c *gin.Context) {
	var pod corev1.Pod
	if err := c.ShouldBindJSON(&pod); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if pod.Namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pod.metadata.namespace is required"})
		return
	}

	client := h.client.CoreV1().Pods(pod.Namespace)

	if _, err := client.Get(c.Request.Context(), pod.Name, metav1.GetOptions{}); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("pod %q already exists", pod.Name),
		})
		return
	} else if !k8serrors.IsNotFound(err) {
		handleK8sError(c, err)
		return
	}

	created, err := client.Create(c.Request.Context(), &pod, metav1.CreateOptions{})
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

func (h *PodHandler) DeletePod(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if namespace == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "namespace and name are required",
		})
		return
	}

	client := h.client.CoreV1().Pods(namespace)

	_, err := client.Get(c.Request.Context(), name, metav1.GetOptions{})
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

	err = client.Delete(c.Request.Context(), name, metav1.DeleteOptions{})
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
