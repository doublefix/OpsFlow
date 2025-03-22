package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type TaskFunc func()

func task1Func() {
	log.Println("Executing task 1 specific logic...")
	time.Sleep(2 * time.Second)
	log.Println("Task 1 completed")
}

func task2Func() {
	log.Println("Executing task 2 specific logic...")
	time.Sleep(3 * time.Second)
	log.Println("Task 2 completed")
}

func task3Func() {
	log.Println("Executing task 3 specific logic...")
	time.Sleep(4 * time.Second)
	log.Println("Task 3 completed")
}

func runTask(redisClient *redis.ClusterClient, taskName string, taskFunc TaskFunc) {
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

	// 锁获取成功，执行任务
	log.Printf("Running task %s...", taskName)

	// 执行任务函数
	taskFunc()

	// 执行完成，释放锁
	_, err = redisClient.Del(ctx, lockKey).Result()
	if err != nil {
		log.Printf("Error releasing lock for %s: %v", taskName, err)
		return
	}

	log.Printf("Task %s completed", taskName)
}

func scheduleTask(redisClient *redis.ClusterClient, taskName string, tickerDuration time.Duration, taskFunc TaskFunc) {
	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	for range ticker.C {
		runTask(redisClient, taskName, taskFunc)
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

	// 定义任务名称、调度周期及对应的任务函数
	tasks := map[string]struct {
		duration time.Duration
		taskFunc TaskFunc
	}{
		"task1": {10 * time.Second, task1Func}, // 每 10 秒执行一次 task1Func
		"task2": {20 * time.Second, task2Func}, // 每 20 秒执行一次 task2Func
		"task3": {30 * time.Second, task3Func}, // 每 30 秒执行一次 task3Func
	}

	for taskName, taskConfig := range tasks {
		go scheduleTask(redisClient, taskName, taskConfig.duration, taskConfig.taskFunc)
	}

	select {}
}
