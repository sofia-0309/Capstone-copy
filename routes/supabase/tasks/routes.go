package tasks

import (
	"encoding/json"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
	request "gitlab.msu.edu/team-corewell-2025/routes/supabase"
	staffTasks "gitlab.msu.edu/team-corewell-2025/routes/supabase/staff_tasks"
)

// getLLMURL returns the Flask LLM service URL from environment or defaults to local
func getLLMURL() string {
	url := os.Getenv("FLASK_EXPLAIN_URL")
	if url == "" {
		return "http://127.0.0.1:5001/api/explain-request" // default for local dev
	}
	return url
}

// Mapping of Subjects selected during signup to actual Task Types
var SimpleTag = map[string][]string{
	"Pediatric Cases":                    {"Child", "Teen"},
	"Behavioral Health":                  {"Behavioral Health"},
	"Nervous System":                     {"Nervous System"},
	"Skin and Subcutaneous Tissue":       {"Dermatologic"},
	"Musculoskeletal System":             {"Musculoskeletal"},
	"Cardiovascular System":              {"Cardiovascular"},
	"Respiratory System":                 {"Respiratory"},
	"Gastrointestinal System":            {"Gastrointestinal"},
	"Renal & Urinary Systems":            {"Urinary"},
	"Pregnancy, Childbirth & Puerperium": {"Pregnancy", "Female Reproductive System"},
	"Female Reproductive System":         {"Female Reproductive System"},
	"Male Reproductive System":           {"Male Reproductive System"},
	"Endocrine System":                   {"Endocrine"},
	"Multi-System Processes & Disorders": {"Multi-System"},
	"Biostatistics & Epidemiology":       {"General Principles"},
	"General Principles":                 {"General Principles"},
}

func ConvertTags(studentTag string) []string {
	if mapped, ok := SimpleTag[studentTag]; ok && len(mapped) > 0 {
		return mapped
	}

	return []string{studentTag}
}

type TaskHandler struct {
	Supabase *supabase.Client
}

/*
*
  - GenerateUniqueIndices generates a list of unique random numbers
  - Helper function to generate a list of random unique numbers for indexing into patients list
  - @param count int		Number of unique numbers to generate. If this number exceeds the max range,
    the function will stop tracking unique values and continue generating
  - @param max int		Generates numbers from range[0, max), excluding max
  - @return []int		List of unique random numbers
*/
func GenerateUniqueIndices(count, max int) []int {
	uniqueNumbers := make(map[int]bool) // Set to track generated numbers
	result := make([]int, 0, count)     // List to store unique numbers

	for len(result) < count {
		if len(uniqueNumbers) >= max {
			// If we need to generate more numbers than the range, reset tracking and continue appending
			// Mostly to avoid errors, shouldn't happen in practice because there's so many patients
			uniqueNumbers = make(map[int]bool)
		}

		num := rand.IntN(max) // Generate a number
		if !uniqueNumbers[num] {
			uniqueNumbers[num] = true // Mark as seen
			result = append(result, num)
		}
	}

	return result
}

// The only thing this function does is extract the number of tasks to generate from the request body
// and then calls the GenerateTasks function
func (h *TaskHandler) GenerateTasksHTMLWrapper(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Generating tasks")
	// Get the number of tasks to generate from request body
	// Example json body:
	// {
	//     "patient_task_count": 3,
	//     "lab_result_task_count": 0,
	//     "prescription_task_count": 0,
	// }
	var taskCreateRequest request.TaskCreateRequest
	bodyBytes, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(bodyBytes, &taskCreateRequest)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error parsing task request body", http.StatusNotFound)
		return
	}

	// Run the task generation function
	// Removed generating patient question for now
	err = h.GenerateTasks(taskCreateRequest.PatientTaskCount, taskCreateRequest.LabResultTaskCount, taskCreateRequest.PrescriptionTaskCount, false, "")
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error generating tasks", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Tasks generated"))

}

// Grabs Student's "Improvement Area" Tags from their profile page
func (h *TaskHandler) GetStudentTags(studentId string) ([]string, error) {
	var profiles []struct {
		ImprovementAreas []string `json:"improvementAreas"`
	}

	err := h.Supabase.DB.From("users").
		Select("improvementAreas").
		Eq("id", studentId).
		Execute(&profiles)

	fmt.Printf("DEBUG: GetStudentTags for %s - Error: %v, Profiles found: %d\n", studentId, err, len(profiles))
	if len(profiles) > 0 {
		fmt.Printf("DEBUG: ImprovementAreas: %v\n", profiles[0].ImprovementAreas)
	}

	// Error has occurred or there are no student profiles
	if err != nil || len(profiles) == 0 {
		return []string{}, err
	}

	return profiles[0].ImprovementAreas, nil
}

