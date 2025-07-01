package job

import (
	"strings"

	"github.com/modcoco/OpsFlow/pkg/model"
	"github.com/modcoco/OpsFlow/pkg/utils"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func CreateConfigMapFromVllmSimpleRunCodeConfig(config VllmSimpleRunCodeConfigForRayCluster) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.ConfigMapName,
		},
		Data: map[string]string{
			config.ScriptName: config.RunCode,
		},
	}
}

func CreateHeadGroupSpec(machines []model.MachineConfig, rayImage string) rayv1.HeadGroupSpec {
	var headMachine *model.MachineConfig
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
		resourceList[corev1.ResourceName(key)] = resource.MustParse(value.Quantity)
		if strings.HasPrefix(key, "nvidia.com") {
			runtimeClassName = ptr.To("nvidia")
		}
	}

	// Create volumes and volume mounts from the machine config
	volumes, volumeMounts := BuildVolumesAndMounts(headMachine.Volumes)
	return rayv1.HeadGroupSpec{
		RayStartParams: map[string]string{},
		Template: corev1.PodTemplateSpec{
			// ObjectMeta: metav1.ObjectMeta{
			// 	CreationTimestamp: metav1.Time{Time: time.Now()},
			// },
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  machines[0].Name,
						Image: rayImage,
						Resources: corev1.ResourceRequirements{
							Limits:   resourceList,
							Requests: resourceList,
						},
						Ports:        utils.ConvertPorts(headMachine.Ports),
						VolumeMounts: volumeMounts,
					},
				},
				RuntimeClassName: runtimeClassName,
				Volumes:          volumes,
			},
		},
	}
}

func CreateWorkerGroupSpecs(machines []model.MachineConfig, rayImage string) []rayv1.WorkerGroupSpec {
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
				resourceList[corev1.ResourceName(key)] = resource.MustParse(value.Quantity)
				if strings.HasPrefix(key, "nvidia.com") {
					runtimeClassName = ptr.To("nvidia")
				}
			}

			// 根据 MachineType 设置 Replicas、MinReplicas 和 MaxReplicas
			var replicas, minReplicas, maxReplicas *int32
			var groupName string

			if machine.MachineType == model.MachineTypeGroup {
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

			// 创建 volumes 和 volume mounts
			volumes, volumeMounts := BuildVolumesAndMounts(machine.Volumes)

			// 创建 WorkerGroupSpec
			workerGroupSpec := rayv1.WorkerGroupSpec{
				Replicas:       replicas,
				MinReplicas:    minReplicas,
				MaxReplicas:    maxReplicas,
				GroupName:      groupName,
				RayStartParams: map[string]string{},
				Template: corev1.PodTemplateSpec{
					// ObjectMeta: metav1.ObjectMeta{
					// 	CreationTimestamp: metav1.Time{Time: time.Now()},
					// },
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  machine.Name,
								Image: rayImage,
								Resources: corev1.ResourceRequirements{
									Limits:   resourceList,
									Requests: resourceList,
								},
								Ports:        utils.ConvertPorts(machine.Ports),
								VolumeMounts: volumeMounts,
							},
						},
						RuntimeClassName: runtimeClassName,
						Volumes:          volumes,
					},
				},
			}
			workerGroupSpecs = append(workerGroupSpecs, workerGroupSpec)
		}
	}

	return workerGroupSpecs
}

func BuildVolumesAndMounts(volumesConfig []model.VolumeConfig) ([]corev1.Volume, []corev1.VolumeMount) {
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount

	for _, volume := range volumesConfig {
		// 挂载 PVC
		if volume.Source.PVC != nil {
			volumes = append(volumes, corev1.Volume{
				Name: volume.Name,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: volume.Source.PVC.ClaimName,
					},
				},
			})

			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      volume.Name,
				MountPath: volume.MountPath,
			})
		}

		// 挂载 ConfigMap
		if volume.Source.ConfigMap != nil {
			volumes = append(volumes, corev1.Volume{
				Name: volume.Name,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: volume.Source.ConfigMap.Name,
						},
						Items: func() []corev1.KeyToPath {
							var items []corev1.KeyToPath
							for _, item := range volume.Source.ConfigMap.Items {
								items = append(items, corev1.KeyToPath{
									Key:  item.Key,
									Path: item.Path,
								})
							}
							return items
						}(),
					},
				},
			})

			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      volume.Name,
				MountPath: volume.MountPath,
				ReadOnly:  true,
			})
		}
	}

	return volumes, volumeMounts
}
