package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ProjectTask struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Duration     string    `json:"duration"`
	Start        time.Time `json:"start"`
	Finish       time.Time `json:"finish"`
	Progress     int       `json:"progress"`      // 0-100%
	Dependencies []string  `json:"dependencies"`  // IDs of predecessor tasks
	Assignee     string    `json:"assignee"`
}

// Sample construction project tasks matching your screenshot
var projectTasks = []ProjectTask{
	// PRECONSTRUCTION & PERMITTING
	{ID: "1", Name: "Preconstruction & Permitting", Duration: "45 days", Start: parseDate("2026-02-05"), Finish: parseDate("2026-03-21"), Progress: 0, Dependencies: []string{}},
	{ID: "2", Name: "Contract Award", Duration: "5 days", Start: parseDate("2026-02-05"), Finish: parseDate("2026-02-09"), Progress: 0, Dependencies: []string{"1"}},
	{ID: "3", Name: "Performance & Payment Bonds", Duration: "10 days", Start: parseDate("2026-02-10"), Finish: parseDate("2026-02-19"), Progress: 0, Dependencies: []string{"2"}},
	{ID: "4", Name: "Building & Civil Permit Approval", Duration: "30 days", Start: parseDate("2026-02-05"), Finish: parseDate("2026-03-06"), Progress: 0, Dependencies: []string{"1"}},
	{ID: "5", Name: "Submittals & Shop Drawings", Duration: "20 days", Start: parseDate("2026-02-20"), Finish: parseDate("2026-03-10"), Progress: 0, Dependencies: []string{"3"}},
	{ID: "6", Name: "Procurement of Long-Lead Items", Duration: "35 days", Start: parseDate("2026-02-20"), Finish: parseDate("2026-03-26"), Progress: 0, Dependencies: []string{"5"}},

	// MOBILIZATION
	{ID: "7", Name: "Mobilization", Duration: "10 days", Start: parseDate("2026-03-22"), Finish: parseDate("2026-03-31"), Progress: 0, Dependencies: []string{"4", "3"}},
	{ID: "8", Name: "Temporary Facilities & Utilities", Duration: "5 days", Start: parseDate("2026-03-22"), Finish: parseDate("2026-03-26"), Progress: 0, Dependencies: []string{"7"}},
	{ID: "9", Name: "Site Safety & Erosion Control", Duration: "2 days", Start: parseDate("2026-03-27"), Finish: parseDate("2026-03-28"), Progress: 0, Dependencies: []string{"8"}},

	// DEMOLITION & SITE PREP
	{ID: "10", Name: "Demolition & Existing Conditions", Duration: "15 days", Start: parseDate("2026-03-29"), Finish: parseDate("2026-04-12"), Progress: 0, Dependencies: []string{"9"}},
	{ID: "11", Name: "Utility Locate", Duration: "3 days", Start: parseDate("2026-03-29"), Finish: parseDate("2026-03-31"), Progress: 0, Dependencies: []string{"9"}},
	{ID: "12", Name: "Minor Demolition - Adjacent Structure", Duration: "8 days", Start: parseDate("2026-04-01"), Finish: parseDate("2026-04-08"), Progress: 0, Dependencies: []string{"11"}},
	{ID: "13", Name: "Selective Removal", Duration: "7 days", Start: parseDate("2026-04-01"), Finish: parseDate("2026-04-07"), Progress: 0, Dependencies: []string{"11"}},

	// SITE WORK & UNDERGROUND UTILITIES
	{ID: "14", Name: "Clearing & Grubbing", Duration: "5 days", Start: parseDate("2026-04-13"), Finish: parseDate("2026-04-17"), Progress: 0, Dependencies: []string{"10"}},
	{ID: "15", Name: "Rough Grading", Duration: "5 days", Start: parseDate("2026-04-18"), Finish: parseDate("2026-04-22"), Progress: 0, Dependencies: []string{"14"}},
	{ID: "16", Name: "Underground Storm, Sanitary & Water", Duration: "12 days", Start: parseDate("2026-04-23"), Finish: parseDate("2026-05-04"), Progress: 0, Dependencies: []string{"15"}},
	{ID: "17", Name: "Underground Electrical & Communications", Duration: "10 days", Start: parseDate("2026-05-05"), Finish: parseDate("2026-05-14"), Progress: 0, Dependencies: []string{"16"}},
	{ID: "18", Name: "Underground Fire Line", Duration: "8 days", Start: parseDate("2026-05-05"), Finish: parseDate("2026-05-12"), Progress: 0, Dependencies: []string{"16"}},
	{ID: "19", Name: "Utility Inspections", Duration: "3 days", Start: parseDate("2026-05-15"), Finish: parseDate("2026-05-17"), Progress: 0, Dependencies: []string{"17", "18"}},
	{ID: "20", Name: "Backfill & Compaction", Duration: "6 days", Start: parseDate("2026-05-18"), Finish: parseDate("2026-05-23"), Progress: 0, Dependencies: []string{"19"}},

	// FOUNDATIONS & SUBSTRUCTURE
	{ID: "21", Name: "Foundation Excavation", Duration: "8 days", Start: parseDate("2026-05-24"), Finish: parseDate("2026-05-31"), Progress: 0, Dependencies: []string{"20"}},
	{ID: "22", Name: "Footings - Form, Reinforce, Pour", Duration: "10 days", Start: parseDate("2026-06-01"), Finish: parseDate("2026-06-10"), Progress: 0, Dependencies: []string{"21"}},
	{ID: "23", Name: "Foundation Walls - Form, Reinforce, Pour", Duration: "12 days", Start: parseDate("2026-06-11"), Finish: parseDate("2026-06-22"), Progress: 0, Dependencies: []string{"22"}},
	{ID: "24", Name: "Dampproofing & Waterproofing", Duration: "5 days", Start: parseDate("2026-06-23"), Finish: parseDate("2026-06-27"), Progress: 0, Dependencies: []string{"23"}},
	{ID: "25", Name: "Under-Slab Utilities", Duration: "4 days", Start: parseDate("2026-06-23"), Finish: parseDate("2026-06-26"), Progress: 0, Dependencies: []string{"23"}},
	{ID: "26", Name: "Slab-on-Grade Preparation & Placement", Duration: "8 days", Start: parseDate("2026-06-27"), Finish: parseDate("2026-07-04"), Progress: 0, Dependencies: []string{"24", "25"}},
	{ID: "27", Name: "Foundation Inspection", Duration: "1 day", Start: parseDate("2026-07-05"), Finish: parseDate("2026-07-05"), Progress: 0, Dependencies: []string{"26"}},

	// STRUCTURAL FRAME
	{ID: "28", Name: "Structural Steel Fabrication & Delivery", Duration: "30 days", Start: parseDate("2026-02-20"), Finish: parseDate("2026-03-21"), Progress: 0, Dependencies: []string{"5"}},
	{ID: "29", Name: "Level 1 Steel Erection", Duration: "12 days", Start: parseDate("2026-07-06"), Finish: parseDate("2026-07-17"), Progress: 0, Dependencies: []string{"27", "28"}},
	{ID: "30", Name: "Level 1 Connections & Bracing", Duration: "8 days", Start: parseDate("2026-07-18"), Finish: parseDate("2026-07-25"), Progress: 0, Dependencies: []string{"29"}},
	{ID: "31", Name: "Level 2 Steel Erection", Duration: "12 days", Start: parseDate("2026-07-26"), Finish: parseDate("2026-08-06"), Progress: 0, Dependencies: []string{"30"}},
	{ID: "32", Name: "Level 2 Connections & Bracing", Duration: "8 days", Start: parseDate("2026-08-07"), Finish: parseDate("2026-08-14"), Progress: 0, Dependencies: []string{"31"}},
	{ID: "33", Name: "Metal Floor & Roof Deck Installation", Duration: "10 days", Start: parseDate("2026-08-15"), Finish: parseDate("2026-08-24"), Progress: 0, Dependencies: []string{"32"}},
	{ID: "34", Name: "Structural Inspection", Duration: "2 days", Start: parseDate("2026-08-25"), Finish: parseDate("2026-08-26"), Progress: 0, Dependencies: []string{"33"}},

	// ROOFING
	{ID: "35", Name: "Roof Insulation & TPO Installation", Duration: "10 days", Start: parseDate("2026-08-27"), Finish: parseDate("2026-09-05"), Progress: 0, Dependencies: []string{"34"}},
	{ID: "36", Name: "Flashing & Roof Drains", Duration: "5 days", Start: parseDate("2026-09-06"), Finish: parseDate("2026-09-10"), Progress: 0, Dependencies: []string{"35"}},
	{ID: "37", Name: "Roof Inspection", Duration: "1 day", Start: parseDate("2026-09-11"), Finish: parseDate("2026-09-11"), Progress: 0, Dependencies: []string{"36"}},

	// BUILDING ENVELOPE
	{ID: "38", Name: "Level 1 Wall Framing", Duration: "8 days", Start: parseDate("2026-08-27"), Finish: parseDate("2026-09-03"), Progress: 0, Dependencies: []string{"34"}},
	{ID: "39", Name: "Level 1 Sheathing & Weather Barrier", Duration: "6 days", Start: parseDate("2026-09-04"), Finish: parseDate("2026-09-09"), Progress: 0, Dependencies: []string{"38"}},
	{ID: "40", Name: "Level 1 Windows & Glazing", Duration: "8 days", Start: parseDate("2026-09-10"), Finish: parseDate("2026-09-17"), Progress: 0, Dependencies: []string{"39"}},
	{ID: "41", Name: "Level 1 Masonry / Metal Panels", Duration: "10 days", Start: parseDate("2026-09-18"), Finish: parseDate("2026-09-27"), Progress: 0, Dependencies: []string{"39"}},
	{ID: "42", Name: "Level 2 Wall Framing", Duration: "8 days", Start: parseDate("2026-09-04"), Finish: parseDate("2026-09-11"), Progress: 0, Dependencies: []string{"34"}},
	{ID: "43", Name: "Level 2 Sheathing & Weather Barrier", Duration: "6 days", Start: parseDate("2026-09-12"), Finish: parseDate("2026-09-17"), Progress: 0, Dependencies: []string{"42"}},
	{ID: "44", Name: "Level 2 Windows & Glazing", Duration: "8 days", Start: parseDate("2026-09-18"), Finish: parseDate("2026-09-25"), Progress: 0, Dependencies: []string{"43"}},
	{ID: "45", Name: "Level 2 Masonry / Metal Panels", Duration: "10 days", Start: parseDate("2026-09-26"), Finish: parseDate("2026-10-05"), Progress: 0, Dependencies: []string{"43"}},
	{ID: "46", Name: "Envelope Inspection", Duration: "1 day", Start: parseDate("2026-10-06"), Finish: parseDate("2026-10-06"), Progress: 0, Dependencies: []string{"41", "45", "37"}},

	// INTERIOR ROUGH-IN
	{ID: "47", Name: "Level 1 Interior Framing", Duration: "8 days", Start: parseDate("2026-10-07"), Finish: parseDate("2026-10-14"), Progress: 0, Dependencies: []string{"46"}},
	{ID: "48", Name: "Level 1 Mechanical Rough-In", Duration: "10 days", Start: parseDate("2026-10-15"), Finish: parseDate("2026-10-24"), Progress: 0, Dependencies: []string{"47"}},
	{ID: "49", Name: "Level 1 Plumbing Rough-In", Duration: "10 days", Start: parseDate("2026-10-15"), Finish: parseDate("2026-10-24"), Progress: 0, Dependencies: []string{"47"}},
	{ID: "50", Name: "Level 1 Fire Sprinkler Rough-In", Duration: "8 days", Start: parseDate("2026-10-15"), Finish: parseDate("2026-10-22"), Progress: 0, Dependencies: []string{"47"}},
	{ID: "51", Name: "Level 1 Electrical & Low Voltage Rough-In", Duration: "10 days", Start: parseDate("2026-10-15"), Finish: parseDate("2026-10-24"), Progress: 0, Dependencies: []string{"47"}},
	{ID: "52", Name: "Level 1 Radiation Shielding", Duration: "5 days", Start: parseDate("2026-10-25"), Finish: parseDate("2026-10-29"), Progress: 0, Dependencies: []string{"48", "49", "50", "51"}},

	{ID: "53", Name: "Level 2 Interior Framing", Duration: "8 days", Start: parseDate("2026-10-30"), Finish: parseDate("2026-11-06"), Progress: 0, Dependencies: []string{"46"}},
	{ID: "54", Name: "Level 2 Mechanical Rough-In", Duration: "10 days", Start: parseDate("2026-11-07"), Finish: parseDate("2026-11-16"), Progress: 0, Dependencies: []string{"53"}},
	{ID: "55", Name: "Level 2 Plumbing Rough-In", Duration: "10 days", Start: parseDate("2026-11-07"), Finish: parseDate("2026-11-16"), Progress: 0, Dependencies: []string{"53"}},
	{ID: "56", Name: "Level 2 Fire Sprinkler Rough-In", Duration: "8 days", Start: parseDate("2026-11-07"), Finish: parseDate("2026-11-14"), Progress: 0, Dependencies: []string{"53"}},
	{ID: "57", Name: "Level 2 Electrical & Low Voltage Rough-In", Duration: "10 days", Start: parseDate("2026-11-07"), Finish: parseDate("2026-11-16"), Progress: 0, Dependencies: []string{"53"}},
	{ID: "58", Name: "Level 2 Radiation Shielding", Duration: "5 days", Start: parseDate("2026-11-17"), Finish: parseDate("2026-11-21"), Progress: 0, Dependencies: []string{"54", "55", "56", "57"}},
	{ID: "59", Name: "Rough-In Inspection", Duration: "2 days", Start: parseDate("2026-11-22"), Finish: parseDate("2026-11-23"), Progress: 0, Dependencies: []string{"52", "58"}},

	// INTERIOR FINISHES
	{ID: "60", Name: "Level 1 Drywall & Finish", Duration: "12 days", Start: parseDate("2026-11-24"), Finish: parseDate("2026-12-05"), Progress: 0, Dependencies: []string{"59"}},
	{ID: "61", Name: "Level 1 Ceilings", Duration: "8 days", Start: parseDate("2026-12-06"), Finish: parseDate("2026-12-13"), Progress: 0, Dependencies: []string{"60"}},
	{ID: "62", Name: "Level 1 Painting", Duration: "8 days", Start: parseDate("2026-12-14"), Finish: parseDate("2026-12-21"), Progress: 0, Dependencies: []string{"61"}},
	{ID: "63", Name: "Level 1 Flooring", Duration: "10 days", Start: parseDate("2026-12-22"), Finish: parseDate("2026-12-31"), Progress: 0, Dependencies: []string{"62"}},
	{ID: "64", Name: "Level 1 Doors & Hardware", Duration: "6 days", Start: parseDate("2027-01-01"), Finish: parseDate("2027-01-06"), Progress: 0, Dependencies: []string{"60"}},
	{ID: "65", Name: "Level 1 Casework & Fixtures", Duration: "8 days", Start: parseDate("2027-01-07"), Finish: parseDate("2027-01-14"), Progress: 0, Dependencies: []string{"64"}},
	{ID: "66", Name: "Level 1 Lighting & Device Trim", Duration: "5 days", Start: parseDate("2027-01-15"), Finish: parseDate("2027-01-19"), Progress: 0, Dependencies: []string{"65"}},
	{ID: "67", Name: "Level 1 Plumbing Fixtures", Duration: "4 days", Start: parseDate("2027-01-20"), Finish: parseDate("2027-01-23"), Progress: 0, Dependencies: []string{"63"}},

	{ID: "68", Name: "Level 2 Drywall & Finish", Duration: "12 days", Start: parseDate("2026-11-24"), Finish: parseDate("2026-12-05"), Progress: 0, Dependencies: []string{"59"}},
	{ID: "69", Name: "Level 2 Ceilings", Duration: "8 days", Start: parseDate("2026-12-06"), Finish: parseDate("2026-12-13"), Progress: 0, Dependencies: []string{"68"}},
	{ID: "70", Name: "Level 2 Painting", Duration: "8 days", Start: parseDate("2026-12-14"), Finish: parseDate("2026-12-21"), Progress: 0, Dependencies: []string{"69"}},
	{ID: "71", Name: "Level 2 Flooring", Duration: "10 days", Start: parseDate("2026-12-22"), Finish: parseDate("2026-12-31"), Progress: 0, Dependencies: []string{"70"}},
	{ID: "72", Name: "Level 2 Doors & Hardware", Duration: "6 days", Start: parseDate("2027-01-01"), Finish: parseDate("2027-01-06"), Progress: 0, Dependencies: []string{"68"}},
	{ID: "73", Name: "Level 2 Casework & Fixtures", Duration: "8 days", Start: parseDate("2027-01-07"), Finish: parseDate("2027-01-14"), Progress: 0, Dependencies: []string{"72"}},
	{ID: "74", Name: "Level 2 Lighting & Device Trim", Duration: "5 days", Start: parseDate("2027-01-15"), Finish: parseDate("2027-01-19"), Progress: 0, Dependencies: []string{"73"}},
	{ID: "75", Name: "Level 2 Plumbing Fixtures", Duration: "4 days", Start: parseDate("2027-01-20"), Finish: parseDate("2027-01-23"), Progress: 0, Dependencies: []string{"71"}},

	// VERTICAL TRANSPORTATION
	{ID: "76", Name: "Elevator Installation", Duration: "15 days", Start: parseDate("2027-01-24"), Finish: parseDate("2027-02-07"), Progress: 0, Dependencies: []string{"67", "75"}},
	{ID: "77", Name: "Elevator Inspection", Duration: "2 days", Start: parseDate("2027-02-08"), Finish: parseDate("2027-02-09"), Progress: 0, Dependencies: []string{"76"}},

	// SITE IMPROVEMENTS
	{ID: "78", Name: "Concrete Flatwork", Duration: "8 days", Start: parseDate("2026-05-24"), Finish: parseDate("2026-05-31"), Progress: 0, Dependencies: []string{"20"}},
	{ID: "79", Name: "Final Grading", Duration: "5 days", Start: parseDate("2027-02-10"), Finish: parseDate("2027-02-14"), Progress: 0, Dependencies: []string{"77"}},
	{ID: "80", Name: "Landscaping & Irrigation", Duration: "10 days", Start: parseDate("2027-02-15"), Finish: parseDate("2027-02-24"), Progress: 0, Dependencies: []string{"79"}},
	{ID: "81", Name: "Site Lighting & Striping", Duration: "5 days", Start: parseDate("2027-02-25"), Finish: parseDate("2027-03-01"), Progress: 0, Dependencies: []string{"80"}},

	// TESTING, COMMISSIONING & INSPECTIONS
	{ID: "82", Name: "HVAC Testing & Balancing", Duration: "5 days", Start: parseDate("2027-02-08"), Finish: parseDate("2027-02-12"), Progress: 0, Dependencies: []string{"77"}},
	{ID: "83", Name: "Plumbing Testing", Duration: "3 days", Start: parseDate("2027-02-13"), Finish: parseDate("2027-02-15"), Progress: 0, Dependencies: []string{"77"}},
	{ID: "84", Name: "Fire Protection Testing", Duration: "3 days", Start: parseDate("2027-02-16"), Finish: parseDate("2027-02-18"), Progress: 0, Dependencies: []string{"77"}},
	{ID: "85", Name: "Electrical Testing", Duration: "3 days", Start: parseDate("2027-02-19"), Finish: parseDate("2027-02-21"), Progress: 0, Dependencies: []string{"77"}},
	{ID: "86", Name: "Final Building Inspection", Duration: "2 days", Start: parseDate("2027-03-02"), Finish: parseDate("2027-03-03"), Progress: 0, Dependencies: []string{"82", "83", "84", "85", "81"}},
	{ID: "87", Name: "Certificate of Occupancy", Duration: "5 days", Start: parseDate("2027-03-04"), Finish: parseDate("2027-03-08"), Progress: 0, Dependencies: []string{"86"}},

	// CLOSEOUT & OWNER TURNOVER
	{ID: "88", Name: "Punch List Completion", Duration: "7 days", Start: parseDate("2027-03-09"), Finish: parseDate("2027-03-15"), Progress: 0, Dependencies: []string{"87"}},
	{ID: "89", Name: "Final Cleaning", Duration: "3 days", Start: parseDate("2027-03-16"), Finish: parseDate("2027-03-18"), Progress: 0, Dependencies: []string{"88"}},
	{ID: "90", Name: "Owner Training", Duration: "3 days", Start: parseDate("2027-03-19"), Finish: parseDate("2027-03-21"), Progress: 0, Dependencies: []string{"89"}},
	{ID: "91", Name: "O&M Manuals & As-Builts", Duration: "5 days", Start: parseDate("2027-03-22"), Finish: parseDate("2027-03-26"), Progress: 0, Dependencies: []string{"90"}},
	{ID: "92", Name: "Substantial Completion", Duration: "0 days", Start: parseDate("2027-03-27"), Finish: parseDate("2027-03-27"), Progress: 0, Dependencies: []string{"89"}},
	{ID: "93", Name: "Final Completion & Owner Turnover", Duration: "0 days", Start: parseDate("2027-03-28"), Finish: parseDate("2027-03-28"), Progress: 0, Dependencies: []string{"91"}},
}

