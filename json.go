package gx

import (
	"encoding/json"
	"fmt"
)

func Json(obj interface{}) string {
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Sprintf("Dump Error: %v", err)
	}
	return string(data)
}

func JsonCompact(obj interface{}) string {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("Dump Error: %v", err)
	}
	return string(data)
}

func JsonPrint(obj interface{}) {
	fmt.Println(Json(obj))
}
