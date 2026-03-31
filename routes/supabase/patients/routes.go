package patients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"math/rand"
	"time"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
	request "gitlab.msu.edu/team-corewell-2025/routes/supabase"
)

type PatientHandler struct {
	Supabase *supabase.Client
}

func (h *PatientHandler) GetPatients(w http.ResponseWriter, r *http.Request) {

	// stores raw data returned from Supabase query
	var rawRows []map[string]interface{}

	// grab all columns
	err := h.Supabase.DB.From("patients").Select("*").Execute(&rawRows)
	if err != nil {
		http.Error(w, "Patients not found", http.StatusNotFound)
		return
	}

	// typed struct
	var patients []model.Patient

	for _, row := range rawRows {
		// Marshal the row back to JSON, then unmarshal into typed Patient
		rowJSON, _ := json.Marshal(row)
		var p model.Patient
		if err := json.Unmarshal(rowJSON, &p); err != nil {
			fmt.Println("Error unmarshaling patient:", err)
			continue
		}
		patients = append(patients, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patients)
}

/**
 * GetPatientByID fetches a patient by ID from the database
 * @param w http.ResponseWriter
 * @param r *http.Request	Authenticated request
 */
func (h *PatientHandler) GetPatientByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"] // gets ID from URL

	var patient []model.Patient // holds query output

	// Queries database for patient using ID from URL, unmarshals into patient struct and returns error, if any
	err := h.Supabase.DB.From("patients").Select("*").Eq("id", id).Execute(&patient)

	if err != nil {
		msg := fmt.Sprintf("GetPatientByID: DB select error (id=%s): %v", id, err)
		fmt.Println(msg)
		http.Error(w, "Error fetching patient", http.StatusInternalServerError)
		return
	}

	if len(patient) == 0 {
		http.Error(w, "Patient not found", http.StatusNotFound) // 404
		return
	}

	// fmt.Println("Patient found:", patient)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patient[0])
}

func (h *PatientHandler) GetMultiplePatientsByID(w http.ResponseWriter, r *http.Request) {
	type PatientMessageRequest struct {
		PatientID string `json:"patient_id"`
	}

	var patientMessageRequest []PatientMessageRequest
	bodyBytes, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(bodyBytes, &patientMessageRequest)
	if err != nil {
		msg := fmt.Sprintf("GetPatientByID: DB select error: %v", err)
		fmt.Println(msg)
		http.Error(w, "Error fetching patient", http.StatusInternalServerError)
		return
	}

	var patient []model.Patient // holds query output

	var list1 = []string{}

	for _, message := range patientMessageRequest {
		list1 = append(list1, message.PatientID)
	}

	// Queries database for patient using ID from URL, unmarshals into patient struct and returns error, if any
	err = h.Supabase.DB.From("patients").Select("*").In("id", list1).Execute(&patient)

	if err != nil {
		msg := fmt.Sprintf("GetPatientByID: DB select error: %v", err)
		fmt.Println(msg)
		http.Error(w, "Error fetching patient", http.StatusInternalServerError)
		return
	}

	if len(patient) == 0 {
		http.Error(w, "Patient not found", http.StatusNotFound) // 404
		return
	}

	// fmt.Println("Patient found:", patient)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patient)
}

func (h *PatientHandler) GetFlaggedPatients(w http.ResponseWriter, r *http.Request) {
	var flaggedPatients []model.FlaggedPatient
	err := h.Supabase.DB.From("flagged").Select("*,patient:patients!flagged_patient_id_fkey(*)").Execute(&flaggedPatients)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error grabbing Flagged Patients", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(flaggedPatients)
	if err != nil {
		msg := fmt.Sprintf("GetFlaggedPatients: error encoding flagged patients: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}
func (h *PatientHandler) AddFlaggedPatient(w http.ResponseWriter, r *http.Request) {
	var req request.FlaggedPatientRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		fmt.Println(err)
		http.Error(w, "Error unmarshaling request", http.StatusBadRequest)
		return
	}

	var existing []request.InsertFlaggedPatient
	err = h.Supabase.DB.
		From("flagged").
		Select("*").
		Eq("patient_id", req.PatientID.String()).
		Execute(&existing)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error checking flagged table", http.StatusInternalServerError)
		return
	}

	if len(existing) == 0 {
		newFlag := request.InsertFlaggedPatient{
			ID:        uuid.New(),
			PatientID: req.PatientID,
			Flaggers:  []uuid.UUID{req.UserID},
			Messages: map[string]string{
				req.Name: req.Explanation,
			},
		}

		err = h.Supabase.DB.
			From("flagged").
			Insert(newFlag).
			Execute(nil)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error inserting new flagged row", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Patient flagged successfully (new row)"))
		return
	}

	flaggedRow := existing[0]

	alreadyFlagged := false
	for _, uid := range flaggedRow.Flaggers {
		if uid == req.UserID {
			alreadyFlagged = true
			break
		}
	}
	if alreadyFlagged {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("User has already flagged this patient"))
		return
	}

	flaggedRow.Flaggers = append(flaggedRow.Flaggers, req.UserID)
	if flaggedRow.Messages == nil {
		flaggedRow.Messages = make(map[string]string)
	}
	flaggedRow.Messages[req.Name] = req.Explanation
	updateData := map[string]interface{}{
		"flaggers": flaggedRow.Flaggers,
		"messages": flaggedRow.Messages,
	}

	err = h.Supabase.DB.
		From("flagged").
		Update(updateData).
		Eq("id", flaggedRow.ID.String()).
		Execute(nil)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error updating flagged row", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Patient flagged successfully (updated existing row)"))
}

