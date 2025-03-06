package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"
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
