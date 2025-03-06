package job

import (
	"fmt"
	"strconv"

	"github.com/modcoco/OpsFlow/pkg/model"
)

func ProcessVllmOnRaySimpleAutoJobClusterConfigByHeaderMachine(clusterConfig *model.ClusterConfig) error {
	if clusterConfig.Job == nil || clusterConfig.Job.Kind != "vllmOnRaySimpleAutoJob" {
		return nil
	}
	// Get cluster total machine count
	machineTypeCount := CountMachines(clusterConfig)
	if machineTypeCount.TotalMachines == 0 {
		return fmt.Errorf("total machine size is zero")
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
		return fmt.Errorf("no header machine")
	}

	// Get header machine modelPath
	for _, volume := range headerMachine.Volumes {
		if _, exists := volume.Label["model"]; exists {
			if volume.MountPath == "" {
				return fmt.Errorf("no model Volum, or path is none")
			}
			modelPath = volume.MountPath
			break
		}
	}
	// Get header machine nvidia GPU count
	var countHeaderMaicheNvidiaGPU int
	if val, exists := headerMachine.CustomResources["nvidia.com/gpu"]; exists {
		intValue, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("nvidia GPU value is none")
		}
		countHeaderMaicheNvidiaGPU = intValue
	}
	if countHeaderMaicheNvidiaGPU == 0 {
		return fmt.Errorf("no gpu")
	}

	vllmJobSimple := VllmSimpleAutoJobScriptParams{
		RayJobName:           clusterConfig.Job.Name,
		ModelPath:            modelPath,
		TensorParallelSize:   machineTypeCount.TotalMachines,
		PipelineParallelSize: countHeaderMaicheNvidiaGPU,
	}

	vllmCodeConfigMap, err := GetVllmOnRaySimpleAutoJobConfigMap(vllmJobSimple)
	if err != nil {
		fmt.Println(err)
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
		return fmt.Errorf("no machine available to add volume")
	}
	if vllmCodeConfigMap == nil {
		return fmt.Errorf("vllmCodeConfigMap is nil")
	}
	targetHeaderMachine.Volumes = append(targetHeaderMachine.Volumes, vllmCodeConfigMap.VolumeConfig)

	// Add run cmd
	clusterConfig.Job.Cmd = "python " + vllmCodeConfigMap.RunCodeFilePath

	fmt.Println(vllmCodeConfigMap.RunCode)
	return nil
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
