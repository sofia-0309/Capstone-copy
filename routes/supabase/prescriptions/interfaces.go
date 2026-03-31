package prescriptions

import "net/http"

type PrescriptionService interface {
	GetPrescriptions(w http.ResponseWriter, r *http.Request)
	GetPrescriptionByID(w http.ResponseWriter, r *http.Request)
	GetPrescriptionsByPatientID(w http.ResponseWriter, r *http.Request)
	GetMultiplePrescriptionsByID(w http.ResponseWriter, r *http.Request)
}