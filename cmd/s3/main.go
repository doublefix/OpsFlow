package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

//  mahongqin minio4399

const (
	endpoint     = "https://s3.openpaper.co:9000"
	accessKey    = "NuIcHmlpQxrRi4LnF67p"
	secretKey    = "JVXujRZzHI1WS5Efc6YrCThVvLJfYXDgZEODlHW9"
	region       = "us-east-1"
	usePathStyle = true
)

func main() {
	// 加载基础配置
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 构建 s3.Client（推荐方式，无需 aws.Endpoint）
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = usePathStyle
		o.BaseEndpoint = aws.String(endpoint) // 👈 推荐设置方式，取代 aws.Endpoint
	})

	// 测试 List Buckets
	resp, err := s3Client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		log.Fatalf("列出 Buckets 失败: %v", err)
	}

	fmt.Println("Buckets:")
	for _, b := range resp.Buckets {
		fmt.Printf("- %s\n", *b.Name)
	}
}
