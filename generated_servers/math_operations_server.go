package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/execute/add_numbers", handleAddNumbers)

	log.Printf("Starting math_operations on port 9001")
	log.Fatal(http.ListenAndServe(":9001", nil))
}

func handleAddNumbers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
	    http.Error(w, err.Error(), http.StatusBadRequest)
	    return
	}

	num1, ok1 := params["num1"].(float64)
	num2, ok2 := params["num2"].(float64)

	if !ok1 || !ok2 {
	    http.Error(w, "Invalid parameters", http.StatusBadRequest)
	    return
	}

	sum := num1 + num2

	result := map[string]interface{}{
	    "sum": sum,
	    "num1": num1,
	    "num2": num2,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

