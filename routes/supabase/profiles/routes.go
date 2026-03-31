package profiles

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gorilla/mux"
	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
	request "gitlab.msu.edu/team-corewell-2025/routes/supabase"
)

type ProfilesHandler struct {
	Supabase *supabase.Client
}

func (h *ProfilesHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Fetch user directly by ID
	vars := mux.Vars(r)
	userId := vars["id"]
	var users []model.User
	fmt.Printf("in profiles, id: %+v\n", userId)
	err := h.Supabase.DB.From("users").Select("*").Eq("id", userId).Execute(&users)
	if err != nil {
		http.Error(w, "failed to query user profile", http.StatusInternalServerError)
		return
	}
	if len(users) == 0 {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	fmt.Println(users)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users[0])
}

func (h *ProfilesHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {

	var userUpdateRequest request.UserUpdateRequest
	bodyBytes, err := io.ReadAll(r.Body)
	err = err
	json.Unmarshal(bodyBytes, &userUpdateRequest)

	err = h.Supabase.DB.From("users").Update(userUpdateRequest).Eq("id", fmt.Sprint(userUpdateRequest.ID)).Execute(nil)
	if err != nil {
		http.Error(w, "failed to query user profile", http.StatusInternalServerError)
		return
	}
}

func (h *ProfilesHandler) UpdateLastActive(w http.ResponseWriter, r *http.Request) {

	type IdHolder struct {
		ID uuid.UUID `json:"id"`
	}
	type TimeInfo struct {
		TimeOnline    int       `json:"timeOnline"`
		LastHeartbeat time.Time `json:"lastHeartbeat"`
	}
	var timeInfo TimeInfo

	var idRequest IdHolder
	bodyBytes, err := io.ReadAll(r.Body)
	err = err
	json.Unmarshal(bodyBytes, &idRequest)

	err = h.Supabase.DB.From("users").Select("timeOnline", "lastHeartbeat").Single().Eq("id", fmt.Sprint(idRequest.ID)).Execute(&timeInfo)
	if err != nil {
		http.Error(w, "failed to query user profile", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	timeInfo.TimeOnline += 5
	timeInfo.LastHeartbeat = time.Now()

	err = h.Supabase.DB.From("users").Update(timeInfo).Eq("id", fmt.Sprint(idRequest.ID)).Execute(nil)
	if err != nil {
		http.Error(w, "failed to query user profile", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}
}

func (h *ProfilesHandler) UpdateFeedback(w http.ResponseWriter, r *http.Request) {

	type FeedbackType struct {
		ID           uuid.UUID `json:"id"`
		FeedbackType string    `json:"feedback_type"`
	}

	var req FeedbackType

	bodyBytes, err := io.ReadAll(r.Body)
	err = json.Unmarshal(bodyBytes, &req)

	if err != nil {
		http.Error(w, "failed to read body ", http.StatusInternalServerError)
		fmt.Println(err)
		return

	}

	updateData := map[string]interface{}{
		"feedback_type": req.FeedbackType,
	}

	err = h.Supabase.DB.From("users").Update(updateData).Eq("id", fmt.Sprint(req.ID)).Execute(nil)
	fmt.Println(err)
	if err != nil {
		http.Error(w, "failed to update feedback type", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

}

func (h *ProfilesHandler) GetFeedback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId := vars["id"]

	type FeedbackType struct {
		FeedbackType string `json:"feedback_type"`
	}
	var result []FeedbackType

	//ADD ERROR CHECKING

	err := h.Supabase.DB.From("users").Select("feedback_type").Eq("id", userId).Execute(&result)
	print(err)
	print(err)
	json.NewEncoder(w).Encode(result[0])

}

func (h *ProfilesHandler) SaveRatings(w http.ResponseWriter, r *http.Request){
	type RatingType struct {
		STUDENT_ID 			 uuid.UUID 	`json:"user_id"`
		Rating 		 		 int 		`json:"rating"`
		FeedbackType		 string		`json:"feedback_type"`
		Tags				 []string		`json:"tags"`
	}

	var req RatingType

	bodyBytes, err := io.ReadAll(r.Body)
	err = json.Unmarshal(bodyBytes, &req)

	if err != nil{
		http.Error(w, "failed to read body ", http.StatusInternalServerError)
		fmt.Println(err)
		return

	}

	InsertData := map[string]interface{}{
		"user_id": req.STUDENT_ID,
		"rating":req.Rating,
		"feedback_type":req.FeedbackType,
		"tags":req.Tags,
	}


	err = h.Supabase.DB.From("response_rating").Insert(InsertData).Execute(nil)
	fmt.Println(err)
	if err != nil {
		http.Error(w, "failed to update feedback type", http.StatusInternalServerError)
		fmt.Println(err)
		return

	}

}

func (h *ProfilesHandler) GetRatings(w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(r)
	userId := vars["user_id"]

	type RatingType struct {
		
		Rating 		 		 int 		    `json:"rating"`
		Date			     string    		`json:"date"`
		FeedbackType	     string			`json:"feedback_type"`
	}

	var ratings []RatingType
	//fmt.Println("HERE")

	err := h.Supabase.DB.From("response_rating").Select("rating,date,feedback_type").Eq("user_id", userId).Execute(&ratings)
	if err != nil {
		http.Error(w, "failed to query user profile", http.StatusInternalServerError)
		return
	}
	if len(ratings) == 0 {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	fmt.Println(ratings)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ratings)

}


func (h *ProfilesHandler) GetRatingsByType(w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(r)
	userId := vars["user_id"]
	Type := vars["type"]

	type RatingType struct {
		
		Rating 		 		 int 		    `json:"rating"`
		Date			     string    		`json:"date"`
		FeedbackType	     string			`json:"feedback_type`
	}

	var ratings []RatingType
	//fmt.Println("HERE")

	err := h.Supabase.DB.From("response_rating").Select("rating,date,feedback_type").Eq("user_id", userId).Eq("feedback_type", Type).Execute(&ratings)
	if err != nil {
		http.Error(w, "failed to query user profile", http.StatusInternalServerError)
		return
	}
	if len(ratings) == 0 {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	fmt.Println(ratings)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ratings)

}

func (h *ProfilesHandler) GetAllRatings(w http.ResponseWriter, r *http.Request){
	type RatingType struct {
		
		Rating 		 		 int 		    `json:"rating"`
		Date			     string    		`json:"date"`
		FeedbackType	     string			`json:"feedback_type"`
	}

	var ratings []RatingType

	err := h.Supabase.DB.From("response_rating").Select("rating,date,feedback_type").Execute(&ratings)
	if err != nil {
		http.Error(w, "failed to query user profile", http.StatusInternalServerError)
		return
	}
	if len(ratings) == 0 {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	fmt.Println(ratings)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ratings)

}

func (h *ProfilesHandler) GetLeastTags(w http.ResponseWriter, r *http.Request){
	fmt.Println("HEREEE")
	vars := mux.Vars(r)
	userIdStr := vars["user_id"]
	userId, err := uuid.Parse(userIdStr)
	fmt.Println(userId)
	


	type LeastTags struct{
		Tag   string `json:"tag"`
		Count int    `json:"count"`

	}

	var result []LeastTags

	params := map[string]interface{}{
		"p_user_id": userId,
	}

	err = h.Supabase.DB.Rpc("get_least_used_tags", params).Execute( &result)
	if err != nil{
		http.Error(w, "failed to query user tags", http.StatusInternalServerError)
		return

	}
	fmt.Println(result)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *ProfilesHandler) GetTagsStats(w http.ResponseWriter, r *http.Request){
	fmt.Println("HEREEE")
	vars := mux.Vars(r)
	userIdStr := vars["user_id"]
	userId, err := uuid.Parse(userIdStr)
	fmt.Println(userId)
	
	type LeastTags struct{
		Tag   		string 			`json:"tag"`
		Average 	float64    		`json:"avg_rating"`

	}

	var result []LeastTags

	params := map[string]interface{}{
		"p_user_id": userId,
	}

	err = h.Supabase.DB.Rpc("get_lowest_rated_tags", params).Execute( &result)
	if err != nil{
		http.Error(w, "failed to query user tags", http.StatusInternalServerError)
		return

	}
	fmt.Println(result)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)



}