func parseDate(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}

func main() {
	http.HandleFunc("/execute/get_all_tasks", handleGetAllTasks)
	http.HandleFunc("/execute/get_task", handleGetTask)
	http.HandleFunc("/execute/create_task", handleCreateTask)
	http.HandleFunc("/execute/update_task", handleUpdateTask)
	http.HandleFunc("/execute/delete_task", handleDeleteTask)
	http.HandleFunc("/execute/update_progress", handleUpdateProgress)
	http.HandleFunc("/execute/get_critical_path", handleGetCriticalPath)
	http.HandleFunc("/execute/export_to_xml", handleExportToXML)

	log.Println("Starting Microsoft Project MCP server on port 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

// Get all tasks in the project
func handleGetAllTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projectTasks)
}

// Get specific task by ID
func handleGetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	taskID, ok := params["task_id"].(string)
	if !ok {
		http.Error(w, "Missing task_id", http.StatusBadRequest)
		return
	}

	for _, task := range projectTasks {
		if task.ID == taskID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)
			return
		}
	}

	http.Error(w, "Task not found", http.StatusNotFound)
}

// Create new task
func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Extract task details
	name, _ := params["name"].(string)
	duration, _ := params["duration"].(string)
	assignee, _ := params["assignee"].(string)
	
	// Parse dependencies if provided
	dependencies := []string{}
	if deps, ok := params["dependencies"].([]interface{}); ok {
		for _, dep := range deps {
			if depStr, ok := dep.(string); ok {
				dependencies = append(dependencies, depStr)
			}
		}
	}

	// Create new task with auto-generated ID
	newTask := ProjectTask{
		ID:           fmt.Sprintf("%d", len(projectTasks)+1),
		Name:         name,
		Duration:     duration,
		Start:        time.Now(),
		Finish:       time.Now().Add(24 * time.Hour),
		Progress:     0,
		Dependencies: dependencies,
		Assignee:     assignee,
	}

	projectTasks = append(projectTasks, newTask)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"task":   newTask,
	})
}

