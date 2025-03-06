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
	Kind          string `json:"kind,omitempty"`          // 任务种类, vllmOnRayAutoJob(自动构建code与模型地址)
	Name          string `json:"name"`                    // Job 名称
	Cmd           string `json:"cmd,omitempty"`           // Job 执行的命令
	TargetCluster string `json:"targetCluster,omitempty"` // 目标集群（可选）
}

type MachineType string

const (
	MachineTypeSingle MachineType = "single" // 单个机器
	MachineTypeGroup  MachineType = "group"  // 机器组
)

type MachineConfig struct {
	Name            string            `json:"name"`                      // 机器名称
	MachineType     MachineType       `json:"machineType"`               // 机器种类：single 或 group
	CPU             string            `json:"cpu"`                       // CPU 资源
	Memory          string            `json:"memory"`                    // 内存资源
	CustomResources map[string]string `json:"customResources,omitempty"` // 自定义资源（如 GPU）
	Ports           []PortConfig      `json:"ports"`                     // 端口配置
	IsHeadNode      bool              `json:"isHeadNode,omitempty"`      // 是否为头节点

	// 以下字段仅在 MachineType 为 group 时有效
	GroupName   string `json:"groupName,omitempty"`   // 机器组名称
	Replicas    *int32 `json:"replicas,omitempty"`    // 副本数量
	MinReplicas *int32 `json:"minReplicas,omitempty"` // 最小副本数量
	MaxReplicas *int32 `json:"maxReplicas,omitempty"` // 最大副本数量

	// 以下字段用来挂载卷
	Volumes []VolumeConfig `json:"volumes,omitempty"` // 卷挂载配置
}

type VolumeConfig struct {
	Name      string            `json:"name"`                // 挂载名字
	Type      string            `json:"type"`                // 挂载种类
	Label     map[string]string `json:"label,omitempty"`     // 挂载标签(比如model挂载代表模型，runcode代表运行代码)
	Source    *string           `json:"source,omitempty"`    // pvc卷
	Path      *string           `json:"path,omitempty"`      // pvc在pod的卷挂载路径
	ConfigMap *ConfigMapVolume  `json:"configMap,omitempty"` // 挂载 configMap
}

type ConfigMapVolume struct {
	Name  string          `jsdon:"name"` // ConfigMap 名称，必须提前存在
	Items []KeyToPathItem `json:"items"` // 配置项映射
}

type KeyToPathItem struct {
	Key  string `json:"key"`  // ConfigMap 中的键
	Path string `json:"path"` // 挂载到容器的路径
}

type PortConfig struct {
	Name          string `json:"name"`
	ContainerPort int32  `json:"containerPort"`
}
