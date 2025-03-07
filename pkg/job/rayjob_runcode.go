package job

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/modcoco/OpsFlow/pkg/model"
	"github.com/modcoco/OpsFlow/pkg/utils"
)

// 通过常见参数构建简单的vllm运行脚本
type VllmSimpleAutoJobScriptParams struct {
	RayJobName           string
	ModelPath            string
	TensorParallelSize   int
	PipelineParallelSize int
}

type VllmSimpleRunCodeConfigForRayCluster struct {
	ConfigMapName                string
	VolumeConfig                 model.VolumeConfig
	RunCodeFilePath              string
	RunCodeFilePathAndScriptName string
	ScriptName                   string
	RunCode                      string
}

func GetVllmOnRaySimpleAutoJobConfigMap(input VllmSimpleAutoJobScriptParams) (*VllmSimpleRunCodeConfigForRayCluster, error) {
	var missingParams []string
	if input.RayJobName == "" {
		missingParams = append(missingParams, "RayJobName")
	}
	if input.ModelPath == "" {
		missingParams = append(missingParams, "ModelPath")
	}
	if input.TensorParallelSize <= 0 {
		missingParams = append(missingParams, "TensorParallelSize")
	}
	if input.PipelineParallelSize <= 0 {
		missingParams = append(missingParams, "PipelineParallelSize")
	}

	if len(missingParams) > 0 {
		return nil, errors.New("missing or invalid parameters: " + strings.Join(missingParams, ", "))
	}

	configMapName := fmt.Sprintf("runcode-%s-%s", input.RayJobName, utils.RandStr(5))
	scriptName := fmt.Sprintf("vllm_%s.py", input.RayJobName)
	runCode := fmt.Sprintf(`
import ray
import subprocess

@ray.remote
def start_vllm():
    command = [
        "vllm",
        "serve",
        "%s",
        "--tensor-parallel-size",
        "%s",
        "--pipeline-parallel-size",
        "%s",
        "--trust-remote-code",
    ]
    process = subprocess.Popen(command)
    process.wait()

if __name__ == "__main__":
    ray.init()
    ray.get(start_vllm.remote())
`, input.ModelPath, strconv.Itoa(input.TensorParallelSize), strconv.Itoa(input.PipelineParallelSize))

	runCodeFilePath := "/home/ray/.runcode"
	runCodeFilePathAndScriptName := runCodeFilePath + "/" + scriptName
	volumeConfig := model.VolumeConfig{
		Name:      "volume-" + configMapName,
		MountPath: runCodeFilePath,
		Source: model.VolumeSource{
			ConfigMap: &model.ConfigMapSource{
				Name: configMapName,
				Items: []model.KeyToPathItem{
					{Key: scriptName, Path: scriptName},
				},
			},
		},
	}

	return &VllmSimpleRunCodeConfigForRayCluster{
		ConfigMapName:                configMapName,
		RunCode:                      runCode,
		RunCodeFilePath:              runCodeFilePath,
		RunCodeFilePathAndScriptName: runCodeFilePathAndScriptName,
		ScriptName:                   scriptName,
		VolumeConfig:                 volumeConfig,
	}, nil
}
