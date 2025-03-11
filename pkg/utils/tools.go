package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/uuid"
	"github.com/modcoco/OpsFlow/pkg/model"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var src = rand.NewSource(time.Now().UnixNano())

const (
	// 6 bits to represent a letter index
	letterIdBits = 6
	// All 1-bits as many as letterIdBits
	letterIdMask = 1<<letterIdBits - 1
	letterIdMax  = 63 / letterIdBits
)

func MarshalToJSON(obj any) string {
	ojJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling object to JSON: %v\n", err)
		return "{}"
	}
	log.Println(string(ojJSON))
	return string(ojJSON)
}

func RandStr(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdMax letters!
	for i, cache, remain := 0, src.Int63(), letterIdMax; i < n; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdMax
		}
		if idx := int(cache & letterIdMask); idx < len(letters) {
			b[i] = letters[idx]
			i++
		}
		cache >>= letterIdBits
		remain--
	}
	return string(b)
}

// ConvertPorts 将端口配置转换为 ContainerPort 列表
func ConvertPorts(ports []model.PortConfig) []corev1.ContainerPort {
	var containerPorts []corev1.ContainerPort
	for _, port := range ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          port.Name,
			ContainerPort: port.ContainerPort,
		})
	}
	return containerPorts
}

func StrPtr(s string) *string {
	return &s
}

func GenerateUniqueStr(name string) string {
	randomSuffix := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s", name, randomSuffix)
}
