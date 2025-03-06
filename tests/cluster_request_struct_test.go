package tests

import (
	"testing"

	"github.com/modcoco/OpsFlow/pkg/model"
	"github.com/modcoco/OpsFlow/pkg/utils"
)

func TestGenerateUserRequest(t *testing.T) {
	// Ray 集群示例
	rayConfig := model.ClusterConfig{
		ClusterType: "ray", // 标明集群类型
		ClusterName: "raycluster-kuberay",
		Namespace:   "chess-kuberay",
		RayVersion:  "2.41.0",
		RayImage:    "rayproject/ray:2.41.0",
		Machines: []model.MachineConfig{
			{
				Name:   "node-1",
				CPU:    "2",
				Memory: "4Gi",
				CustomResources: map[string]string{
					"GPU": "1", // 用户自定义资源
				},
				Ports: []model.PortConfig{{Name: "gcs-server", ContainerPort: 6379}},
			},
			{
				Name:   "node-2",
				CPU:    "1",
				Memory: "1Gi",
				CustomResources: map[string]string{
					"GPU": "2", // 用户自定义资源
				},
				Ports: []model.PortConfig{{Name: "gcs-server", ContainerPort: 6379}},
			},
		},
	}
	utils.MarshalToJSON(rayConfig)

	// Volcano 集群示例
	volcanoConfig := model.ClusterConfig{
		ClusterType:        "volcano", // 标明集群类型
		ClusterName:        "volcanocluster-kuberay",
		Namespace:          "chess-volcano",
		VolcanoWorkerQueue: "default", // Volcano 特有的字段
		Machines: []model.MachineConfig{
			{
				Name:   "node-1",
				CPU:    "4",
				Memory: "8Gi",
				CustomResources: map[string]string{
					"GPU": "2", // 用户自定义资源
				},
				Ports:      []model.PortConfig{{Name: "job-scheduler", ContainerPort: 8080}},
				IsHeadNode: true,
			},
			{
				Name:   "node-2",
				CPU:    "2",
				Memory: "4Gi",
				CustomResources: map[string]string{
					"GPU": "1", // 用户自定义资源
				},
				Ports:      []model.PortConfig{{Name: "job-scheduler", ContainerPort: 8080}},
				IsHeadNode: false,
			},
		},
	}
	utils.MarshalToJSON(volcanoConfig)
}
