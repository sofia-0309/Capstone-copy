package staff_tasks

import "net/http"

type StaffTaskService interface {
	GenerateStaffTasks(numStaffMessages int, studentId string) error
	GetStaffTasksByStudentID(w http.ResponseWriter, r *http.Request)
	GetStaffTaskByID(w http.ResponseWriter, r *http.Request)
	CompleteStaffTask(w http.ResponseWriter, r *http.Request)
	GenerateInitialStaffTasksHandler(w http.ResponseWriter, r *http.Request)
}
