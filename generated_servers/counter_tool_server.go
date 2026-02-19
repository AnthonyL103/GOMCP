package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/execute/count_to_ten", handleCountToTen)

	log.Printf("Starting counter_tool on port 9002")
	log.Fatal(http.ListenAndServe(":9002", nil))
}

func handleCountToTen(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
	  params = make(map[string]interface{})
	}
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := map[string]interface{}{"status": "success", "numbers": numbers, "count": 10}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

