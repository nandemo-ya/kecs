package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	// Call the admin API directly to test instance listing
	resp, err := http.Get("http://localhost:8081/api/instances")
	if err != nil {
		fmt.Printf("Error calling API: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return
	}

	fmt.Printf("Instances: %+v\n", result)
}