func (h *PatientHandler) RemoveFlaggedPatient(w http.ResponseWriter, r *http.Request) {
	var request request.FlaggedPatientRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(bodyBytes, &request)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}
	patientID := request.PatientID.String()
	err = h.Supabase.DB.From("patients").Delete().Eq("id", patientID).Execute(nil)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Could not delete from patients", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Patient Removed"))
}

func (h *PatientHandler) KeepPatient(w http.ResponseWriter, r *http.Request) {
	var request request.FlaggedPatientRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("KeepPatient: failed to read request body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(bodyBytes, &request)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}
	patientID := request.PatientID.String()
	err = h.Supabase.DB.From("flagged").Delete().Eq("patient_id", patientID).Execute(nil)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Could not delete from flagged", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Patient Kept"))
}

func (h *PatientHandler) UpdateFlaggedPatientByID(w http.ResponseWriter, r *http.Request) {
	var request request.UpdateFlaggedPatientByIDRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("UpdatePatient: failed to read request body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(bodyBytes, &request); err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}

	var patients []model.Patient
	err = h.Supabase.DB.From("patients").Select("*").Eq("id", request.PatientID).Execute(&patients)
	if err != nil || len(patients) == 0 {
		http.Error(w, "patient not found", http.StatusNotFound)
		return
	}
	patient := patients[0]
	updateData := map[string]interface{}{}

	if request.Field == "patient.immunization" {
		if request.NewKey != "" && request.NewValue != "" {
			delete(patient.Immunization, request.OldKey)
			patient.Immunization[request.NewKey] = request.NewValue
		} else {
			patient.Immunization[request.OldKey] = request.NewValue
		}
		updateData["immunization"] = patient.Immunization
	} else {
		fieldName := strings.TrimPrefix(request.Field, "patient.")
		updateData[fieldName] = request.NewValue
	}

	if err := h.Supabase.DB.From("patients").Update(updateData).Eq("id", request.PatientID).Execute(nil); err != nil {
		fmt.Println(err)
		http.Error(w, "Could not update patient", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Patient Updated"))
}

// Function to get patients from db
// func(h *PatientHandler) GetPatients( w http.ResponseWriter, r *http.Request){
// 	var request = request.PatientGetRequest

// 		var rows []request.PatientGetRequest
// 	 	h.Supabase.DB.From("patients").Select("*").Execute(&rows)
// 		fmt.Println(rows)

	// bodyBytes, err := io.ReadAll(r.Body)
	// if err != nil {
	// 	msg := fmt.Sprintf("GetPatient: failed to read request body: %v", err)
	// 	fmt.Println(msg)
	// 	http.Error(w, msg, http.StatusBadRequest)
	// 	return
	// }

	// if err := json.Unmarshal(bodyBytes, &request); err != nil {
	// 	fmt.Println(err)
	// 	http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
	// 	return
	// }

	// fmt.Println("Got the call from js") 

//}
//  Function to add patients into the database

func (h *PatientHandler) AddNewPatient(w http.ResponseWriter, r *http.Request) {
	
	var request request.PatientCreateRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("AddPatient: failed to read request body: %v", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(bodyBytes, &request); err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}

	
	fmt.Println("Got the call from js")
	//fmt.Println(request)
	// patient := PatientCreateRequest{
	// 	//ID: uuid.New(),
	// 	name: request.Name,
	// 	date_of_birth: time.Date(1998, 5, 12, 0, 0, 0, 0, time.UTC),
	// 	gender: request.Genre,
	// 	Height: request.Height,
	// 	last_bp: request.BloodPressure,
	// 	medica_history: request.MedicalHistory,
	// 	medical_condition: request.Symptoms,
	// 	family_medical_history: request.FamilyMedicalHistory,	
	// }

	err = h.Supabase.DB.From("featured_patients").Insert(request).Execute(nil)
	if err != nil {
		fmt.Println("Error adding patient: ", err)
	
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "Patient inserted",
	})

}

// Functions to generate patient information on database access page
func (h *PatientHandler) GetRandomName(w http.ResponseWriter, r *http.Request){

	var rows []request.PatientNames
	h.Supabase.DB.From("patients").Select("name").Execute(&rows)
	
	var first_names, last_names []string
	
	
	
	for _, r := range rows {
		var first string
		var last string
		fullname := r.Name
		spaces := strings.Fields(fullname)

		if len(spaces) > 0 {
			first = spaces[0]
		}

		if len(spaces) > 1 {
			last = spaces[len(spaces)-1]
		}
		first_names=append(first_names,first)
		last_names=append(last_names,last)
	}
	

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(first_names))
	randomName := first_names[randomIndex] +" " + last_names[randomIndex]
	fmt.Println(randomName)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"name": randomName,
	})

}

