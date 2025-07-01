package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/google/uuid"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const lowerLetters = "abcdefghijklmnopqrstuvwxyz"

var src = rand.NewSource(time.Now().UnixNano())

const (
	// 6 bits to represent a letter index
	letterIdBits = 6
	// All 1-bits as many as letterIdBits
	letterIdMask = 1<<letterIdBits - 1
	letterIdMax  = 63 / letterIdBits
)

const (
	letterIdBitsLower = 5
	letterIdMaskLower = 1<<letterIdBitsLower - 1
	letterIdMaxLower  = 63 / letterIdBitsLower
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

func RandStrLower(n int) string {
	b := make([]byte, n)
	for i, cache, remain := 0, src.Int63(), letterIdMaxLower; i < n; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdMaxLower
		}
		if idx := int(cache & letterIdMaskLower); idx < len(lowerLetters) {
			b[i] = lowerLetters[idx]
			i++
		}
		cache >>= letterIdBitsLower
		remain--
	}
	return string(b)
}

func StrPtr(s string) *string {
	return &s
}

func GenerateUniqueStr(name string) string {
	randomSuffix := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s", name, randomSuffix)
}

func ScaledValue(mem resource.Quantity, scale resource.Scale) int64 {
	byteValue := mem.Value()
	shift := int(scale / 3)
	conversionFactor := int64(1) << (10 * uint(shift))
	return byteValue / conversionFactor
}
