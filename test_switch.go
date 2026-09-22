package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	var obj interface{}
	j := `{"HEADERS": {"A": "B"}}`
	json.Unmarshal([]byte(j), &obj)

	m := obj.(map[string]interface{})
	for k, v := range m {
		switch v.(type) {
		case []interface{}, map[string]interface{}:
			b, _ := json.Marshal(v)
			fmt.Printf("MATCHED! %s = %s\n", k, string(b))
			continue
		}
		fmt.Printf("FELL THROUGH! %s = %v\n", k, v)
	}
}
