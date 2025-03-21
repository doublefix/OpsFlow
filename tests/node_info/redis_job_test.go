package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func TestRedisJobs(t *testing.T) {
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

	val, err := redisClient.Get(ctx, "foo").Result()
	if err != nil {
		panic(err)
	}

	fmt.Println("foo:", val)
}
