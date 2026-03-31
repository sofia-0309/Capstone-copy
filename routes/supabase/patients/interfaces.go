package patients

import "net/http"

type PatientService interface {
	GetPatients(w http.ResponseWriter, r *http.Request)
	GetPatientByID(w http.ResponseWriter, r *http.Request)
	GetFlaggedPatients(w http.ResponseWriter, r *http.Request)
	AddFlaggedPatient(w http.ResponseWriter, r *http.Request)
	RemoveFlaggedPatient(w http.ResponseWriter, r *http.Request)
	KeepPatient(w http.ResponseWriter, r *http.Request)
	UpdateFlaggedPatientByID(w http.ResponseWriter, r *http.Request)
	GetMultiplePatientsByID(w http.ResponseWriter, r *http.Request)
	AddNewPatient(w http.ResponseWriter, r *http.Request)
	GetRandomName(w http.ResponseWriter, r *http.Request)
	GetDemo(w http.ResponseWriter, r *http.Request)
	GetVitals(w http.ResponseWriter, r *http.Request)
	GetMedH(w http.ResponseWriter, r *http.Request)
	GetFMH(w http.ResponseWriter, r *http.Request)
	GetPatientsData(w http.ResponseWriter, r *http.Request)
	UpdateData(w http.ResponseWriter, r *http.Request)
}