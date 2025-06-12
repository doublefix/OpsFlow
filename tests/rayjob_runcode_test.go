package tests

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"maps"

	"github.com/modcoco/OpsFlow/pkg/job"
	"github.com/modcoco/OpsFlow/pkg/model"
	"github.com/modcoco/OpsFlow/pkg/utils"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
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

	config, err := job.ProcessVllmOnRaySimpleAutoJobClusterConfigByHeaderMachine(&clusterConfig)
	if err != nil {
		fmt.Println(err)
	}
	if config != nil {
		fmt.Println(config.RunCode)
	}
	utils.MarshalToJSON(clusterConfig)
}

func TestGenPythonCode(t *testing.T) {
	input := job.VllmSimpleAutoJobScriptParams{
		RayJobName:           "test",
		ModelPath:            "/mnt/data/test",
		TensorParallelSize:   2,
		PipelineParallelSize: 1,
		Args: []model.ArgItem{
			{
				Label: map[string]string{
					"vllmRuncodeCustomParams": "true",
				},
				Params: map[string]string{
					"--tensor-parallel-size":   "4000",
					"--pipeline-parallel-size": "2000",
					"--swap-space":             "800",
				},
			},
		},
	}
	fmt.Println(input.RayJobName)
	fmt.Println(input.ModelPath)
	fmt.Println(input.TensorParallelSize)
	fmt.Println(input.PipelineParallelSize)
	baseCommand := []string{
		"vllm", "serve", "/mnt/data",
	}

	vllmParamMap := map[string]string{
		"--tensor-parallel-size":   "100",
		"--pipeline-parallel-size": "200",
		"--trust-remote-code":      "",
	}

	// Change params
	for _, argItem := range input.Args {
		if value, exists := argItem.Label["vllmRuncodeCustomParams"]; exists && value == "true" {
			maps.Copy(vllmParamMap, argItem.Params)
		}
	}

	// Add new params
	for param, value := range vllmParamMap {
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
	runCode, err := job.GenerateRunCode(commandStr)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(runCode)
}

func TestGenRayJob(t *testing.T) {
	rayJob := rayv1.RayJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rayjob",
			Namespace: "default",
		},
		Spec: rayv1.RayJobSpec{
			Entrypoint:     "python main.py",
			RayClusterSpec: &rayv1.RayClusterSpec{},
		},
	}
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	// _ = rayv1.AddToScheme(scheme)

	serializer := json.NewSerializerWithOptions(
		json.DefaultMetaFactory,
		scheme,
		scheme,
		json.SerializerOptions{
			Yaml:   false,
			Pretty: true,
			Strict: true,
		},
	)

	var buf bytes.Buffer
	if err := serializer.Encode(&rayJob, &buf); err != nil {
		t.Fatalf("Failed to serialize deployment: %v", err)
	}
	jsonOutput := buf.String()
	t.Logf("Serialized Deployment JSON:\n%s", jsonOutput)

	// decodedObj, _, err := serializer.Decode(buf.Bytes(), nil, nil)
	// if err != nil {
	// 	t.Fatalf("Failed to deserialize: %v", err)
	// }

	// rayjob, ok := decodedObj.(*rayv1.RayJob)
	// if !ok {
	// 	t.Fatalf("Decoded object is not a RayJob")
	// }

	// t.Logf("%s", rayjob.Name)
}