// Update task details
func handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	taskID, ok := params["task_id"].(string)
	if !ok {
		http.Error(w, "Missing task_id", http.StatusBadRequest)
		return
	}

	for i, task := range projectTasks {
		if task.ID == taskID {
			// Update fields if provided
			if name, ok := params["name"].(string); ok {
				projectTasks[i].Name = name
			}
			if duration, ok := params["duration"].(string); ok {
				projectTasks[i].Duration = duration
			}
			if assignee, ok := params["assignee"].(string); ok {
				projectTasks[i].Assignee = assignee
			}
			if startDate, ok := params["start"].(string); ok {
				if t, err := time.Parse("2006-01-02", startDate); err == nil {
					projectTasks[i].Start = t
				}
			}
			if finishDate, ok := params["finish"].(string); ok {
				if t, err := time.Parse("2006-01-02", finishDate); err == nil {
					projectTasks[i].Finish = t
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"task":   projectTasks[i],
			})
			return
		}
	}

	http.Error(w, "Task not found", http.StatusNotFound)
}

// Delete task
func handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	taskID, ok := params["task_id"].(string)
	if !ok {
		http.Error(w, "Missing task_id", http.StatusBadRequest)
		return
	}

	for i, task := range projectTasks {
		if task.ID == taskID {
			projectTasks = append(projectTasks[:i], projectTasks[i+1:]...)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"message": "Task deleted",
			})
			return
		}
	}

	http.Error(w, "Task not found", http.StatusNotFound)
}

