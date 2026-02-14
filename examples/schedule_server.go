package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type CalvinsEvent struct {
	Title string `json:"title"`
	Time  string `json:"time"`
}

var Calvins_events = []CalvinsEvent{
	{Title: "Meeting with Bob", Time: "2024-10-01T10:00:00Z"},
	{Title: "Dentist appointment", Time: "2024-10-02T15:00:00Z"},
}

func main() {
	http.HandleFunc("/execute/get_schedule", handleGetSchedule)

	http.HandleFunc("/execute/add_event", handleAddEvent)

	log.Println("Starting test schedule server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}




func handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	password, ok := params["password"].(string)
	if !ok {
		http.Error(w, "Missing or invalid password", http.StatusBadRequest)
		return
	}

	if password != "secret123" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Calvins_events)
}

func handleAddEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	password, ok := params["password"].(string)
	if !ok {
		http.Error(w, "Missing or invalid password", http.StatusBadRequest)
		return
	}

	if password != "secret123" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	events, ok := params["events"].([]interface{})
	if !ok {
		http.Error(w, "Missing or invalid events", http.StatusBadRequest)
		return
	}

	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			Calvins_events = append(Calvins_events, CalvinsEvent{
				Title: eventMap["title"].(string),
				Time:  eventMap["time"].(string),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"events": Calvins_events,
	})
}

	