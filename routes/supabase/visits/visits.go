package visits

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
)

type VisitHandler struct {
	Supabase *supabase.Client
}

type VisitService interface {
	GetPatientVisits(w http.ResponseWriter, r *http.Request)
	CreatePatientVisit(w http.ResponseWriter, r *http.Request)
}

// GetPatientVisits - Get all visits for a specific patient
func (h *VisitHandler) GetPatientVisits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	patientID := vars["patient_id"]

	// Fetch all visits for this patient
	var visits []model.PatientVisit
	err := h.Supabase.DB.From("patient_visits").
		Select("*").
		Eq("patient_id", patientID).
		Execute(&visits)

	if err != nil {
		fmt.Println("Error fetching visits:", err)
		http.Error(w, "Failed to fetch visits", http.StatusInternalServerError)
		return
	}

	// Enrich with provider names
	visitsWithProviders := []model.PatientVisitWithProvider{}

	for _, visit := range visits {
		visitWithProvider := model.PatientVisitWithProvider{
			PatientVisit: visit,
			ProviderName: "Unknown Provider",
		}

		// Get provider name if provider_id exists
		if visit.ProviderId != nil {
			var users []struct {
				Name string `json:"name"`
			}
			err := h.Supabase.DB.From("users").
				Select("name").
				Eq("id", visit.ProviderId.String()).
				Execute(&users)

			if err == nil && len(users) > 0 {
				visitWithProvider.ProviderName = users[0].Name
			}
		}

		visitsWithProviders = append(visitsWithProviders, visitWithProvider)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(visitsWithProviders)
}

// CreatePatientVisit - Create a new visit record
func (h *VisitHandler) CreatePatientVisit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PatientId     string `json:"patient_id"`
		ProviderId    string `json:"provider_id"`
		VisitDate     string `json:"visit_date"`
		VisitType     string `json:"visit_type"`
		ClinicalNotes string `json:"clinical_notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Insert visit into database
	insertData := map[string]interface{}{
		"patient_id":     req.PatientId,
		"provider_id":    req.ProviderId,
		"visit_date":     req.VisitDate,
		"visit_type":     req.VisitType,
		"clinical_notes": req.ClinicalNotes,
	}

	err := h.Supabase.DB.From("patient_visits").Insert(insertData).Execute(nil)
	if err != nil {
		fmt.Println("Error creating visit:", err)
		http.Error(w, "Failed to create visit", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Visit created successfully"})
}
