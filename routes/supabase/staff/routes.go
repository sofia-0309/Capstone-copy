package staff

import (
	"encoding/json"
	"fmt"
	"net/http"

	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
	"github.com/gorilla/mux"
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

// GetStaffByID fetches a single staff member by ID
func (h *StaffHandler) GetStaffByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]

	t := h.Supabase.DB.From("staff")
	q := t.Select("*").Eq("id", id)

	var rows []map[string]interface{}
	err := q.Execute(&rows)

	// dod the db query fail? return server error if so 
	if err == nil {
		rowCount := len(rows)


		// did the que return any rows, return 404 error if no staff has that ID
		if rowCount != 0 {

			// get the first row from the result
			rowOne := rows[0]

			// convert the row data into json so we can map it into the staff struct
			rowJSON, err := json.Marshal(rowOne)

			// did converting fail to json fail, if it did then there is something
			// wrong with the data forming
			if err == nil {

				var staffMember model.Staff

				// convert the json data into the staff struct
				err = json.Unmarshal(rowJSON, &staffMember)

				// did converting json into the struct fail
				if err == nil {
					// everything worked so return the staff member
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(staffMember)
					return


				} else {

					// error converting json into the staff struct
					fmt.Println("error converting staff", err)
				}

			} else {

				// error converting the row data into json
				fmt.Println("error converting row:", err)

			}

		} else {

			// no rows were returned from the database
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)

			response := map[string]string{
				"error": "staff not found",
			}
			json.NewEncoder(w).Encode(response)
			return

		}

	} else {
		// error occurred while running the database query
		fmt.Println("error getting staff:", err)

	}

	// fallback error if something unexpected happens
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	response := map[string]string{
		"error": "server error",
	}

	json.NewEncoder(w).Encode(response)
}
