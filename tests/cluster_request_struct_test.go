package tests

import (
	"testing"

	"github.com/modcoco/OpsFlow/internal"
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
	internal.MarshalToJSON(rayConfig)

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
	internal.MarshalToJSON(volcanoConfig)
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

type MachineConfig struct {
	Name            string            `json:"name"`
	CPU             string            `json:"cpu"`
	Memory          string            `json:"memory"`
	CustomResources map[string]string `json:"customResources,omitempty"` // 自定义资源
	Ports           []PortConfig      `json:"ports"`
	IsHeadNode      bool              `json:"isHeadNode,omitempty"`
}

type PortConfig struct {
	Name          string `json:"name"`
	ContainerPort int32  `json:"containerPort"`
}
