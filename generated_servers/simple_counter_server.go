package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/execute/count_to_ten", handleCountToTen)

	log.Printf("Starting simple_counter on port 9003")
	log.Fatal(http.ListenAndServe(":9003", nil))
}

func handleCountToTen(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	json.NewDecoder(r.Body).Decode(&params)
	numbers := make([]int, 10)
	for i := 0; i < 10; i++ {
	  numbers[i] = i + 1
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"numbers": numbers})
}

