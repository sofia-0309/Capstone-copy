package tasks

import (
	"net/http"

	model "gitlab.msu.edu/team-corewell-2025/models"
)

type TaskService interface {
	GenerateTasks(numQuestions int, numResults int, numPrescriptions int, generate_question bool, studentId string) error
	GetFullTasks(tasks []model.Task) ([]interface{}, error)
	GetTaskByID(w http.ResponseWriter, r *http.Request)
	GetTasksByStudentID(w http.ResponseWriter, r *http.Request)
	CompleteTask(w http.ResponseWriter, r *http.Request)
	GetTasksByWeekAndDay(w http.ResponseWriter, r *http.Request)
	GenerateTasksHTMLWrapper(w http.ResponseWriter, r *http.Request)
	GenerateInitialTasksHandler(w http.ResponseWriter, r *http.Request)
	GenerateNewTasks(w http.ResponseWriter, r *http.Request)
	GetTaskCalendar(w http.ResponseWriter, r *http.Request)
}
