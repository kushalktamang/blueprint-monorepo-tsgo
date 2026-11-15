package utils

import (
	"encoding/json"
	"fmt"
)

func PrintJSON(value any) {
	json, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling to JSON:", err)
		return
	}
	fmt.Println("JSON:", string(json))
}
