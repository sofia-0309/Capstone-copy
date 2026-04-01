package staff

import "net/http"

type StaffService interface {
	GetAllStaff(w http.ResponseWriter, r *http.Request)
	GetStaffByID(w http.ResponseWriter, r *http.Request)
}