// Update task progress (0-100%)
func handleUpdateProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	taskID, ok := params["task_id"].(string)
	if !ok {
		http.Error(w, "Missing task_id", http.StatusBadRequest)
		return
	}

	progress, ok := params["progress"].(float64)
	if !ok {
		http.Error(w, "Missing or invalid progress", http.StatusBadRequest)
		return
	}

	for i, task := range projectTasks {
		if task.ID == taskID {
			projectTasks[i].Progress = int(progress)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"task":   projectTasks[i],
			})
			return
		}
	}

	http.Error(w, "Task not found", http.StatusNotFound)
}

// Get critical path (tasks with dependencies)
func handleGetCriticalPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	criticalTasks := []ProjectTask{}
	for _, task := range projectTasks {
		if len(task.Dependencies) > 0 || task.Progress < 100 {
			criticalTasks = append(criticalTasks, task)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(criticalTasks)
}

// Export project to Microsoft Project XML format
func handleExportToXML(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get file path from params or use default
	filePath := "C:/Users/antho/construction_project.xml"
	if path, ok := params["file_path"].(string); ok && path != "" {
		filePath = path
	}

	// Generate Microsoft Project XML
	xmlContent := generateProjectXML(projectTasks)

	// Write to file
	if err := os.WriteFile(filePath, []byte(xmlContent), 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"file":   filePath,
		"message": fmt.Sprintf("Project exported to %s - Open in Microsoft Project", filePath),
	})
}

