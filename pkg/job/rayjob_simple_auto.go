package job

import (
	"fmt"
	"strconv"

	"github.com/modcoco/OpsFlow/pkg/model"
)

// Build runcode config
func ProcessVllmOnRaySimpleAutoJobClusterConfigByHeaderMachine(clusterConfig *model.ClusterConfig) (*VllmSimpleRunCodeConfigForRayCluster, error) {
	if clusterConfig.Job == nil || clusterConfig.Job.Kind != "vllmOnRaySimpleAutoJob" {
		return nil, nil
	}
	// Get cluster total machine count
	machineTypeCount := CountMachines(clusterConfig)
	if machineTypeCount.TotalMachines == 0 {
		return nil, fmt.Errorf("total machine size is zero")
	}

	// Get config
	var modelPath string
	var headerMachine *model.MachineConfig

	// Get selectedMachine
	for _, machine := range clusterConfig.Machines {
		if machine.IsHeadNode { // First
			headerMachine = &machine
			break
		}
	}
	if headerMachine == nil && len(clusterConfig.Machines) > 0 {
		headerMachine = &clusterConfig.Machines[0]
	}
	if headerMachine == nil {
		return nil, fmt.Errorf("no header machine")
	}

	// Get header machine modelPath
	for _, volume := range headerMachine.Volumes {
		if _, exists := volume.Label["model"]; exists {
			if path, ok := volume.Label["actualModelPathInPod"]; ok {
				modelPath = path
			} else if volume.MountPath != "" {
				modelPath = volume.MountPath
			} else {
				return nil, fmt.Errorf("no model volume, or path is none")
			}
			break
		}
	}
	// Get header machine nvidia GPU count
	var countHeaderMaicheNvidiaGPU int
	if val, exists := headerMachine.CustomResources["nvidia.com/gpu"]; exists {
		intValue, err := strconv.Atoi(val.Quantity)
		if err != nil {
			return nil, fmt.Errorf("nvidia GPU value is none")
		}
		countHeaderMaicheNvidiaGPU = intValue
	}
	if countHeaderMaicheNvidiaGPU == 0 {
		return nil, fmt.Errorf("no gpu")
	}

	vllmJobSimple := VllmSimpleAutoJobScriptParams{
		RayJobName:           clusterConfig.Job.Name,
		ModelPath:            modelPath,
		TensorParallelSize:   countHeaderMaicheNvidiaGPU,
		PipelineParallelSize: machineTypeCount.TotalMachines,
	}

	vllmCodeConfigMap, err := GetVllmOnRaySimpleAutoJobConfigMap(vllmJobSimple)
	if err != nil {
		return nil, fmt.Errorf("can't create configmap")
	}

	// Add runcode to header machine
	var targetHeaderMachine *model.MachineConfig
	for i := range clusterConfig.Machines {
		if clusterConfig.Machines[i].IsHeadNode {
			targetHeaderMachine = &clusterConfig.Machines[i]
			break
		}
	}
	if targetHeaderMachine == nil {
		targetHeaderMachine = &clusterConfig.Machines[0]
	}
	if targetHeaderMachine == nil {
		return nil, fmt.Errorf("no machine available to add volume")
	}
	if vllmCodeConfigMap == nil {
		return nil, fmt.Errorf("vllmCodeConfigMap is nil")
	}
	targetHeaderMachine.Volumes = append(targetHeaderMachine.Volumes, vllmCodeConfigMap.VolumeConfig)

	// Add run cmd
	clusterConfig.Job.Cmd = "python " + vllmCodeConfigMap.RunCodeFilePathAndScriptName

	fmt.Println(vllmCodeConfigMap.RunCode)
	return vllmCodeConfigMap, nil
}

type MachineTypeCount struct {
	TotalMachines    int `json:"totalMachines"`
	HeadNodeCount    int `json:"headNodeCount"`
	NonHeadNodeCount int `json:"nonHeadNodeCount"`
}

func CountMachines(clusterConfig *model.ClusterConfig) MachineTypeCount {
	stats := MachineTypeCount{}

	for _, machine := range clusterConfig.Machines {
		count := 1
		if machine.Replicas != nil {
			count = int(*machine.Replicas)
		}

		stats.TotalMachines += count

		if machine.IsHeadNode {
			stats.HeadNodeCount += count
		} else {
			stats.NonHeadNodeCount += count
		}
	}

	return stats
}
