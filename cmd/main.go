package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// 执行任务的函数
func runTask(redisClient *redis.ClusterClient, taskName string) {
	// 为每个任务设置不同的锁
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

	// 模拟任务执行
	time.Sleep(10 * time.Second)

	// 执行完成，释放锁
	_, err = redisClient.Del(ctx, lockKey).Result()
	if err != nil {
		log.Printf("Error releasing lock for %s: %v", taskName, err)
		return
	}

	log.Printf("Task %s completed", taskName)
}

// 任务调度函数
func scheduleTask(redisClient *redis.ClusterClient, taskName string, tickerDuration time.Duration) {
	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	for range ticker.C {
		runTask(redisClient, taskName)
	}
}

func main() {
	// 初始化 Redis 集群客户端
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

	// 定时任务名称及调度周期
	taskNames := []string{"task1", "task2", "task3"}
	tickerDuration := 10 * time.Second // 任务每 10 秒执行一次

	// 为每个任务启动一个独立的调度器
	for _, taskName := range taskNames {
		go scheduleTask(redisClient, taskName, tickerDuration)
	}

	// 保持程序运行
	select {}
}
