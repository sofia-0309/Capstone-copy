package reports

import "net/http"

type ReportService interface {
	GenerateProgressReport(w http.ResponseWriter, r *http.Request)
}
