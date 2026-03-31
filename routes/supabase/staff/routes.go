package staff

import (
	"encoding/json"
	"fmt"
	"net/http"

	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
)

type StaffHandler struct {
	Supabase *supabase.Client
}

// GetAllStaff fetches all staff members from the database
func (h *StaffHandler) GetAllStaff(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	fmt.Println("GetAllStaff: Received request")

	// Query all staff from the database using raw rows first (like GetPatients)
	var rawRows []map[string]interface{}
	err := h.Supabase.DB.From("staff").Select("*").Execute(&rawRows)

	if err != nil {
		msg := fmt.Sprintf("GetAllStaff: DB select error: %v", err)
		fmt.Println(msg)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}

	fmt.Printf("GetAllStaff: Found %d raw rows\n", len(rawRows))

	if len(rawRows) == 0 {
		// Return empty array instead of error if no staff found
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]model.Staff{})
		return
	}

	// Convert raw rows to typed structs
	var staff []model.Staff
	for _, row := range rawRows {
		// Marshal the row back to JSON, then unmarshal into typed Staff
		rowJSON, _ := json.Marshal(row)
		var s model.Staff
		if err := json.Unmarshal(rowJSON, &s); err != nil {
			fmt.Printf("Error unmarshaling staff: %v, row: %v\n", err, row)
			continue
		}
		staff = append(staff, s)
	}

	fmt.Printf("GetAllStaff: Successfully converted %d staff members\n", len(staff))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(staff)
}
