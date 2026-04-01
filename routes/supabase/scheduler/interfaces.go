package scheduler

import "net/http"

type SchedulerService interface {
	CreateVisit(w http.ResponseWriter, r *http.Request)
}
