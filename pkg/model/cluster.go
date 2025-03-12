package model

type ClusterType string

const (
	ClusterTypeRay     ClusterType = "ray"
	ClusterTypeVolcano ClusterType = "volcano"
)

type ClusterConfig struct {
	ClusterType ClusterType `json:"clusterType,omitempty"`
	ClusterName string      `json:"clusterName,omitempty"`
	Namespace   string      `json:"namespace"`
	// Ray-specific fields
	RayVersion string `json:"rayVersion,omitempty"`
	RayImage   string `json:"rayImage,omitempty"`
	// Volcano-specific fields
	VolcanoWorkerQueue string `json:"workerQueue,omitempty"` // Volcano 特有的字段

	Machines []MachineConfig `json:"machines"`
	// 可选的 Job 配置
	Job *JobConfig `json:"job,omitempty"`
}

type JobConfig struct {
	Kind          string    `json:"kind,omitempty"`          // 任务种类, vllmOnRayAutoJob(自动构建code与模型地址)
	Name          string    `json:"name"`                    // Job 名称
	Cmd           string    `json:"cmd,omitempty"`           // Job 执行的命令
	TargetCluster string    `json:"targetCluster,omitempty"` // 目标集群（可选）
	Args          []ArgItem `json:"args,omitempty"`          // 自定义参数，适用于修改运行脚本，cmd会运行一个脚本
}
type ArgItem struct {
	Label  map[string]string `json:"label"`
	Params map[string]string `json:"params"`
}

type MachineType string

const (
	MachineTypeSingle MachineType = "single" // 单个机器
	MachineTypeGroup  MachineType = "group"  // 机器组
)

type MachineConfig struct {
	Name            string                    `json:"name"`                      // 机器名称
	MachineType     MachineType               `json:"machineType"`               // 机器种类：single 或 group
	CPU             string                    `json:"cpu"`                       // CPU 资源
	Memory          string                    `json:"memory"`                    // 内存资源
	CustomResources map[string]CustomResource `json:"customResources,omitempty"` // 自定义资源（如 GPU）
	Ports           []PortConfig              `json:"ports,omitempty"`           // 端口配置
	IsHeadNode      bool                      `json:"isHeadNode,omitempty"`      // 是否为头节点

	// 以下字段仅在 MachineType 为 group 时有效
	GroupName   string `json:"groupName,omitempty"`   // 机器组名称
	Replicas    *int32 `json:"replicas,omitempty"`    // 副本数量
	MinReplicas *int32 `json:"minReplicas,omitempty"` // 最小副本数量
	MaxReplicas *int32 `json:"maxReplicas,omitempty"` // 最大副本数量

	// 以下字段用来挂载卷
	Volumes []VolumeConfig `json:"volumes,omitempty"` // 卷挂载配置
}

type CustomResource struct {
	Quantity string            `json:"quantity"` // 资源类型，如 GPU、TPU、RDMA 等
	Labels   map[string]string `json:"labels"`   // 资源标签
}

type VolumeConfig struct {
	Name      string            `json:"name"`            // 卷名称
	Label     map[string]string `json:"label,omitempty"` // 挂载标签(比如model挂载代表模型，runcode代表运行代码)
	MountPath string            `json:"mountPath"`       // 挂载路径
	Source    VolumeSource      `json:"source"`          // 卷来源
}

// VolumeSource 定义了 PVC 或 ConfigMap 作为卷的来源
type VolumeSource struct {
	PVC       *PVCSource       `json:"pvc,omitempty"`       // 持久化存储卷
	ConfigMap *ConfigMapSource `json:"configMap,omitempty"` // ConfigMap 作为存储卷
}

// PVCSource 表示 PVC 相关信息
type PVCSource struct {
	ClaimName string `json:"claimName"` // PVC 名称
}

// ConfigMapSource 表示 ConfigMap 相关信息
type ConfigMapSource struct {
	Name  string          `json:"name"`  // ConfigMap 名称
	Items []KeyToPathItem `json:"items"` // 配置项映射
}

// KeyToPathItem 表示 ConfigMap 的键值到路径的映射
type KeyToPathItem struct {
	Key  string `json:"key"`  // ConfigMap 中的键
	Path string `json:"path"` // 挂载到容器的路径
}

type PortConfig struct {
	Name          string `json:"name"`
	ContainerPort int32  `json:"containerPort"`
}
