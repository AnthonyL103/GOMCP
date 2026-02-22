package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/execute/add_numbers", handleAddNumbers)

	log.Printf("Starting math_ops on port 9002")
	log.Fatal(http.ListenAndServe(":9002", nil))
}

func handleAddNumbers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	json.NewDecoder(r.Body).Decode(&params)
	num1 := params["num1"].(float64)
	num2 := params["num2"].(float64)
	sum := num1 + num2
	result := map[string]interface{}{"sum": sum}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

