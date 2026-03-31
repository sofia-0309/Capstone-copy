package staff_tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
)

type StaffTaskHandler struct {
	Supabase *supabase.Client
}

// GenerateUniqueIndices generates unique random indices (similar to tasks/routes.go)
func GenerateUniqueIndices(count, max int) []int {
	if count > max {
		count = max
	}
	indices := make([]int, count)
	used := make(map[int]bool)
	for i := 0; i < count; i++ {
		for {
			idx := rand.IntN(max)
			if !used[idx] {
				indices[i] = idx
				used[idx] = true
				break
			}
		}
	}
	return indices
}

// GenerateStaffTasks generates staff message tasks for a student
// Similar to GenerateTasks for patient messages
func (h *StaffTaskHandler) GenerateStaffTasks(numStaffMessages int, studentId string) error {
	var students []model.User
	var err error

	if studentId != "" {
		var student []model.User
		err = h.Supabase.DB.From("users").Select("*").Eq("id", studentId).Execute(&student)
		if err != nil || len(student) == 0 {
			fmt.Println("Student not found")
			return err
		}
		students = student
	} else {
		err := h.Supabase.DB.From("users").Select("*").Eq("isAdmin", "FALSE").Execute(&students)
		if err != nil || len(students) == 0 {
			fmt.Println("No students found")
			return err
		}
	}

	// Get all staff from the database
	var allStaff []model.Staff
	err = h.Supabase.DB.From("staff").Select("*").Execute(&allStaff)
	if err != nil || len(allStaff) == 0 {
		fmt.Println("No staff found")
		return err
	}

	createdAt := time.Now()

	for _, student := range students {
		// Randomly select staff members
		numToSelect := numStaffMessages
		if len(allStaff) < numStaffMessages {
			numToSelect = len(allStaff)
		}

		// Generate unique random indices
		randomIndices := GenerateUniqueIndices(numToSelect, len(allStaff))

		// Create staff tasks
		var staffTasks []model.StaffTask
		for _, idx := range randomIndices {
			staff := allStaff[idx]
			staffQuestion := staff.StaffMessage
			staffTask := model.StaffTask{
				StaffId:         staff.Id,
				UserId:          student.Id,
				Completed:       false,
				CreatedAt:       &createdAt,
				StudentResponse: nil,
				LLMResponse:     nil,
				LLMFeedback:     nil,
				StaffQuestion:   &staffQuestion,
			}
			staffTasks = append(staffTasks, staffTask)
		}

		err = h.Supabase.DB.From("staff_tasks").Insert(staffTasks).Execute(nil)
		if err != nil {
			fmt.Printf("Failed to insert staff tasks: %v\n", err)
			return err
		}
		fmt.Printf("Successfully created %d staff tasks for student %s\n", len(staffTasks), student.Id.String())
	}

	return nil
}

// GetStaffTasksByStudentID gets all staff tasks for a student
func (h *StaffTaskHandler) GetStaffTasksByStudentID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	studentId := vars["student_id"]

	type StaffTaskRequest struct {
		GetIncompleteTasks *bool `json:"get_incomplete_tasks,omitempty"`
		GetCompleteTasks   *bool `json:"get_complete_tasks,omitempty"`
	}

	var req StaffTaskRequest
	bodyBytes, _ := io.ReadAll(r.Body)
	json.Unmarshal(bodyBytes, &req)

	var tasks []model.StaffTask

	if req.GetCompleteTasks != nil && *req.GetCompleteTasks {
		var queryOutput []model.StaffTask
		err := h.Supabase.DB.From("staff_tasks").Select("*").Eq("user_id", studentId).Eq("completed", "TRUE").Execute(&queryOutput)
		if err == nil {
			tasks = append(tasks, queryOutput...)
		}
	}

	if req.GetIncompleteTasks != nil && *req.GetIncompleteTasks {
		var queryOutput []model.StaffTask
		err := h.Supabase.DB.From("staff_tasks").Select("*").Eq("user_id", studentId).Eq("completed", "FALSE").Execute(&queryOutput)
		if err == nil {
			tasks = append(tasks, queryOutput...)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GetStaffTaskByID gets a single staff task by ID
func (h *StaffTaskHandler) GetStaffTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	taskId := vars["task_id"]

	var task []model.StaffTask
	err := h.Supabase.DB.From("staff_tasks").Select("*").Eq("id", taskId).Execute(&task)

	if err != nil || len(task) == 0 {
		http.Error(w, "Staff task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task[0])
}

// CompleteStaffTask marks a staff task as completed
func (h *StaffTaskHandler) CompleteStaffTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	taskId := vars["task_id"]

	// Find the task
	var task []model.StaffTask
	err := h.Supabase.DB.From("staff_tasks").Select("*").Eq("id", taskId).Execute(&task)
	if err != nil || len(task) == 0 {
		http.Error(w, "Staff task not found", http.StatusNotFound)
		return
	}

	type TaskCompleteRequest struct {
		StudentResponse string `json:"student_response"`
		LLMResponse     string `json:"llm_response"`
		LLMFeedback     string `json:"llm_feedback"`
	}

	var taskCompleteRequest TaskCompleteRequest
	bodyBytes, _ := io.ReadAll(r.Body)
	err = json.Unmarshal(bodyBytes, &taskCompleteRequest)
	if err != nil {
		http.Error(w, "Cannot unmarshal task completion request", http.StatusBadRequest)
		return
	}

	// Update the task
	updateData := map[string]interface{}{
		"completed":        true,
		"student_response": taskCompleteRequest.StudentResponse,
		"llm_response":     taskCompleteRequest.LLMResponse,
		"llm_feedback":     taskCompleteRequest.LLMFeedback,
		"completed_at":     time.Now(),
	}

	err = h.Supabase.DB.From("staff_tasks").Update(updateData).Eq("id", taskId).Execute(nil)
	if err != nil {
		http.Error(w, "Failed to update staff task", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Staff task completed"))
}

// GenerateInitialStaffTasksHandler generates initial staff tasks for a new student
func (h *StaffTaskHandler) GenerateInitialStaffTasksHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentId := vars["student_id"]

	// Generate 10 staff message tasks (adjust as needed)
	err := h.GenerateStaffTasks(10, studentId)
	if err != nil {
		http.Error(w, "Failed to generate staff tasks for new student", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Staff tasks generated successfully!"))
}
