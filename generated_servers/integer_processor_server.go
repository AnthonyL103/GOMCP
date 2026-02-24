package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

func main() {
	http.HandleFunc("/execute/process_to_integer", handleProcessToInteger)

	log.Printf("Starting integer_processor on port 9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}

func handleProcessToInteger(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
		json.NewDecoder(r.Body).Decode(&params)
	
		input1 := params["input1"].(string)
		input2 := params["input2"].(string)
	
		// Convert inputs to integers and add them
		val1, err1 := strconv.Atoi(input1)
		val2, err2 := strconv.Atoi(input2)
	
		var resultInt int
		if err1 == nil && err2 == nil {
			resultInt = val1 + val2
		} else {
			// If conversion fails, return sum of string lengths as integer
			resultInt = len(input1) + len(input2)
		}
	
		result := map[string]interface{}{
			"result": resultInt,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
}

