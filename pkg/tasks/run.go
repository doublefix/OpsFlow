package tasks

import (
	"context"
	"time"

	"github.com/modcoco/OpsFlow/pkg/core"
	"github.com/modcoco/OpsFlow/pkg/crd"
	"github.com/redis/go-redis/v9"
)

type TaskConfig struct {
	Duration          time.Duration // 任务调度周期
	TaskFunc          TaskFunc      // 任务函数
	WaitForCompletion bool          // 是否等待上一个任务完成
}

func InitializeTasks(clent core.Client, redisClient *redis.ClusterClient) map[string]TaskConfig {
	updateNodeInfoConfig := &QueueConfig{
		Clientset:   clent.Core(),
		RedisClient: redisClient,
		QueueName:   "task_queue",
		PageSize:    50,
	}

	deleteNodeInfoConfig := crd.DeleteNodeResourceInfoOptions{
		CRDClient:   clent.DynamicNRI(),
		KubeClient:  clent.Core(),
		Parallelism: 3,
	}

	return map[string]TaskConfig{
		"task1": {10 * time.Second, func(ctx context.Context) error {
			return task1Func(ctx)
		}, true},
		"task2": {20 * time.Second, func(ctx context.Context) error {
			return task2Func(ctx)
		}, false},
		"task3": {30 * time.Second, func(ctx context.Context) error {
			return task3Func(ctx)
		}, true},
		"add_update_node_info": {
			Duration: 30 * time.Second,
			TaskFunc: func(ctx context.Context) error {
				return UpdateNodeInfo(ctx, updateNodeInfoConfig)
			},
			WaitForCompletion: true,
		},
		"del_node_info": {
			Duration: 40 * time.Second,
			TaskFunc: func(ctx context.Context) error {
				return DeleteNonExistingNodeResourceInfoTask(ctx, deleteNodeInfoConfig)
			},
			WaitForCompletion: true,
		},
	}
}

func StartTaskScheduler(redisClient *redis.ClusterClient, tasks map[string]TaskConfig) {
	for taskName, taskConfig := range tasks {
		go scheduleTask(context.Background(), redisClient, taskName, taskConfig.Duration, taskConfig.TaskFunc, taskConfig.WaitForCompletion)
	}
}
