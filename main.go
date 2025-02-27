package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modcoco/OpsFlow/internal"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/utils/ptr"
)

func CreateGinRouter(client internal.Client) *gin.Engine {
	r := gin.Default()
	r.Use(internal.AppContextMiddleware(client))

	r.GET("/api/v1/test", GetPodInfo)
	r.POST("/api/v1/raycluster", GetCreateRayClusterInfo)

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
	var clusterConfig ClusterConfig
	if err := c.ShouldBindJSON(&clusterConfig); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	internal.MarshalToJSON(clusterConfig)

	appCtx := internal.GetAppContext(c)
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
	// internal.MarshalToJSON(rayCluster)
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

func CreateRayCluster(config ClusterConfig) *rayv1.RayCluster {
	rayVersion := config.RayVersion
	if rayVersion == "" {
		rayVersion = "2.41.0"
	}
	rayImage := config.RayImage
	if rayImage == "" {
		rayImage = "rayproject/ray:" + rayVersion
	}

	headGroupSpec := CreateHeadGroupSpec(config.Machines, rayImage)
	workerGroupSpecs := CreateWorkerGroupSpecs(config.Machines, rayImage)

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

func CreateHeadGroupSpec(machines []MachineConfig, rayImage string) rayv1.HeadGroupSpec {
	var headMachine *MachineConfig
	for _, machine := range machines {
		if machine.IsHeadNode {
			headMachine = &machine
			break
		}
	}

	if headMachine == nil {
		headMachine = &machines[0]
	}

	resourceList := corev1.ResourceList{
		"cpu":    resource.MustParse(headMachine.CPU),
		"memory": resource.MustParse(headMachine.Memory),
	}
	var runtimeClassName *string
	for key, value := range headMachine.CustomResources {
		resourceList[corev1.ResourceName(key)] = resource.MustParse(value)
		if strings.HasPrefix(key, "nvidia.com") {
			runtimeClassName = ptr.To("nvidia")
		}
	}

	return rayv1.HeadGroupSpec{
		RayStartParams: map[string]string{},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  machines[0].Name,
						Image: rayImage,
						Resources: corev1.ResourceRequirements{
							Limits:   resourceList,
							Requests: resourceList,
						},
						Ports: ConvertPorts(headMachine.Ports),
					},
				},
				RuntimeClassName: runtimeClassName,
			},
		},
	}
}

func CreateWorkerGroupSpecs(machines []MachineConfig, rayImage string) []rayv1.WorkerGroupSpec {
	var workerGroupSpecs []rayv1.WorkerGroupSpec
	defaultReplicas := int32(1)
	defaultMinReplicas := int32(1)
	defaultMaxReplicas := int32(1)
	defaultGroupName := "workergroup"

	for _, machine := range machines {
		if !machine.IsHeadNode {
			// 创建资源列表，包括 CPU、内存和自定义资源
			resourceList := corev1.ResourceList{
				"cpu":    resource.MustParse(machine.CPU),
				"memory": resource.MustParse(machine.Memory),
			}
			var runtimeClassName *string
			for key, value := range machine.CustomResources {
				resourceList[corev1.ResourceName(key)] = resource.MustParse(value)
				if strings.HasPrefix(key, "nvidia.com") {
					runtimeClassName = ptr.To("nvidia")
				}
			}

			// 根据 MachineType 设置 Replicas、MinReplicas 和 MaxReplicas
			var replicas, minReplicas, maxReplicas *int32
			var groupName string

			if machine.MachineType == MachineTypeGroup {
				// 如果是 group 类型，使用用户指定的值
				replicas = machine.Replicas
				minReplicas = machine.MinReplicas
				maxReplicas = machine.MaxReplicas
				groupName = machine.GroupName
			} else {
				// 如果是 single 类型，使用默认值
				replicas = &defaultReplicas
				minReplicas = &defaultMinReplicas
				maxReplicas = &defaultMaxReplicas
				groupName = defaultGroupName
			}

			// 创建 WorkerGroupSpec
			workerGroupSpec := rayv1.WorkerGroupSpec{
				Replicas:       replicas,
				MinReplicas:    minReplicas,
				MaxReplicas:    maxReplicas,
				GroupName:      groupName,
				RayStartParams: map[string]string{},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  machine.Name,
								Image: rayImage,
								Resources: corev1.ResourceRequirements{
									Limits:   resourceList,
									Requests: resourceList,
								},
								Ports: ConvertPorts(machine.Ports),
							},
						},
						RuntimeClassName: runtimeClassName,
					},
				},
			}
			workerGroupSpecs = append(workerGroupSpecs, workerGroupSpec)
		}
	}

	return workerGroupSpecs
}

// ConvertPorts 将端口配置转换为 ContainerPort 列表
func ConvertPorts(ports []PortConfig) []corev1.ContainerPort {
	var containerPorts []corev1.ContainerPort
	for _, port := range ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          port.Name,
			ContainerPort: port.ContainerPort,
		})
	}
	return containerPorts
}

type ClusterType string

const (
	ClusterTypeRay     ClusterType = "ray"
	ClusterTypeVolcano ClusterType = "volcano"
)

type ClusterConfig struct {
	ClusterType ClusterType `json:"clusterType"`
	ClusterName string      `json:"clusterName"`
	Namespace   string      `json:"namespace"`
	// Ray-specific fields
	RayVersion string `json:"rayVersion,omitempty"`
	RayImage   string `json:"rayImage,omitempty"`
	// Volcano-specific fields
	VolcanoWorkerQueue string `json:"workerQueue,omitempty"` // Volcano 特有的字段

	Machines []MachineConfig `json:"machines"`
}

type MachineType string

const (
	MachineTypeSingle MachineType = "single" // 单个机器
	MachineTypeGroup  MachineType = "group"  // 机器组
)

type MachineConfig struct {
	Name            string            `json:"name"`                      // 机器名称
	MachineType     MachineType       `json:"machineType"`               // 机器种类：single 或 group
	CPU             string            `json:"cpu"`                       // CPU 资源
	Memory          string            `json:"memory"`                    // 内存资源
	CustomResources map[string]string `json:"customResources,omitempty"` // 自定义资源（如 GPU）
	Ports           []PortConfig      `json:"ports"`                     // 端口配置
	IsHeadNode      bool              `json:"isHeadNode,omitempty"`      // 是否为头节点

	// 以下字段仅在 MachineType 为 group 时有效
	GroupName   string `json:"groupName,omitempty"`   // 机器组名称
	Replicas    *int32 `json:"replicas,omitempty"`    // 副本数量
	MinReplicas *int32 `json:"minReplicas,omitempty"` // 最小副本数量
	MaxReplicas *int32 `json:"maxReplicas,omitempty"` // 最大副本数量
}
type PortConfig struct {
	Name          string `json:"name"`
	ContainerPort int32  `json:"containerPort"`
}
