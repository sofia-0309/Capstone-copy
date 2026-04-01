package scheduler

import (
	"encoding/json"
	"net/http"

	supabase "github.com/nedpals/supabase-go"
)
// handler struct that holds supabase client
type SchedulerHandler struct {
	Supabase *supabase.Client
}
// request structure
type SchedulerVisitRequest struct {
	PatientID string `json:"patient_id"`
	VisitDate string `json:"visit_date"`
	VisitTime string `json:"visit_time"`
	Notes string `json:"notes"`
}
// main function that runs when endpoint is hit
func (h *SchedulerHandler) CreateVisit(w http.ResponseWriter, r *http.Request) {
	header := w.Header()

	met := r.Method
	if met != "POST" {
		header.Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)

		resp := map[string]string{}
		resp["error"] = "method not allowed"

		enc := json.NewEncoder(w)
		enc.Encode(resp)
		return
	}
	// create the obj
	var req SchedulerVisitRequest

	// decode json step by step
	dec := json.NewDecoder(r.Body)
	decError := dec.Decode(&req)

	if decError != nil {
		header.Set("Content-Type", "application/json")

		statCode := http.StatusBadRequest
		w.WriteHeader(statCode)

		errMess := "invalid request"
		resp := map[string]string{}
		resp["error"] = errMess

		enc := json.NewEncoder(w)
		enc.Encode(resp)

		return
	}

	// put the fields in variables
	patientID := req.PatientID
	visitDate := req.VisitDate
	visitTime := req.VisitTime
	notes := req.Notes

	// check for flag fields

	flag := false
	if patientID == "" {
		flag = true
	}
	if visitDate == "" {
		flag = true
	}
	if visitTime == "" {
		flag = true
	}





	if flag == true {
		header.Set("Content-Type", "application/json")
		statusCode := http.StatusBadRequest
		w.WriteHeader(statusCode)

		errMes := "missing required fields"
		response := map[string]string{}
		response["error"] = errMes

		enc := json.NewEncoder(w)
		enc.Encode(response)

		return
	}



	// build putInVisit step by step 
	putInVisit := map[string]interface{}{}
	putInVisit["patient_id"] = patientID
	putInVisit["visit_date"] = visitDate
	putInVisit["visit_time"] = visitTime
	putInVisit["notes"] = notes



	// db steps 
	db := h.Supabase.DB
	table := db.From("scheduler")
	insertQuery := table.Insert(putInVisit)
	insertErr := insertQuery.Execute(nil)


	if insertErr != nil {
		header.Set("Content-Type", "application/json")
		statCod := http.StatusInternalServerError
		w.WriteHeader(statCod)

		errMes := "failed to put in visit"
		resp := map[string]string{}
		resp["error"] = errMes

		enc := json.NewEncoder(w)
		enc.Encode(resp)
		return
	}





	// success resp
	header.Set("Content-Type", "application/json")
	passedStat := "ok"
	response := map[string]string{}
	response["status"] = passedStat

	encoder := json.NewEncoder(w)
	encoder.Encode(response)
}