// Generate Microsoft Project XML format
func generateProjectXML(tasks []ProjectTask) string {
	var sb strings.Builder

	// XML header and Project root
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
`)
	sb.WriteString(`<Project xmlns="http://schemas.microsoft.com/project">
`)

	// Project metadata
	sb.WriteString(`  <Name>Construction Project</Name>
`)
	sb.WriteString(`  <Title>Construction Engineering Management</Title>
`)
	sb.WriteString(fmt.Sprintf(`  <CreationDate>%s</CreationDate>
`, time.Now().Format("2006-01-02T15:04:05")))
	sb.WriteString(`  <ScheduleFromStart>1</ScheduleFromStart>
`)
	sb.WriteString(fmt.Sprintf(`  <StartDate>%s</StartDate>
`, time.Now().Format("2006-01-02T15:04:05")))

	// Tasks section
	sb.WriteString(`  <Tasks>
`)

	// Summary task (root)
	sb.WriteString(`    <Task>
`)
	sb.WriteString(`      <UID>0</UID>
`)
	sb.WriteString(`      <ID>0</ID>
`)
	sb.WriteString(`      <Name>Construction Project</Name>
`)
	sb.WriteString(`      <Type>1</Type>
`)
	sb.WriteString(`      <IsNull>0</IsNull>
`)
	sb.WriteString(`      <Summary>1</Summary>
`)
	sb.WriteString(`    </Task>
`)

	// Individual tasks
	for i, task := range tasks {
		sb.WriteString(`    <Task>
`)
		sb.WriteString(fmt.Sprintf(`      <UID>%d</UID>
`, i+1))
		sb.WriteString(fmt.Sprintf(`      <ID>%d</ID>
`, i+1))
		sb.WriteString(fmt.Sprintf(`      <Name>%s</Name>
`, escapeXML(task.Name)))
		sb.WriteString(`      <Type>0</Type>
`)
		sb.WriteString(`      <IsNull>0</IsNull>
`)
		sb.WriteString(fmt.Sprintf(`      <Start>%s</Start>
`, task.Start.Format("2006-01-02T15:04:05")))
		sb.WriteString(fmt.Sprintf(`      <Finish>%s</Finish>
`, task.Finish.Format("2006-01-02T15:04:05")))
		sb.WriteString(fmt.Sprintf(`      <Duration>PT%dH0M0S</Duration>
`, 8)) // Default 8 hours
		sb.WriteString(fmt.Sprintf(`      <PercentComplete>%d</PercentComplete>
`, task.Progress))
		sb.WriteString(`      <OutlineLevel>1</OutlineLevel>
`)

		// Add dependencies (predecessors)
		if len(task.Dependencies) > 0 {
			sb.WriteString(`      <PredecessorLink>
`)
			for _, depID := range task.Dependencies {
				// Find the UID of the predecessor
				for j, t := range tasks {
					if t.ID == depID {
						sb.WriteString(fmt.Sprintf(`        <PredecessorUID>%d</PredecessorUID>
`, j+1))
						sb.WriteString(`        <Type>1</Type>
`) // Finish-to-Start
						break
					}
				}
			}
			sb.WriteString(`      </PredecessorLink>
`)
		}

		// Add assignee/resource if present
		if task.Assignee != "" {
			sb.WriteString(`      <Assignments>
`)
			sb.WriteString(`        <Assignment>
`)
			sb.WriteString(fmt.Sprintf(`          <ResourceUID>%d</ResourceUID>
`, 1))
			sb.WriteString(fmt.Sprintf(`          <TaskUID>%d</TaskUID>
`, i+1))
			sb.WriteString(`        </Assignment>
`)
			sb.WriteString(`      </Assignments>
`)
		}

		sb.WriteString(`    </Task>
`)
	}

	sb.WriteString(`  </Tasks>
`)

	// Resources section (optional)
	sb.WriteString(`  <Resources>
`)
	sb.WriteString(`    <Resource>
`)
	sb.WriteString(`      <UID>1</UID>
`)
	sb.WriteString(`      <ID>1</ID>
`)
	sb.WriteString(`      <Name>Construction Team</Name>
`)
	sb.WriteString(`      <Type>1</Type>
`)
	sb.WriteString(`    </Resource>
`)
	sb.WriteString(`  </Resources>
`)

	sb.WriteString(`</Project>
`)

	return sb.String()
}

// Escape XML special characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