// Searches for Patients with specific tags
func (h *TaskHandler) GetPatientsWithTag(tags []string, needed int) ([]model.Patient, error) {
	var allPatients []model.Patient
	err := h.Supabase.DB.From("patients").
		Select("*").
		Execute(&allPatients)
	if err != nil {
		return nil, fmt.Errorf("error fetching patients: %w", err)
	}

	tagMap := make(map[string][]model.Patient)
	for _, p := range allPatients {
		for _, t := range p.ChiefConcern.ChiefComplaintTags {
			t = strings.ToLower(strings.TrimSpace(t))
			tagMap[t] = append(tagMap[t], p)
		}
	}

	var matched []model.Patient
	seen := make(map[uuid.UUID]bool)

	for _, tag := range tags {
		lowerTag := strings.ToLower(strings.TrimSpace(tag))
		for _, p := range tagMap[lowerTag] {
			if !seen[p.Id] {
				matched = append(matched, p)
				seen[p.Id] = true
				if len(matched) >= needed {
					return matched, nil
				}
			}
		}
	}

	return matched, nil
}

// Handles Task Generation for Students when Account is Created
// 28 of each task for students, 1/4 of tasks will be using tags if marked during account creation
func (h *TaskHandler) GenerateInitialTasksHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentId := vars["student_id"]

	err := h.GenerateTasks(28, 28, 28, false, studentId)
	if err != nil {
		http.Error(w, "Failed to generate tasks for new student", http.StatusInternalServerError)
		return
	}

	// Generate staff tasks (10 staff message tasks)
	staffTaskHandler := &staffTasks.StaffTaskHandler{Supabase: h.Supabase}
	err = staffTaskHandler.GenerateStaffTasks(10, studentId)
	if err != nil {
		fmt.Printf("Warning: Failed to generate staff tasks: %v\n", err)
		// Don't fail the whole request, just log the error
	}

	type achievement struct {
		UserId uuid.UUID `json:"user_id"`
	}
	//insert achievement table to supabase
	id, _ := uuid.Parse(studentId)
	err = h.Supabase.DB.From("achievements").Insert(achievement{UserId: id}).Execute(nil)
	if err != nil {
		fmt.Println(err)
	}
	w.Write([]byte("Tasks generated successfully!"))
}

