package internal

import (
	"encoding/json"
	"fmt"
	"log"
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
