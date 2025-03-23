package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type TaskFunc func() error

func task1Func() error {
	log.Println("Executing task 1 specific logic...")
	time.Sleep(70 * time.Second) // 模拟长时间任务
	log.Println("Task 1 completed")
	return nil
}

func task2Func() error {
	log.Println("Executing task 2 specific logic...")
	time.Sleep(3 * time.Second)
	log.Println("Task 2 completed")
	return nil
}

func task3Func() error {
	log.Println("Executing task 3 specific logic...")
	time.Sleep(4 * time.Second)
	log.Println("Task 3 completed")
	return nil
}

func renewLock(redisClient *redis.ClusterClient, lockKey string, lockExpire time.Duration, stopChan chan struct{}) {
	ticker := time.NewTicker(lockExpire / 2) // 每过期时间的一半续期一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 续期锁
			_, err := redisClient.Expire(ctx, lockKey, lockExpire).Result()
			if err != nil {
				log.Printf("Error renewing lock for key %s: %v", lockKey, err)
				continue // 续期失败，重试
			}
			log.Printf("Lock renewed for key %s", lockKey)
		case <-stopChan:
			// 停止续期
			log.Printf("Stopping lock renewal for key %s", lockKey)
			return
		}
	}
}

func runTask(redisClient *redis.ClusterClient, taskName string, taskFunc TaskFunc, wg *sync.WaitGroup) {
	defer wg.Done() // 任务完成后通知 WaitGroup

	lockKey := "job_lock:" + taskName
	lockValue := "locked"
	lockExpire := 60 * time.Second // 锁过期时间

	// 尝试获取锁，确保只有一个实例在执行任务
	set, err := redisClient.SetNX(ctx, lockKey, lockValue, lockExpire).Result()
	if err != nil {
		log.Printf("Error acquiring lock for %s: %v", taskName, err)
		return
	}

	if !set {
		// 如果锁已经被其他实例获取，跳过任务
		log.Printf("Lock already acquired for task %s, skipping execution", taskName)
		return
	}

	// 锁获取成功，启动锁续期
	stopChan := make(chan struct{})
	go renewLock(redisClient, lockKey, lockExpire, stopChan)

	// 执行任务函数
	log.Printf("Running task %s...", taskName)
	err = taskFunc()
	if err != nil {
		log.Printf("Task %s failed: %v", taskName, err)
	}

	// 任务完成，停止续期并释放锁
	close(stopChan)
	_, err = redisClient.Del(ctx, lockKey).Result()
	if err != nil {
		log.Printf("Error releasing lock for %s: %v", taskName, err)
		return
	}

	log.Printf("Task %s completed", taskName)
}

func scheduleTask(redisClient *redis.ClusterClient, taskName string, tickerDuration time.Duration, taskFunc TaskFunc, waitForCompletion bool) {
	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	var wg sync.WaitGroup // 用于等待任务完成

	for range ticker.C {
		if waitForCompletion {
			// 如果配置为等待模式，等待上一个任务完成
			wg.Wait()
		}

		wg.Add(1) // 标记新任务开始
		go runTask(redisClient, taskName, taskFunc, &wg)
	}
}

func main() {
	redisClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"10.187.6.3:31000",
			"10.187.6.4:31001",
			"10.187.6.5:31002",
			"10.187.6.3:31100",
			"10.187.6.4:31101",
			"10.187.6.5:31102",
		},
		Password: "pass12345",
	})

	// 定义任务名称、调度周期、任务函数及是否等待上一个任务完成
	tasks := map[string]struct {
		duration          time.Duration
		taskFunc          TaskFunc
		waitForCompletion bool
	}{
		"task1": {10 * time.Second, task1Func, true},  // 每 10 秒执行一次，等待上一个任务完成
		"task2": {20 * time.Second, task2Func, false}, // 每 20 秒执行一次，不等待上一个任务完成
		"task3": {30 * time.Second, task3Func, true},  // 每 30 秒执行一次，等待上一个任务完成
	}

	// 为每个任务启动一个独立的调度器
	for taskName, taskConfig := range tasks {
		go scheduleTask(redisClient, taskName, taskConfig.duration, taskConfig.taskFunc, taskConfig.waitForCompletion)
	}

	// 保持程序运行
	select {}
}

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
