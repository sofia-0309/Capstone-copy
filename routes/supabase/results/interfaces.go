package results

import "net/http"

type ResultService interface {
	GetResults(w http.ResponseWriter, r *http.Request)
	GetResultByID(w http.ResponseWriter, r *http.Request)
	GetResultsByPatientID(w http.ResponseWriter, r *http.Request)
	GetMultipleResultsByID(w http.ResponseWriter, r *http.Request)
}