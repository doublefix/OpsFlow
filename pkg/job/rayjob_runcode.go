package job

import (
	"errors"
	"fmt"
	"html/template"
	"maps"
	"strconv"
	"strings"

	"github.com/modcoco/OpsFlow/pkg/model"
	"github.com/modcoco/OpsFlow/pkg/utils"
)

const pythonTemplate = `
import ray
import subprocess

@ray.remote
def start_vllm():
    {{.CommandStr | safe}}
    process = subprocess.Popen(command)
    process.wait()

if __name__ == "__main__":
    ray.init()
    ray.get(start_vllm.remote())
`

// 通过常见参数构建简单的vllm运行脚本
type VllmSimpleAutoJobScriptParams struct {
	RayJobName           string
	ModelPath            string
	TensorParallelSize   int
	PipelineParallelSize int
	Args                 []model.ArgItem
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
		return nil, errors.New("missing or invalid parameters: " + strings.Join(missingParams, ","))
	}

	configMapName := fmt.Sprintf("runcode-%s-%s", input.RayJobName, utils.RandStrLower(5))
	scriptName := fmt.Sprintf("vllm_%s.py", input.RayJobName)

	baseCommand := []string{
		"vllm", "serve", input.ModelPath,
	}

	baseVllmParamMap := map[string]string{
		"--tensor-parallel-size":   strconv.Itoa(input.TensorParallelSize),
		"--pipeline-parallel-size": strconv.Itoa(input.PipelineParallelSize),
		"--trust-remote-code":      "",
	}

	// Change params
	for _, argItem := range input.Args {
		if value, exists := argItem.Label["vllmRuncodeCustomParams"]; exists && value == "true" {
			maps.Copy(baseVllmParamMap, argItem.Params)
		}
	}
	for param, value := range baseVllmParamMap {
		baseCommand = append(baseCommand, param)
		if value != "" {
			baseCommand = append(baseCommand, value)
		}
	}

	// Fix ""
	quotedCommand := make([]string, len(baseCommand))
	for i, item := range baseCommand {
		quotedCommand[i] = fmt.Sprintf(`"%s"`, item)
	}
	commandStr := fmt.Sprintf("command = [%s]", strings.Join(quotedCommand, ", "))

	// Get runcode
	runCode, err := GenerateRunCode(commandStr)
	if err != nil {
		return nil, err
	}

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

func GenerateRunCode(commandStr string) (string, error) {
	funcMap := template.FuncMap{
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	tmpl, err := template.New("pythonCode").Funcs(funcMap).Parse(pythonTemplate)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	err = tmpl.Execute(&builder, struct {
		CommandStr string
	}{
		CommandStr: commandStr,
	})
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}