// Does the actual generation of tasks
// This function is called by the API endpoint
func (h *TaskHandler) GenerateTasks(numQuestions int, numResults int, numPrescriptions int, generate_question bool, studentId string) error {
	var students []model.User
	var err error
	errLatest := error(nil) // Keeps track of the latest error, if any

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

	// Gets all patients from the database
	var patients []model.Patient
	err = h.Supabase.DB.From("patients").Select("*").Execute(&patients)
	fmt.Printf("DEBUG: Patients query error: %v\n", err)
	fmt.Printf("DEBUG: Number of patients found: %d\n", len(patients))
	if err != nil || len(patients) == 0 {
		fmt.Println("No patients found")
		return err
	}

	createdAt := time.Now() // Timestamp for task creation
	for _, student := range students {
		studentTags, err := h.GetStudentTags(student.Id.String())
		if err != nil {
			fmt.Printf("Could Not fetch tags for student, patients will be randomly assigned")
			studentTags = []string{}
		}

		var taggedPatients []model.Patient

		if len(studentTags) > 0 {
			fmt.Printf("Student has tags: %v\n", studentTags)

			for _, studentTag := range studentTags {
				mappedTags := ConvertTags(studentTag)
				fmt.Printf("Searching tags for improvement area '%s': %v\n", studentTag, mappedTags)

				matchingPatients, errFind := h.GetPatientsWithTag(mappedTags, numQuestions/4-len(taggedPatients))
				if errFind != nil {
					fmt.Printf("Error finding patients for %s: %v\n", studentTag, errFind)
					continue
				}

				taggedPatients = append(taggedPatients, matchingPatients...)

				if len(taggedPatients) >= numQuestions/4 {
					fmt.Printf("Found enough tagged patients: %d\n", len(taggedPatients))
					break
				}
			}

			if len(taggedPatients) == 0 {
				fmt.Println("No patients found for any of the student's improvement areas")
			}
		}

		availablePatients := make([]model.Patient, 0)
		for _, p := range patients {
			skip := false
			for _, tp := range taggedPatients {
				if p.Id == tp.Id {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			availablePatients = append(availablePatients, p)
		}

		numRandom := numQuestions - len(taggedPatients)
		if numRandom < 0 {
			numRandom = 0
		}

		totalRandomPatients := numRandom + numResults + numPrescriptions

		random_indices := GenerateUniqueIndices(totalRandomPatients, len(availablePatients))
		random_index := 0

		for i := 0; i < numQuestions; i++ {
			// Generate a patient question task
			var patient_uuid string
			var fullPatient model.Patient

			if i < len(taggedPatients) {
				fullPatient = taggedPatients[i]
				patient_uuid = fullPatient.Id.String()
			} else {
				fullPatient = availablePatients[random_indices[random_index]]
				patient_uuid = fullPatient.Id.String()
				random_index++
			}

			var question string
			if generate_question {
				// Retrieve the patient record.
				// Retrieve prescriptions for this patient.
				var prescriptions []model.Prescription
				err = h.Supabase.DB.From("prescriptions").Select("*,patient:patients(name)").Eq("patient_id", patient_uuid).Execute(&prescriptions)
				if err != nil {
					// If an error occurs, you might choose to continue with an empty list.
					prescriptions = []model.Prescription{}
				}

				// Retrieve test results for this patient.
				var results []model.Result
				err = h.Supabase.DB.From("results").Select("*,patient:patients(name)").Eq("patient_id", patient_uuid).Execute(&results)
				if err != nil {
					results = []model.Result{}
				}

				// Combine the patient, prescriptions, and results into one object.
				combinedData := map[string]interface{}{
					"patient":       fullPatient,
					"prescriptions": prescriptions,
					"results":       results,
				}

				// Marshal the entire combined object into a pretty JSON string.
				combinedJSON, err := json.MarshalIndent(combinedData, "", "  ")
				if err != nil {
					fmt.Println("Error encoding combined patient data")
					return err
				}

				// Build a prompt that includes all of the data.
				prompt := fmt.Sprintf("Patient Data:\n%s\n Using this data, generate a new potential question that the patient may ask their doctor. The question should be about recent symptoms the patient has been experiencing. Respond with only the message and nothing else. Do not include the quotation marks with the message.", string(combinedJSON))

				// Create the LLM request payload.
				llmRequest := map[string]string{
					"message": prompt,
				}

				reqBody, err := json.Marshal(llmRequest)
				if err != nil {
					fmt.Println("Error encoding LLM request")
					return err
				}

				// Send the request to the LLM microservice.
				response, err := http.Post(getLLMURL(), "application/json", bytes.NewBuffer(reqBody))
				if err != nil {
					fmt.Println("Error communicating with LLM")
					return err
				}
				defer response.Body.Close()

				body, err := io.ReadAll(response.Body)
				if err != nil {
					fmt.Println("Error reading LLM response")
					fmt.Println(err) // skip this task instead of returning
					errLatest = err
				}

				question_body := map[string]string{}
				err = json.Unmarshal(body, &question_body)
				if err != nil {
					fmt.Println("Error parsing LLM response")
					fmt.Println(err) // skip this task instead of returning
					errLatest = err
				}

				question = strings.Trim(question_body["completion"], "\"")
			} else {
				// don't generate question, just use current patient message in supabase
				question = fullPatient.PatientMessage
			}

			patient_task := model.PatientTask{
				Task: model.Task{
					PatientId:       fullPatient.Id, // Little convoluted but it keeps track of the index from other loops
					UserId:          student.Id,
					TaskType:        model.PatientQuestionTaskType,
					Completed:       false,
					CreatedAt:       &createdAt,
					StudentResponse: nil, // won't be filled in until student responds
					LLMResponse:     nil, // won't be filled in until LLM provides response
					LLMFeedback:     nil, // won't be filled in until LLM provides feedback
				},
				PatientQuestion: &question, // task is generated with patient question
			}
			err = h.Supabase.DB.From("tasks").Insert(patient_task).Execute(nil)
			if err != nil {
				fmt.Println("Failed to insert patient question task")
				return err
			}
		}

		availablePatientIds := make([]string, len(availablePatients))
		for i, p := range availablePatients {
			availablePatientIds[i] = p.Id.String()
		}

		var allResults []model.Result
		err = h.Supabase.DB.From("results").
			Select("*").
			In("patient_id", availablePatientIds).
			Execute(&allResults)

		fmt.Printf("DEBUG: Lab results query error: %v\n", err)
		fmt.Printf("DEBUG: Number of lab results found: %d\n", len(allResults))

		if err != nil || len(allResults) == 0 {
			fmt.Println("No lab results available for any patients")
			// Don't return error, just skip result tasks
		} else {

			startIndex := rand.IntN(len(allResults))

			for i := 0; i < numResults; i++ {
				// Generate a lab result task

				currentIndex := (startIndex + i) % len(allResults)
				selectedResult := allResults[currentIndex]

				result_task := model.ResultTask{
					Task: model.Task{
						PatientId:       selectedResult.Patient_id, // Little convoluted but it keeps track of the index from other loops
						UserId:          student.Id,
						TaskType:        model.LabResultTaskType,
						Completed:       false,
						CreatedAt:       &createdAt,
						StudentResponse: nil, // won't be filled in until student responds
						LLMResponse:     nil, // won't be filled in until LLM provides response
						LLMFeedback:     nil, // won't be filled in until LLM provides feedback
					},
					ResultId: selectedResult.ID,
				}
				err = h.Supabase.DB.From("tasks").Insert(result_task).Execute(nil)
				if err != nil {
					fmt.Println("Failed to insert lab result task")
					return err
				}
			}
		}

		var allPrescriptions []model.Prescription
		err = h.Supabase.DB.From("prescriptions").
			Select("*").
			In("patient_id", availablePatientIds).
			Execute(&allPrescriptions)

		fmt.Printf("DEBUG: Prescriptions query error: %v\n", err)
		fmt.Printf("DEBUG: Number of prescriptions found: %d\n", len(allPrescriptions))

		if err != nil || len(allPrescriptions) == 0 {
			fmt.Println("No prescriptions available for any patients")
		} else {

			startIndex := rand.IntN(len(allPrescriptions))

			for i := 0; i < numPrescriptions; i++ {
				// Generate a prescription task

				currentIndex := (startIndex + i) % len(allPrescriptions)
				selectedPrescription := allPrescriptions[currentIndex]

				prescription_task := model.PrescriptionTask{
					Task: model.Task{
						PatientId:       selectedPrescription.Patient_id, // Little convoluted but it keeps track of the index from other loops
						UserId:          student.Id,
						TaskType:        model.PrescriptionTaskType,
						Completed:       false,
						CreatedAt:       &createdAt,
						StudentResponse: nil, // won't be filled in until student responds
						LLMResponse:     nil, // won't be filled in until LLM provides response
						LLMFeedback:     nil, // won't be filled in until LLM provides feedback
					},
					PrescriptionId: selectedPrescription.ID,
				}
				err = h.Supabase.DB.From("tasks").Insert(prescription_task).Execute(nil)
				if err != nil {
					fmt.Println("Failed to insert prescription task")
					return err
				}
			}
		}
	}

	if errLatest != nil {
		fmt.Println("One or more tasks failed to generate, but continuing")
		return errLatest // return the error if one or more tasks failed to generate
	} else {
		fmt.Println("All tasks generated successfully")
		return nil // no errors
	}
}

// Helper function for getting the entire task (including specific task type parts)
// Takes a list of tasks (output from Supabase query), returns the list of full tasks
// Note: Full tasks only used in detailed task view (GetTaskByID, GetTasksByStudentID), not in overall task list view (GetTasksByWeekAndDay)
func (h *TaskHandler) GetFullTasks(tasks []model.Task) ([]interface{}, error) {
	fullTasks := make([]interface{}, 0)
	for _, task := range tasks {
		switch task.TaskType {
		case "patient_question":
			var patientTask []model.PatientTask
			var patient []model.Patient
			err := h.Supabase.DB.From("tasks").Select("*").Eq("id", task.Id.String()).Execute(&patientTask)
			if err == nil {
				h.Supabase.DB.From("patients").Select("*").Eq("id", patientTask[0].PatientId.String()).Execute(&patient)
				patientTask[0].PatientQuestion = &patient[0].PatientMessage
				fullTasks = append(fullTasks, patientTask[0])
			} else {
				fmt.Println(err)
				return nil, err
			}
		case "lab_result":
			var labResult []model.ResultTask
			err := h.Supabase.DB.From("tasks").Select("*").Eq("id", task.Id.String()).Execute(&labResult)
			if err == nil {
				fullTasks = append(fullTasks, labResult[0])
			} else {
				fmt.Println(err)
				return nil, err
			}
		case "prescription":
			var prescription []model.PrescriptionTask
			err := h.Supabase.DB.From("tasks").Select("*").Eq("id", task.Id.String()).Execute(&prescription)
			if err == nil {
				fullTasks = append(fullTasks, prescription[0])
			} else {
				fmt.Println(err)
				return nil, err
			}
		}
	}
	return fullTasks, nil
}

// Helper function for getting rid of null values from an interface slice
// Useful for cleaning up the marshaled output from Supabase query --> interface (instead of struct)
// Currently not being used, I'm adding it just in case
func removeNullsFromSlice(data []interface{}) []interface{} {
	cleanedSlice := make([]interface{}, 0)

	for _, item := range data {
		if itemMap, ok := item.(map[string]interface{}); ok {
			cleanedMap := make(map[string]interface{})
			for key, value := range itemMap {
				if value != nil { // Only add non-nil values
					cleanedMap[key] = value
				}
			}
			cleanedSlice = append(cleanedSlice, cleanedMap)
		} else {
			// If it's not a map[string]interface{}, just add it as is
			cleanedSlice = append(cleanedSlice, item)
		}
	}
	return cleanedSlice
}

// Gets a singular task by ID
// The URL contains the task ID
// Contains the full task (including specific task type parts)
func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["task_id"]
	var task []model.Task
	err := h.Supabase.DB.From("tasks").Select("*").Eq("id", id).Execute(&task)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Gets the full task
	fullTask, err := h.GetFullTasks(task)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Full Task not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	//fullTask[0].PatientQuestion = "hel"
	json.NewEncoder(w).Encode(fullTask[0])

}

// This function gets all the tasks for a student
// The URL contains the student ID
// The request body tells the function which type of tasks to get (completed or incomplete)
func (h *TaskHandler) GetTasksByStudentID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["student_id"]

	//necessary to enable cors
	(w).Header().Set("Access-Control-Allow-Origin", "*")
	(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	// Get the request body
	// Example request body:
	// {
	// 	"get_incomplete_tasks": true,
	// 	"get_complete_tasks": false
	// }
	var taskGetRequest request.TaskGetRequest
	bodyBytes, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(bodyBytes, &taskGetRequest)
	if err != nil {
		http.Error(w, "Cannot Unmarshal task request from request", http.StatusBadRequest)
		return
	}
	var tasks []model.FullTask // Holds the tasks to be returned
	if *taskGetRequest.GetCompleteTasks {
		var queryOutput []model.FullTask
		err = h.Supabase.DB.From("tasks").Select("*").Eq("user_id", id).Eq("completed", "TRUE").Execute(&queryOutput)
		if err != nil {
			http.Error(w, "No completed tasks found", http.StatusNotFound)
		}
		tasks = append(tasks, queryOutput...) // Adds query output to list of tasks
	}
	if *taskGetRequest.GetIncompleteTasks {
		var queryOutput []model.FullTask
		err = h.Supabase.DB.From("tasks").Select("*").Eq("user_id", id).Eq("completed", "FALSE").Execute(&queryOutput)
		if err != nil {
			http.Error(w, "No incomplete tasks found", http.StatusNotFound)
		}
		tasks = append(tasks, queryOutput...) // Adds query output to list of tasks
	}

	if taskGetRequest.AgeFilter != "" && taskGetRequest.AgeFilter != "all" {
		filteredTasks := make([]model.FullTask, 0)
		var list1 = []string{}
		for _, task := range tasks {
			list1 = append(list1, task.PatientId.String())
		}
		var patient []model.Patient
		if taskGetRequest.AgeFilter == "pediatric" {
			h.Supabase.DB.From("patients").Select("*").In("id", list1).Lte("age", "17").Execute(&patient)
		}
		if taskGetRequest.AgeFilter == "adult" {
			h.Supabase.DB.From("patients").Select("*").In("id", list1).Gt("age", "17").Execute(&patient)
		}
		for _, task := range tasks {
			for _, patients := range patient {
				if patients.Id.String() == task.PatientId.String() {
					filteredTasks = append(filteredTasks, task)
				}
			}
		}
		tasks = filteredTasks
	}

	if len(tasks) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]model.FullTask{})
		return
	}

	// Get the full tasks for the entire list
	//fullTasks, err := h.GetFullTasks(tasks)

	if err != nil {
		http.Error(w, "Failed to get full tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(tasks)
}

// Helper function to insert NULL into supabase when string is empty instead of empty string
func NilIfEmptyString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// CompleteTask marks a task as completed and fills in information for the task from the request body
// The URL contains the task ID
func (h *TaskHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["task_id"]

	// Finds the task
	var task []model.Task
	err := h.Supabase.DB.From("tasks").Select("*").Eq("id", id).Execute(&task)
	if err != nil || len(task) == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// All tasks have a student response and LLM feedback
	var taskCompleteRequest request.TaskCompleteRequest
	// Get the request body
	// Example request body:
	// {
	// 	"student_response": "The student's response to the task",
	//  "llm_response": "The LLM's sample response to the task",
	// 	"llm_feedback": "The LLM's feedback to the student response"
	// }
	bodyBytes, _ := io.ReadAll(r.Body)
	err = json.Unmarshal(bodyBytes, &taskCompleteRequest)
	if err != nil {
		http.Error(w, "Cannot Unmarshal task completion request from request", http.StatusBadRequest)
		return
	}

	// Update the fields for the task
	updateData := map[string]interface{}{
		"completed":        true,
		"student_response": taskCompleteRequest.StudentResponse,
		"llm_response":     taskCompleteRequest.LLMResponse,
		"llm_feedback":     taskCompleteRequest.LLMFeedback,
		"completed_at":     time.Now(),
	}

	// Update DB
	err = h.Supabase.DB.From("tasks").Update(updateData).Eq("id", id).Execute(nil)
	if err != nil {
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Task completed"))
}

// GetTasksByWeekAndDay gets all the tasks for a student and sorts them by week and day
// The URL contains the student ID
// Also includes the student's completion rate for each week
func (h *TaskHandler) GetTasksByWeekAndDay(w http.ResponseWriter, r *http.Request) {
	// Example output:
	// [
	//	{
	//	    "Week": 0,
	//	    "Days": [
	//	        {
	//	            "Day": 0,
	//	            "Tasks": [
	//	                {
	//	                    "id": "bd6acf0c-2e88-4c9f-bfe2-342919e538d6",
	//	                    "created_at": "2025-03-14T00:04:31.372774Z",
	//	                    "patient_id": "1f1d78f4-345b-48d7-9045-8baa6b6e070a",
	//	                    "user_id": "b66e3169-2335-48f8-a43f-e4730c053ad8",
	//	                    "task_type": "patient_question",
	//	                    "completed": false
	//	                },
	//	            ],
	//	            "CompletionRate": 0
	//			}
	//			],
	//			"CompletionRate": 0
	//		}
	// ]

	// Get the student ID and week number from the URL
	vars := mux.Vars(r)
	id := vars["student_id"]

	// Get all the tasks for the student
	//var tasks []model.Task
	type Task struct {
		Id          *uuid.UUID `json:"id,omitempty"`
		CreatedAt   time.Time  `json:"created_at,omitempty"` // Pointer to avoid default time
		PatientId   uuid.UUID  `json:"patient_id"`
		Completed   bool       `json:"completed"`
		CompletedAt time.Time  `json:"completed_at,omitempty"`
		Name        string     `json:"name"`
		TaskType    string     `json:"task_type"`
	}
	var tasks []Task
	idInterface := make(map[string]interface{})
	idInterface["student_id"] = id
	err := h.Supabase.DB.Rpc("get_tasks", idInterface).Execute(&tasks)
	//err := h.Supabase.DB.From("tasks").Select("*").Eq("user_id", id).Execute(&tasks)
	if err != nil || len(tasks) == 0 {
		msg := fmt.Sprintf("%v", err)
		fmt.Println(msg)
		http.Error(w, "No tasks found", http.StatusNotFound)
		return
	}

	// Gets the bounds (earliest and latest week) for counting by week
	startDate := time.Now() // Latest date (in theory)
	endDate := time.Time{}  // Earliest possible date
	for _, task := range tasks {
		if task.CreatedAt.Before(startDate) {
			startDate = task.CreatedAt
		}
		if task.CreatedAt.After(endDate) {
			endDate = task.CreatedAt
		}
	}

	// Struct to store info per day (tasks, completion rate)
	type DayPackage struct {
		Day            int     // Day number
		Tasks          []Task  // List of tasks for the day
		CompletionRate float64 // Percentage of completed tasks
	}

	// Struct to store info for each week
	type WeekPackage struct {
		Week           int          // Week number
		Days           []DayPackage // List of Days in the week
		CompletionRate float64      // Percentage of completed tasks
	}

	// List of weeks
	weekList := make([]WeekPackage, 0)
	numWeeks := int(endDate.Sub(startDate).Hours() / (24 * 7)) // Number of weeks between the two dates
	numWeeks++                                                 // Add one to include the last week
	numDays := int(endDate.Sub(startDate).Hours() / 24)        // Number of days between the two dates
	// fmt.Println(numWeeks)

	// Initialize the list of weeks with empty days
	for i := 0; i < numWeeks; i++ {
		// Add each week
		weekList = append(weekList, WeekPackage{Week: i, Days: []DayPackage{}, CompletionRate: 0})

		// Add each day
		if i < numWeeks-1 {
			// Full week
			for j := 0; j < 7; j++ {
				weekList[i].Days = append(weekList[i].Days, DayPackage{Day: j, Tasks: []Task{}, CompletionRate: 0})
			}
		} else {
			// Last week may not have all days, cutoff early if so
			for j := 0; j <= numDays%7; j++ {
				weekList[i].Days = append(weekList[i].Days, DayPackage{Day: j, Tasks: []Task{}, CompletionRate: 0})
			}
		}
	}

	// Loop through each task to categorize it by week
	for _, task := range tasks {
		// Calculate the number of weeks since the start date
		weekNumber := int(task.CreatedAt.Sub(startDate).Hours() / (24 * 7)) // Week difference
		dayNumber := int(task.CreatedAt.Sub(startDate).Hours()/24) % 7      // Day of the week (0-6)

		for i := range weekList {
			for j := range weekList[i].Days {
				if weekList[i].Week == weekNumber && j == dayNumber {
					weekList[i].Days[j].Tasks = append(weekList[i].Days[j].Tasks, task)
				}
			}
		}
	}

	// Calculate the completion rate for each week + day
	if len(weekList) > 0 {
		for i, week := range weekList {
			completedTasksWeekly := 0
			totalTasksWeekly := 0
			for j, day := range week.Days {
				if len(day.Tasks) > 0 {
					completedTasksDaily := 0
					for _, task := range day.Tasks {
						totalTasksWeekly++
						if task.Completed {
							completedTasksDaily++
							completedTasksWeekly++
						}
					}
					weekList[i].Days[j].CompletionRate = float64(completedTasksDaily) / float64(len(day.Tasks)) * 100
				} else {
					// Avoids errors in case the day does not have any tasks (should not be possible)
					weekList[i].Days[j].CompletionRate = 0
				}

			}
			weekList[i].CompletionRate = float64(completedTasksWeekly) / float64(totalTasksWeekly) * 100
			if totalTasksWeekly == 0 || math.IsNaN(weekList[i].CompletionRate) {
				weekList[i].CompletionRate = 0
			}
		}
	} else {
		// No tasks (meaning no weeks), so no completion rate
		// Shouldn't get to this point because we already checked for tasks
		http.Error(w, "No tasks found", http.StatusNotFound)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(weekList)
	if err != nil {
		msg := fmt.Sprintf("%v", err)
		fmt.Println(msg)
		http.Error(w, "Failed to encode response when getting tasks grouped chronologically", http.StatusInternalServerError)
	}
}

func (h *TaskHandler) GenerateNewTasks(w http.ResponseWriter, r *http.Request) {

	type NewTaskRequest struct {
		UserId    uuid.UUID `json:"user_id"`
		TaskCount int       `json:"task_count"`
		Tags      []string  `json:"tags"`
		TaskType  string    `json:"task_type"`
	}
	type patientTaskId struct {
		model.Patient
		PatientId uuid.UUID `json:"patient_id"`
	}
	type ResultPatientName struct {
		model.Result
		Patient struct {
			Name string `json:"name"`
		} `json:"patient"`
	}
	type PrescriptionPatientName struct {
		model.Prescription
		Patient struct {
			Name string `json:"name"`
		} `json:"patient"`
	}

	var newTask NewTaskRequest
	bodyBytes, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(bodyBytes, &newTask)
	if err != nil {
		fmt.Println(err)
		return
	}

	accurateTags := make([]string, 0)
	for i := 0; i < len(newTask.Tags); i++ {
		accurateTags = append(accurateTags, ConvertTags(newTask.Tags[i])...)
	}
	if len(accurateTags) == 0 {
		accurateTags = append(accurateTags, "notags")
	}

	if newTask.TaskType == "Patient Message" {
		var patient []patientTaskId
		err = h.Supabase.DB.From("patients").Select("*").Execute(&patient)
		if err != nil {
			fmt.Println(err)
			return
		}

		var relatedPatients []patientTaskId
		var unRelatedPatients []patientTaskId

		for i := 0; i < len(patient); i++ {
			var inserted = false
			for j := 0; j < len(accurateTags); j++ {
				if slices.Contains(patient[i].ChiefConcern.ChiefComplaintTags, accurateTags[j]) && !inserted {
					relatedPatients = slices.Insert(relatedPatients, rand.IntN(len(relatedPatients)+1), patient[i])
					inserted = true
					continue
				}
			}
			if inserted == false {
				unRelatedPatients = slices.Insert(unRelatedPatients, rand.IntN(len(unRelatedPatients)+1), patient[i])
			}
		}

		oldLength := len(relatedPatients)
		if len(relatedPatients) < newTask.TaskCount {
			for k := 0; k < (newTask.TaskCount - oldLength); k++ {
				relatedPatients = slices.Insert(relatedPatients, rand.IntN(len(relatedPatients)+1), unRelatedPatients[k])
			}
		} else {
			relatedPatients = relatedPatients[0:newTask.TaskCount]
		}

		createdAt := time.Now()
		var patient_task []model.PatientTask

		for k := 0; k < len(relatedPatients); k++ {
			patient_task = append(patient_task, model.PatientTask{
				Task: model.Task{
					PatientId:   relatedPatients[k].Id,
					UserId:      newTask.UserId,
					TaskType:    "patient_question",
					Completed:   false,
					CreatedAt:   &createdAt,
					LLMResponse: nil, // won't be filled in until LLM provides response
					LLMFeedback: nil, // won't be filled in until LLM provides feedback
				},
				PatientQuestion: &relatedPatients[k].PatientMessage,
			})
		}

		err = h.Supabase.DB.From("tasks").Insert(patient_task).Execute(&patient_task)

		for k := 0; k < len(patient_task); k++ {
			relatedPatients[k].PatientId = relatedPatients[k].Id
			relatedPatients[k].Id = *patient_task[k].Id
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(relatedPatients)
		return
	}

	var patient []model.Patient
	err = h.Supabase.DB.From("patients").Select("*").Execute(&patient)
	if err != nil {
		fmt.Println(err)
		return
	}

	var relatedPatients []model.Patient
	var unRelatedPatients []model.Patient

	for i := 0; i < len(patient); i++ {
		var inserted = false
		for j := 0; j < len(accurateTags); j++ {
			if slices.Contains(patient[i].ChiefConcern.ChiefComplaintTags, accurateTags[j]) && !inserted {
				relatedPatients = slices.Insert(relatedPatients, rand.IntN(len(relatedPatients)+1), patient[i])
				inserted = true
				continue
			}
		}
		if inserted == false {
			unRelatedPatients = slices.Insert(unRelatedPatients, rand.IntN(len(unRelatedPatients)+1), patient[i])
		}
	}

	oldLength := len(relatedPatients)
	if len(relatedPatients) < newTask.TaskCount {
		for k := 0; k < (newTask.TaskCount - oldLength); k++ {
			relatedPatients = slices.Insert(relatedPatients, rand.IntN(len(relatedPatients)+1), unRelatedPatients[k])
		}
	} else {
		relatedPatients = relatedPatients[0:newTask.TaskCount]
	}

	var relatedPatientsIds []string
	for k := 0; k < len(relatedPatients); k++ {
		relatedPatientsIds = append(relatedPatientsIds, relatedPatients[k].Id.String())
	}

	if newTask.TaskType == "Lab Result" {
		var result []ResultPatientName
		err = h.Supabase.DB.From("results").Select("*").Limit(newTask.TaskCount).In("patient_id", relatedPatientsIds).Execute(&result)
		if len(result) < newTask.TaskCount {
			var relatedPatientsIds []string
			var resultExtra []ResultPatientName
			for k := 0; k < (newTask.TaskCount - len(result)); k++ {
				relatedPatients = append(relatedPatients, unRelatedPatients[len(unRelatedPatients)-(k+1)])
				relatedPatientsIds = append(relatedPatientsIds, unRelatedPatients[len(unRelatedPatients)-(k+1)].Id.String())
				print(relatedPatientsIds[k])
				print(" ")
			}
			err = h.Supabase.DB.From("results").Select("*").Limit(newTask.TaskCount-len(result)).In("patient_id", relatedPatientsIds).Execute(&resultExtra)
			for k := 0; k < (len(resultExtra)); k++ {
				result = append(result, resultExtra[k])
			}
		}
		createdAt := time.Now()
		var result_task []model.ResultTask

		for k := 0; k < len(result); k++ {
			result_task = append(result_task, model.ResultTask{
				Task: model.Task{
					PatientId:   result[k].Patient_id,
					UserId:      newTask.UserId,
					TaskType:    "lab_result",
					Completed:   false,
					CreatedAt:   &createdAt,
					LLMResponse: nil, // won't be filled in until LLM provides response
					LLMFeedback: nil, // won't be filled in until LLM provides feedback
				},
				ResultId: result[k].ID,
			})
		}

		err = h.Supabase.DB.From("tasks").Insert(result_task).Execute(&result_task)
		for k := 0; k < len(result_task); k++ {
			for i := 0; i < len(relatedPatients); i++ {
				if result[k].Patient_id == relatedPatients[i].Id {
					result[k].Patient.Name = relatedPatients[i].Name
					break
				}
			}
			result[k].ID = *result_task[k].Id
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	if newTask.TaskType == "Prescription" {
		var prescription []PrescriptionPatientName
		err = h.Supabase.DB.From("prescriptions").Select("*").Limit(newTask.TaskCount).In("patient_id", relatedPatientsIds).Execute(&prescription)
		if len(prescription) < newTask.TaskCount {
			var relatedPatientsIds []string
			var prescriptionExtra []PrescriptionPatientName
			for k := 0; k < (newTask.TaskCount - len(prescription)); k++ {
				relatedPatients = append(relatedPatients, unRelatedPatients[len(unRelatedPatients)-(k+1)])
				relatedPatientsIds = append(relatedPatientsIds, unRelatedPatients[len(unRelatedPatients)-(k+1)].Id.String())
				print(relatedPatientsIds[k])
				print(" ")
			}
			err = h.Supabase.DB.From("prescriptions").Select("*").Limit(newTask.TaskCount-len(prescription)).In("patient_id", relatedPatientsIds).Execute(&prescriptionExtra)
			for k := 0; k < (len(prescriptionExtra)); k++ {
				prescription = append(prescription, prescriptionExtra[k])
			}
		}
		createdAt := time.Now()
		var prescription_task []model.PrescriptionTask

		for k := 0; k < len(prescription); k++ {
			prescription_task = append(prescription_task, model.PrescriptionTask{
				Task: model.Task{
					PatientId:   prescription[k].Patient_id,
					UserId:      newTask.UserId,
					TaskType:    "prescription",
					Completed:   false,
					CreatedAt:   &createdAt,
					LLMResponse: nil, // won't be filled in until LLM provides response
					LLMFeedback: nil, // won't be filled in until LLM provides feedback
				},
				PrescriptionId: prescription[k].ID,
			})
		}

		err = h.Supabase.DB.From("tasks").Insert(prescription_task).Execute(&prescription_task)
		for k := 0; k < len(prescription_task); k++ {
			for i := 0; i < len(relatedPatients); i++ {
				if prescription[k].Patient_id == relatedPatients[i].Id {
					prescription[k].Patient.Name = relatedPatients[i].Name
					break
				}
			}
			prescription[k].ID = *prescription_task[k].Id
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prescription)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nil)
}

func (h *TaskHandler) GetTaskCalendar(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["student_id"]
	type Task struct {
		Id          *uuid.UUID `json:"id,omitempty"`
		CreatedAt   time.Time  `json:"created_at,omitempty"` // Pointer to avoid default time
		PatientId   uuid.UUID  `json:"patient_id"`
		Completed   bool       `json:"completed"`
		CompletedAt time.Time  `json:"completed_at,omitempty"`
		Name        string     `json:"name"`
		TaskType    string     `json:"task_type"`
	}
	type TaskReturn struct {
		Tasks      map[string][]Task `json:"tasks,omitempty"`
		OldestDate time.Time         `json:"oldestDate,omitempty"`
	}
	var tasks []Task
	returnTasks := TaskReturn{
		Tasks:      nil,
		OldestDate: time.Now(),
	}

	idInterface := make(map[string]interface{})
	idInterface["student_id"] = id
	err := h.Supabase.DB.Rpc("get_tasks", idInterface).Execute(&tasks)
	if err != nil {
		fmt.Println(err)
		return
	}

	var allTasks map[string][]Task
	allTasks = make(map[string][]Task)

	for j := 0; j < len(tasks); j++ {
		orientedTaskDate := tasks[j].CompletedAt.In(time.Now().Location())
		orientedTaskDate = time.Date(orientedTaskDate.Year(), orientedTaskDate.Month(), orientedTaskDate.Day(), 0, 0, 0, 0, orientedTaskDate.Location())
		allTasks[orientedTaskDate.String()] = append(allTasks[orientedTaskDate.String()], tasks[j])

		tasks[j].CreatedAt = tasks[j].CreatedAt.In(time.Now().Location())
		tasks[j].CreatedAt = time.Date(tasks[j].CreatedAt.Year(), tasks[j].CreatedAt.Month(), tasks[j].CreatedAt.Day(), 0, 0, 0, 0, tasks[j].CreatedAt.Location())
		if tasks[j].CreatedAt.Before(returnTasks.OldestDate) {
			returnTasks.OldestDate = tasks[j].CreatedAt
		}
	}
	returnTasks.Tasks = allTasks

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(returnTasks)
}
