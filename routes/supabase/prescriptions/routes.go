package prescriptions

import (
	"encoding/json"
	"io"
	"net/http"

	"fmt"

	"github.com/gorilla/mux"
	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
)

type PrescriptionHandler struct {
	Supabase *supabase.Client
}

func (h *PrescriptionHandler) GetPrescriptions(w http.ResponseWriter, r *http.Request) {
	var prescriptions []model.Prescription
	err := h.Supabase.DB.From("prescriptions").Select("*,patient:patients(name)").Execute(&prescriptions)
	if err != nil {
		msg := fmt.Sprintf("GetPrescriptions: DB select error: %v", err)
		fmt.Println(msg)
		http.Error(w, "Failed to fetch prescriptions", http.StatusInternalServerError)
		return
	}
	if len(prescriptions) == 0 {
		http.Error(w, "No prescriptions found", http.StatusNotFound)
		return
	}
	prescriptionsJSON, err := json.MarshalIndent(prescriptions, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling prescriptions json in Get Prescriptions:", err)
		http.Error(w, "Failed to convert prescriptions to JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(prescriptionsJSON)

}

func (h *PrescriptionHandler) GetPrescriptionByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var prescriptions []model.Prescription
	err := h.Supabase.DB.
		From("prescriptions").
		Select("*,patient:patients(name)").
		Eq("id", id).
		Execute(&prescriptions)
	if err != nil {
		msg := fmt.Sprintf("GetPrescriptionByID: DB error (id=%s): %v", id, err)
		fmt.Println(msg)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if len(prescriptions) == 0 {
		http.Error(w, "Prescription not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(prescriptions[0])
	if err != nil {
		http.Error(w, "Failed to encode prescription", http.StatusInternalServerError)
	}
}

func (h *PrescriptionHandler) GetMultiplePrescriptionsByID(w http.ResponseWriter, r *http.Request) {
	type PrescriptionMessageRequest struct {
		PrescriptionID string `json:"prescription_id"`
	}

	var prescriptionMessageRequest []PrescriptionMessageRequest
	bodyBytes, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(bodyBytes, &prescriptionMessageRequest)
	if err != nil {
		msg := fmt.Sprintf("GetPatientByID: DB select error: %v", err)
		fmt.Println(msg)
		http.Error(w, "Error fetching patient", http.StatusInternalServerError)
		return
	}

	var prescription []model.Prescription // holds query output

	var list1 = []string{}

	for _, message := range prescriptionMessageRequest {
		list1 = append(list1, message.PrescriptionID)
	}

	// Queries database for patient using ID from URL, unmarshals into patient struct and returns error, if any
	err = h.Supabase.DB.From("prescriptions").Select("*,patient:patients(name)").In("id", list1).Execute(&prescription)

	if err != nil {
		msg := fmt.Sprintf("GetPrescriptionByID: DB select error: %v", err)
		fmt.Println(msg)
		http.Error(w, "Error fetching prescriptions", http.StatusInternalServerError)
		return
	}

	if len(prescription) == 0 {
		http.Error(w, "Prescription not found", http.StatusNotFound) // 404
		return
	}

	// fmt.Println("Patient found:", patient)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prescription)
}

func (h *PrescriptionHandler) GetPrescriptionsByPatientID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var prescription []model.Prescription
	err := h.Supabase.DB.From("prescriptions").Select("*,patient:patients(name)").Eq("patient_id", id).Execute(&prescription)
	if err != nil {
		msg := fmt.Sprintf("GetPrescriptionsByPatientID: DB error (patient_id=%s): %v", id, err)
		fmt.Println(msg)
		http.Error(w, "Prescriptions not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prescription)

}
