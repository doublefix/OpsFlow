package tests

import (
	"fmt"
	"testing"

	"github.com/modcoco/OpsFlow/pkg/job"
	"github.com/modcoco/OpsFlow/pkg/model"
	"github.com/modcoco/OpsFlow/pkg/utils"
)

func TestVllmOnRayAutoJob(t *testing.T) {
	// 示例 ClusterConfig
	var replicas int32 = 3
	clusterConfig := model.ClusterConfig{
		ClusterType: "ray",
		ClusterName: "test-cluster",
		Namespace:   "default",
		Job: &model.JobConfig{
			Kind: "vllmOnRaySimpleAutoJob",
			Name: "deepseek-r1-671b",
		},
		Machines: []model.MachineConfig{
			{
				Name:        "node-1",
				IsHeadNode:  true,
				MachineType: model.MachineTypeSingle,
				CPU:         "8",
				Memory:      "16Gi",
				CustomResources: map[string]model.CustomResource{
					"nvidia.com/gpu": {
						Quantity: "8",
					},
				},
				Volumes: []model.VolumeConfig{
					{
						Name: "model-volume",
						Label: map[string]string{
							"model": "true",
						},
						MountPath: "/mnt/data/models/DeepSeek-R1",
					},
				},
			},
			{
				Name:        "node-2",
				MachineType: model.MachineTypeSingle,
				CPU:         "8",
				Memory:      "16Gi",
				CustomResources: map[string]model.CustomResource{
					"nvidia.com/gpu": {
						Quantity: "8",
					},
				},
				Volumes: []model.VolumeConfig{
					{
						Name: "model-volume",
						Label: map[string]string{
							"model": "true",
						},
						MountPath: "/mnt/data/models/DeepSeek-R1",
					},
				},
				Replicas: &replicas,
			},
		},
	}

	err := job.ProcessVllmOnRaySimpleAutoJobClusterConfigByHeaderMachine(&clusterConfig)
	if err != nil {
		fmt.Println(err)
	}
	utils.MarshalToJSON(clusterConfig)
}
