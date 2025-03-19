package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedis(t *testing.T) {
	ctx := context.Background()

	// Cmdable 是 Client 和 ClusterClient 的通用接口
	var redisClient redis.Cmdable

	if true {
		redisClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs: []string{
				"10.187.6.3:31000",
				"10.187.6.4:31001",
				"10.187.6.5:31002",
				"10.187.6.3:31003",
				"10.187.6.4:31004",
				"10.187.6.5:31005",
			},
			Password: "pass12345",
		})
	} else {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     "10.187.6.5:30531",
			Password: "pass12345",
		})
	}

	err := redisClient.Set(ctx, "foo", "bar", 0).Err()
	if err != nil {
		panic(err)
	}

	val, err := redisClient.Get(ctx, "foo").Result()
	if err != nil {
		panic(err)
	}

	fmt.Println("foo:", val)
}
