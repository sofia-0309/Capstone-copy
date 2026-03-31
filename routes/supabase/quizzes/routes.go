package quizzes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
	"github.com/gorilla/mux"
	model "gitlab.msu.edu/team-corewell-2025/models"
	supabase "github.com/nedpals/supabase-go"
	request "gitlab.msu.edu/team-corewell-2025/routes/supabase"

	
)

// getExplainURL returns the Flask explain service URL from environment or defaults to local
func getExplainURL() string {
	url := os.Getenv("FLASK_EXPLAIN_URL")
	if url == "" {
		return "http://127.0.0.1:5001/api/explain-request" // default for local dev
	}
	return url
}

type QuizHandler struct {
	Supabase *supabase.Client
}

func (h *QuizHandler) GenerateQuestions(topics []string) error {

	if h.Supabase == nil || h.Supabase.DB == nil {
        return fmt.Errorf("Supabase client is not initialized")
    }
	flaskURL := getExplainURL()

	llmRequest := map[string]interface{}{
    "task_type": "mcq",
    "topics":    topics,
	}

	// convert to slice bytes
	json_data, err := json.Marshal(llmRequest)
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	// make request
	req, err:= http.NewRequest("POST",flaskURL, bytes.NewBuffer(json_data))
	if err != nil {
		return fmt.Errorf("failed to send stuff")
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed")
	}
	defer response.Body.Close()
	
	// read request 
	b, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response")
	}

	fmt.Println("Flask Response:")
	fmt.Println(string(b))


	var questions []model.Questions
    if err := json.Unmarshal(b,&questions); err != nil {
        return fmt.Errorf("failed to decode response: %w", err)
    }

	

	// create quiz in db
	quiz_data := map[string]interface{}{
    "created_at": time.Now().Format(time.RFC3339),
	}
	var quiz_res []map[string]interface{}
	if err := h.Supabase.DB.From("quizzes").Insert(quiz_data).Execute(&quiz_res); err != nil {
		return fmt.Errorf("failed to insert quiz: %w", err)
	}
	quiz_id:= int(quiz_res[0]["quiz_id"].(float64))

	for _, q := range questions {

		// insrt question
		question_data := map[string]interface{}{
			"quiz_id":       quiz_id,
			"question_text": q.Question,
		}
		var q_res []map[string]interface{}
		if err := h.Supabase.DB.From("questions").Insert(question_data).Execute(&q_res); err != nil {
			return fmt.Errorf("failed to insert question: %w", err)
		}
		question_id := int(q_res[0]["question_id"].(float64))

		// insert option A
		optiona_data := map[string]interface{}{
			"question_id": question_id,
			"option_text": q.A,
			"is_correct":  q.Answer == "A",
		}
		if err := h.Supabase.DB.From("options").Insert(optiona_data).Execute(nil); err != nil {
			fmt.Println(question_id)
			return fmt.Errorf("failed to insert option A: %w", err)
		}

		// insert option B
		optionb_data := map[string]interface{}{
			"question_id": question_id,
			"option_text": q.B,
			"is_correct":  q.Answer == "B",
		}
		if err := h.Supabase.DB.From("options").Insert(optionb_data).Execute(nil); err != nil {
			fmt.Println(question_id)
			return fmt.Errorf("failed to insert option B: %w", err)
		}

		// insert option C
		optionc_data := map[string]interface{}{
			"question_id": question_id,
			"option_text": q.C,
			"is_correct":  q.Answer == "C",
		}
		if err := h.Supabase.DB.From("options").Insert(optionc_data).Execute(nil); err != nil {
			fmt.Println(question_id)
			return fmt.Errorf("failed to insert option C: %w", err)
		}

		// insert option D
		optiond_data := map[string]interface{}{
			"question_id": question_id,
			"option_text": q.D,
			"is_correct":  q.Answer == "D",
		}
		if err :=h.Supabase.DB.From("options").Insert(optiond_data).Execute(nil); err != nil {
			fmt.Println(question_id)
			return fmt.Errorf("failed to insert option D: %w", err)
		}
	}

	// get students
	var students []model.User
	err = h.Supabase.DB.From("users").Select("*").Eq("isAdmin", "FALSE").Execute(&students)
	if err != nil || len(students) == 0 {
		fmt.Println("No students found")
		return err
	}

	// create records for each students (all get the same quiz for now)
	for _, student := range students {
		record_data := map[string]interface{}{
				"quiz_id": quiz_id,
				"user_id": student.Id,
				"score": 0,
				"created_at": time.Now().Format(time.RFC3339),
		}
		if err :=h.Supabase.DB.From("records").Insert(record_data).Execute(nil); err != nil {
			return fmt.Errorf("failed to insert rec: %w", err)
		}
	}






	

	
	return nil

}

func (h *QuizHandler) GiveInitialQuiz(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["student_id"]
	fmt.Println("give init quiz steady")
	type Quiz struct {
		QuizID int `json:"quiz_id"`
	}

	var quizzes []Quiz
	if err := h.Supabase.DB.From("quizzes").Select("quiz_id").Execute(&quizzes); err != nil {
		fmt.Println("Fail getting quizzes", err)
		http.Error(w, "Failed to fetch quizzes", http.StatusInternalServerError)
		return
	}

	if len(quizzes) == 0 {
		http.Error(w, "No quiz found to give first", http.StatusInternalServerError)
		return
	}

	last_id := quizzes[len(quizzes)-1].QuizID

	record_data := map[string]interface{}{
		"quiz_id": last_id,
		"user_id": id,
		"score": 0,
		"created_at": time.Now().Format(time.RFC3339),
	}
	if err :=h.Supabase.DB.From("records").Insert(record_data).Execute(nil); err != nil {
		fmt.Println("Fail insert record for first quiz",err)
		http.Error(w, "Fail to save first quiz to db", http.StatusInternalServerError)
		return 
	}
}

func (h* QuizHandler) GetQuizForIndividuals(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["student_id"]
	fmt.Println("😥Oh no...")

	//necessary to enable cors 
	(w).Header().Set("Access-Control-Allow-Origin", "*")
	(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	w.Header().Set("Content-Type", "application/json")

	var record_query []model.Records
	if err := h.Supabase.DB.From("records").Select("*").Eq("user_id", id).Execute(&record_query); err != nil {
		http.Error(w, "Cannot fetch record", http.StatusInternalServerError)
		return

	}

	if err := json.NewEncoder(w).Encode(record_query); err != nil {
		http.Error(w, "Cannot encode record", http.StatusInternalServerError)
		return

	}


}

func (h* QuizHandler) CompleteQuiz(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	q_id := vars["quiz_id"]
	u_id :=  vars["student_id"]


	// find the quiz  
	var rec []model.Records
	if err := h.Supabase.DB.From("records").Select("*").Eq("quiz_id", q_id).Eq("user_id", u_id).Execute(&rec); err != nil || len(rec) == 0 {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}
	var QuizCompleteReq request.QuizCompleteReq

	bodyBytes, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(bodyBytes, &QuizCompleteReq); err != nil {
		http.Error(w, "Cannot Unmarshal quiz completion request from request", http.StatusBadRequest)
		return
	}

	updateData := map[string]interface{}{
		"score": QuizCompleteReq.Score,
	}

	if err := h.Supabase.DB.From("records").Update(updateData).Eq("quiz_id", q_id).Eq("user_id", u_id).Execute(nil); err != nil {
		http.Error(w, "Failed to update score", http.StatusInternalServerError)
		return
	}





}

