package tasks

import (
	"time"

	"github.com/redis/go-redis/v9"
)

type TaskConfig struct {
	Duration          time.Duration // 任务调度周期
	TaskFunc          TaskFunc      // 任务函数
	WaitForCompletion bool          // 是否等待上一个任务完成
}

func InitializeTasks() map[string]TaskConfig {
	return map[string]TaskConfig{
		"task1": {10 * time.Second, task1Func, true},  // 每 10 秒执行一次，等待上一个任务完成
		"task2": {20 * time.Second, task2Func, false}, // 每 20 秒执行一次，不等待上一个任务完成
		"task3": {30 * time.Second, task3Func, true},  // 每 30 秒执行一次，等待上一个任务完成
	}
}

func StartTaskScheduler(redisClient *redis.ClusterClient, tasks map[string]TaskConfig) {
	for taskName, taskConfig := range tasks {
		go scheduleTask(redisClient, taskName, taskConfig.Duration, taskConfig.TaskFunc, taskConfig.WaitForCompletion)
	}
}
