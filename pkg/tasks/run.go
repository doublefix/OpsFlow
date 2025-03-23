package tasks

import (
	"time"

	"github.com/redis/go-redis/v9"
	"k8s.io/client-go/kubernetes"
)

type TaskConfig struct {
	Duration          time.Duration // 任务调度周期
	TaskFunc          TaskFunc      // 任务函数
	WaitForCompletion bool          // 是否等待上一个任务完成
}

func InitializeTasks(clientset kubernetes.Interface, redisClient *redis.ClusterClient) map[string]TaskConfig {
	// config := &QueueConfig{
	// 	Clientset:   clientset,
	// 	RedisClient: redisClient,
	// 	QueueName:   "task_queue",
	// 	PageSize:    50,
	// }
	return map[string]TaskConfig{
		"task1": {10 * time.Second, task1Func, true},
		"task2": {20 * time.Second, task2Func, false},
		"task3": {30 * time.Second, task3Func, true},
		// "add_update_node_info": {30 * time.Second, AddNodeCheckJobToQueue(ctx, config), true},
	}
}

func StartTaskScheduler(redisClient *redis.ClusterClient, tasks map[string]TaskConfig) {
	for taskName, taskConfig := range tasks {
		go scheduleTask(redisClient, taskName, taskConfig.Duration, taskConfig.TaskFunc, taskConfig.WaitForCompletion)
	}
}
