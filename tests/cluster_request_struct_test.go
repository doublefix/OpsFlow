package tests

import (
	"testing"

	"github.com/modcoco/OpsFlow/pkg/utils"
)

func TestGenerateUserRequest(t *testing.T) {
	// Ray 集群示例
	rayConfig := ClusterConfig{
		ClusterType: "ray", // 标明集群类型
		ClusterName: "raycluster-kuberay",
		Namespace:   "chess-kuberay",
		RayVersion:  "2.41.0",
		RayImage:    "rayproject/ray:2.41.0",
		Machines: []MachineConfig{
			{
				Name:   "node-1",
				CPU:    "2",
				Memory: "4Gi",
				CustomResources: map[string]string{
					"GPU": "1", // 用户自定义资源
				},
				Ports: []PortConfig{{Name: "gcs-server", ContainerPort: 6379}},
			},
			{
				Name:   "node-2",
				CPU:    "1",
				Memory: "1Gi",
				CustomResources: map[string]string{
					"GPU": "2", // 用户自定义资源
				},
				Ports: []PortConfig{{Name: "gcs-server", ContainerPort: 6379}},
			},
		},
	}
	utils.MarshalToJSON(rayConfig)

	// Volcano 集群示例
	volcanoConfig := ClusterConfig{
		ClusterType:        "volcano", // 标明集群类型
		ClusterName:        "volcanocluster-kuberay",
		Namespace:          "chess-volcano",
		VolcanoWorkerQueue: "default", // Volcano 特有的字段
		Machines: []MachineConfig{
			{
				Name:   "node-1",
				CPU:    "4",
				Memory: "8Gi",
				CustomResources: map[string]string{
					"GPU": "2", // 用户自定义资源
				},
				Ports:      []PortConfig{{Name: "job-scheduler", ContainerPort: 8080}},
				IsHeadNode: true,
			},
			{
				Name:   "node-2",
				CPU:    "2",
				Memory: "4Gi",
				CustomResources: map[string]string{
					"GPU": "1", // 用户自定义资源
				},
				Ports:      []PortConfig{{Name: "job-scheduler", ContainerPort: 8080}},
				IsHeadNode: false,
			},
		},
	}
	utils.MarshalToJSON(volcanoConfig)
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

// TODO: 不同的集群启动必须加上参数检查，报错啥参数不对
