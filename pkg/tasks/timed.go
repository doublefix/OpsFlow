package tasks

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

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

func runTask(ctx context.Context, redisClient *redis.ClusterClient, taskName string, taskFunc TaskFunc, wg *sync.WaitGroup) {
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
	err = taskFunc(ctx)
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

func scheduleTask(ctx context.Context, redisClient *redis.ClusterClient, taskName string, tickerDuration time.Duration, taskFunc TaskFunc, waitForCompletion bool) {
	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	var wg sync.WaitGroup

	for {
		select {
		case <-ticker.C:
			if waitForCompletion {
				wg.Wait() // 等待上一个任务完成
			}

			wg.Add(1)
			go runTask(ctx, redisClient, taskName, taskFunc, &wg)
		case <-ctx.Done():
			// 上下文取消，停止调度
			log.Printf("Stopping task scheduler for %s due to context cancellation", taskName)
			return
		}
	}
}
