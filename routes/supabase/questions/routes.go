package questions

import (
	"net/http"
	"github.com/gorilla/mux"
	"encoding/json"
	"fmt"
	"strconv"
	model "gitlab.msu.edu/team-corewell-2025/models"
	supabase "github.com/nedpals/supabase-go"

	
)
type QuestionHandler struct {
	Supabase *supabase.Client
}
func (h* QuestionHandler) GetQuestionsForQuiz(w http.ResponseWriter, r *http.Request) {

	fmt.Println("😥😥 Get the fuck over here come on...")
	vars := mux.Vars(r)
	id := vars["quiz_id"]
	fmt.Println("quiz_id is:", id)
	sid := vars["student_id"]
	fmt.Println("student_id is:", sid)
	//necessary to enable cors 
	(w).Header().Set("Access-Control-Allow-Origin", "*")
	(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	w.Header().Set("Content-Type", "application/json")

	var questions_query []model.DQuestions
	if err := h.Supabase.DB.From("questions").Select("*").Eq("quiz_id", id).Execute(&questions_query); err != nil {
		http.Error(w, "Cannot fetch question", http.StatusInternalServerError)
		return


	}

	var progress_query []model.UserProgress
	if err := h.Supabase.DB.From("user_progress").Select("*").Eq("user_id", sid).Eq("quiz_id", id).Execute(&progress_query); err != nil {
		http.Error(w, "Cannot fetch progress", http.StatusInternalServerError)
		return


	}


	type WholeThing struct {
		model.DQuestions
		Options []model.Options `json:"options"`
		SelectedOptionId *int `json:"selected_option_id"`
	}

	var data []WholeThing

	// i think theres maybe a better way to design schema but it works then works😅
	for _ , q := range questions_query {
		var options_query []model.Options
		if err := h.Supabase.DB.From("options").Select("*").Eq("question_id", strconv.Itoa(q.QuestionId)).Execute(&options_query); err != nil {
			http.Error(w, "Cannot fetch answer", http.StatusInternalServerError)
			return


		}
		var selection *int = nil
		for _, p := range progress_query {
			if p.QuestionId == q.QuestionId {
				selection = &p.SelectedOptionId
				break
			}
		}

		qwitho := WholeThing{
			DQuestions:      q,
			Options:         options_query,
			SelectedOptionId: selection,
		}

		data = append(data, qwitho)

	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Cannot encode question", http.StatusInternalServerError)
		return

	}

}

func (h *QuestionHandler) CompleteQuestion(w http.ResponseWriter, r *http.Request) {
	// get stuff from URL
	fmt.Println("😥😥 Noooooooooo...")

	vars := mux.Vars(r)
	q_id := vars["quiz_id"]
	u_id := vars["student_id"]
	qu_id := vars["question_id"]
	o_id := vars["selected_option_id"]

	progress_data := map[string]interface{}{
		"user_id":            u_id,
		"quiz_id":            q_id,
		"question_id":        qu_id,
		"selected_option_id": o_id,
	}

	update_data := map[string]interface{}{
		"selected_option_id": o_id,
	}

	var existing []map[string]interface{}
	if err := h.Supabase.DB.From("user_progress").Select("selected_option_id").Eq("user_id", u_id).Eq("quiz_id", q_id).Eq("question_id", qu_id).Execute(&existing); err != nil {
		fmt.Println("Error fetching progress:", err)
		http.Error(w, "Failed to fetch progress", http.StatusInternalServerError)
		return
	}
	fmt.Println(update_data)

	if len(existing) > 0 {
		if err := h.Supabase.DB.From("user_progress").Update(update_data).Eq("user_id", u_id).Eq("quiz_id", q_id).Eq("question_id", qu_id).Execute(nil); err != nil {
			http.Error(w, "Failed to update progress", http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.Supabase.DB.From("user_progress").Insert(progress_data).Execute(nil); err != nil {
			http.Error(w, "Failed to insert progress", http.StatusInternalServerError)
			return
		}
	}


}


func (h* QuestionHandler) GetQuestionProgress(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	u_id :=  vars["student_id"]
	q_id := vars["quiz_id"]

	//necessary to enable cors 
	(w).Header().Set("Access-Control-Allow-Origin", "*")
	(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	w.Header().Set("Content-Type", "application/json")

	// fetching current progress
	var progress_data []model.UserProgress
	if err := h.Supabase.DB.From("user_progress").Select("*").Eq("user_id", u_id).Eq("quiz_id", q_id).Execute(&progress_data); err != nil {
		http.Error(w, "Cannot fetch progress", http.StatusInternalServerError)
		return


	}

	//send to frontend
	if err := json.NewEncoder(w).Encode(progress_data); err != nil {
		http.Error(w, "Cannot encode progress", http.StatusInternalServerError)
		return

	}

	
}

func (h* QuestionHandler) GetQuestionStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	qu_id := vars["question_id"]

	// enable cors
	(w).Header().Set("Access-Control-Allow-Origin", "*")
	(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	w.Header().Set("Content-Type", "application/json")

	// get the options for this question, we need this later
	var option_data []model.Options
	if err := h.Supabase.DB.From("options").Select("*").Eq("question_id", qu_id).Execute(&option_data); err != nil {
		http.Error(w, "Failed to fetch options for questions", http.StatusInternalServerError)
		return
	}

	// get all the entries across all students for this question
	var progress_data []model.UserProgress
	if err := h.Supabase.DB.From("user_progress").Select("*").Eq("question_id", qu_id).Execute(&progress_data); err != nil {
		http.Error(w, "Failed to fetch user progress", http.StatusInternalServerError)
		return
	}

	c := map[int]int{}
	total := 0

	for _, entry := range progress_data {
		c[entry.SelectedOptionId]++
		total++
	}

	type OptionStat struct {
		OptionID   int     `json:"option_id"`
		Count      int     `json:"count"`
		Percentage int `json:"percentage"`
	}

	// calculate percentages for ech option
	var stats []OptionStat
	for _, o := range option_data {
		count := c[o.OptionId]
		percentage := 0
		if total> 0 {
			percentage = int(float64(count)*100/float64(total))
			fmt.Println("percentage")
			fmt.Println(percentage)
		}
		stats = append(stats, OptionStat{
			OptionID:   o.OptionId,
			Count:      count,
			Percentage: percentage,
		})
	}


	statistics := map[string]int{
		"A": stats[0].Percentage,
		"B": stats[1].Percentage,
		"C": stats[2].Percentage,
		"D": stats[3].Percentage,
	}

	if err := json.NewEncoder(w).Encode(statistics); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}



}


func (h* QuestionHandler) FlaggedQuestionRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	qu_id := vars["question_id"]

	(w).Header().Set("Access-Control-Allow-Origin", "*")
	(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	w.Header().Set("Content-Type", "application/json")

	type FlagRequest struct {
		UserID     string `json:"user_id"`
		UserName   string `json:"user_name"` 
		Explanation string `json:"explanation"`
	}

	var flag FlagRequest

	if err := json.NewDecoder(r.Body).Decode(&flag); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	data := map[string]interface{}{
		"user_id":     flag.UserID,
		"question_id": qu_id,
		"message":     flag.Explanation,
	}

	if err := h.Supabase.DB.From("flagged_questions").Insert(data).Execute(nil); err != nil {
		http.Error(w, "Failed to flag question", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Question flagged successfully",
	})
	
}

func (h* QuestionHandler) GetFlaggedQuestions(w http.ResponseWriter, r *http.Request) {

	(w).Header().Set("Access-Control-Allow-Origin", "*")
	(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	w.Header().Set("Content-Type", "application/json")
	var flagged_data []model.FlaggedQuestion
	if err := h.Supabase.DB.From("flagged_questions").Select("*").Execute(&flagged_data); err != nil {
		http.Error(w, "Cannot fetch flagged", http.StatusInternalServerError)
		return


	}
	grouped := map[int][]map[string]string{}



	for _, flag := range flagged_data {
		qu_id := flag.QuestionId 
		if _, exists := grouped[qu_id]; !exists {
			grouped[qu_id] = []map[string]string{}
		}
		grouped[qu_id] = append(grouped[qu_id],map[string]string{
			"user_id": flag.UserId.String(),
			"message": flag.Message,
		})
	}

	type FlagOutput struct {
		QuestionId int                 `json:"question_id"`
		Question model.DQuestions `json:"question"`
		Options []model.Options `json:"options"`
		Flags      []map[string]string `json:"flags"`
	}
	var output []FlagOutput
	for qu_id, flags := range grouped {
		var questions_query []model.DQuestions
		if err := h.Supabase.DB.From("questions").Select("*").Eq("question_id", strconv.Itoa(qu_id)).Execute(&questions_query); err != nil {
			http.Error(w, "Cannot fetch question", http.StatusInternalServerError)
			return
		}
		var options_query []model.Options
		if err := h.Supabase.DB.From("options").Select("*").Eq("question_id", strconv.Itoa(qu_id)).Execute(&options_query); err != nil {
			http.Error(w, "Cannot fetch answer", http.StatusInternalServerError)
			return


		}
		output = append(output, FlagOutput{
			QuestionId: qu_id,
			Question:      questions_query[0],
			Options:         options_query,
			Flags:      flags,
		})
	}

	if err := json.NewEncoder(w).Encode(output); err != nil {
		http.Error(w, "Cannot encode flagged data", http.StatusInternalServerError)
		return
	}

	
}


func (h* QuestionHandler) FlaggedQuestionResolve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	qu_id := vars["question_id"]
	fmt.Println("heheheheehehehe")

	type Option struct {
		OptionId   int    `json:"option_id"`
		QuestionId int    `json:"question_id"`
		OptionText string `json:"option_text"`
		IsCorrect  bool   `json:"is_correct"`
	}

	type FlaggedQuestionUpdate struct {
		QuestionText string   `json:"question_text"`
		Options      []Option `json:"options"`
	}
	var update FlaggedQuestionUpdate

    if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

	if err := h.Supabase.DB.From("questions").Update(map[string]interface{}{"question_text": update.QuestionText,}).Eq("question_id", qu_id).Execute(nil); err != nil {
		http.Error(w, "Failed to update question", http.StatusInternalServerError)
		return
	}

	for _, o := range update.Options {
		update_option := map[string]interface{}{
			"option_text": o.OptionText,
			"is_correct":  o.IsCorrect,
		}

		if err := h.Supabase.DB.From("options").Update(update_option).Eq("option_id", strconv.Itoa(o.OptionId)).Eq("question_id", strconv.Itoa(o.QuestionId)).Execute(nil); err != nil {
			http.Error(w, "Failed to update option", http.StatusInternalServerError)
			return
		}
	}

	if err := h.Supabase.DB.From("flagged_questions").Delete().Eq("question_id", qu_id).Execute(nil); err != nil {
		http.Error(w, "Failed to remove from flagged_questions", http.StatusInternalServerError)
		return
	}
	
}