func (h *PatientHandler) GetDemo(w http.ResponseWriter, r *http.Request){
	var rows []request.Demographics
	h.Supabase.DB.From("patients").Select("date_of_birth,gender").Execute(&rows)
	

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(rows))
	randomEntry := rows[randomIndex]
	fmt.Println(randomEntry)

	dob := randomEntry.DateOfBirth
	gender := randomEntry.Gender

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"dob": dob,
		"gender":gender,
	})
	
}

func (h *PatientHandler) GetVitals(w http.ResponseWriter, r *http.Request){
	var rows []request.Vitals
	h.Supabase.DB.From("patients").Select("height,weight,last_bp").Execute(&rows)
	

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(rows))
	randomEntry := rows[randomIndex]
	fmt.Println(randomEntry)

	height := randomEntry.Height
	weight := randomEntry.Weight
	last_bp := randomEntry.Last_bp

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"height": height,
		"weight":weight,
		"bp":last_bp,
	})
	
}

func (h *PatientHandler) GetMedH(w http.ResponseWriter, r *http.Request){
	var rows []request.MedH
	h.Supabase.DB.From("patients").Select("medical_history").Execute(&rows)
	

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(rows))
	randomEntry := rows[randomIndex]
	fmt.Println(randomEntry)

	MedicalHistory := randomEntry.MedicalHistory

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"medical_history": MedicalHistory,
		
	})
	
}


func (h *PatientHandler) GetFMH(w http.ResponseWriter, r *http.Request){
	var rows []request.FMH
	h.Supabase.DB.From("patients").Select("family_medical_history").Execute(&rows)
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(rows))
	randomEntry := rows[randomIndex]
	//fmt.Println(randomEntry)

	FamilyMedicalHistory := randomEntry.FamilyMedicalHistory
	fmt.Println(FamilyMedicalHistory)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"family_medical_history": FamilyMedicalHistory,
		
	})
	
}


func (h *PatientHandler) GetPatientsData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	table := vars["table"]
	// stores raw data returned from Supabase query
	var rawRows []map[string]interface{}

	feature :="false"

	if table == "featured_patients"{
		feature = "true"
	}


	// grab all columns
	err := h.Supabase.DB.From("patients").Select("id,name,date_of_birth,gender,medical_condition,medical_history,patient_message").Eq("featured_patient",feature).Execute(&rawRows)
	if err != nil {
		http.Error(w, "Patients not found", http.StatusNotFound)
		return
	}
	fmt.Println(rawRows)

	

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rawRows)
}


//Modify data from the database
func(h *PatientHandler) UpdateData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Data Updated")
	var request request.Modify
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("UpdatePatient: failed to read request body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(bodyBytes, &request); err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}

	updateData := map[string]interface{}{}

	if request.MedicalHistory != ""{
		updateData["medical_history"] = request.MedicalHistory
	}
	if request.MedicalCondition != ""{
		updateData["medical_condition"] = request.MedicalCondition
	}
	if request.Message != ""{
		updateData["patient_message"] = request.Message
	}

	if err := h.Supabase.DB.From("patients").Update(updateData).Eq("id", request.PatientID).Execute(nil); err != nil {
		fmt.Println(err)
		http.Error(w, "Could not update patient", http.StatusInternalServerError)
		return
	}
}