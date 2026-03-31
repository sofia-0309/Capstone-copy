package questionboard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
	request "gitlab.msu.edu/team-corewell-2025/routes/supabase"
)

type QuestionBoardHandler struct {
	Supabase *supabase.Client
}

func (h *QuestionBoardHandler) PostQuestion(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("in post q")

	var req request.InsertQuestionRequest

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		fmt.Println("Unmarshal error:", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	now := time.Now()

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	newQuestion := request.InsertQuestionRequest{
		QuestionID: fmt.Sprintf("q_%d", time.Now().UnixNano()),
		UserID:     req.UserID,
		Nickname:   req.Nickname,
		Content:    req.Content,
		CreatedAt:  &now,
		Tags:       tags,
		PatientID:  req.PatientID,
	}

	// If a patient is attached, ensure it's part of the request user's patient_question tasks.
	// This prevents clients from attaching arbitrary patient records.
	if req.PatientID != nil {
		pid := *req.PatientID
		var matchingTasks []struct {
			PatientID uuid.UUID `json:"patient_id"`
		}

		err := h.Supabase.DB.From("tasks").
			Select("patient_id").
			Eq("user_id", req.UserID.String()).
			Eq("task_type", "patient_question").
			Eq("patient_id", pid.String()).
			Execute(&matchingTasks)

		if err != nil {
			http.Error(w, "failed to validate patient attachment", http.StatusInternalServerError)
			return
		}

		if len(matchingTasks) == 0 {
			http.Error(w, "patient_id is not associated with your tasks", http.StatusForbidden)
			return
		}
	}

	fmt.Printf("question: %+v\n", newQuestion)

	err = h.Supabase.DB.From("question").Insert(newQuestion).Execute(nil)
	if err != nil {
		fmt.Println("Error inserting question:", err)
		http.Error(w, "failed to insert question", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newQuestion)
}

// GetQuestionTags returns the distinct set of patient message tags so the question board can reuse the same vocabulary.
func (h *QuestionBoardHandler) GetQuestionTags(w http.ResponseWriter, r *http.Request) {
	// Keep this endpoint simple: pull all patients' chief_complaint_tags and return the distinct union.
	// If the dataset grows, we can switch to doing distinct/unnest in SQL.
	type patientChiefConcern struct {
		ChiefConcern model.ChiefConcern `json:"chief_concern"`
	}

	var patients []patientChiefConcern
	err := h.Supabase.DB.From("patients").
		Select("chief_concern").
		Execute(&patients)
	if err != nil {
		http.Error(w, "failed to fetch tags", http.StatusInternalServerError)
		return
	}

	tagSet := make(map[string]struct{})
	for _, p := range patients {
		for _, t := range p.ChiefConcern.ChiefComplaintTags {
			if t == "" {
				continue
			}
			tagSet[t] = struct{}{}
		}
	}

	tags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func (h *QuestionBoardHandler) GetQuestions(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("in get q")
	var questions []model.Question

	err := h.Supabase.DB.From("question").Select("*").Execute(&questions)
	if err != nil {
		http.Error(w, "failed to fetch questions", http.StatusInternalServerError)
		return
	}

	//aattach votes
	for i := range questions {
		var votes []struct {
			IsUpvote bool `json:"is_upvote"`
		}
		err := h.Supabase.DB.From("votes").
			Select("is_upvote").
			Eq("target_id", questions[i].QuestionID).
			Execute(&votes)

		if err == nil {
			up, down := 0, 0
			for _, v := range votes {
				if v.IsUpvote {
					up++
				} else {
					down++
				}
			}
			questions[i].Upvotes = up
			questions[i].Downvotes = down
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}

func (h *QuestionBoardHandler) PostReply(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("in post reply")
	var req request.InsertReplyRequest

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	fmt.Println("Request body:", string(bodyBytes))

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		fmt.Println("JSON error:", err)
		http.Error(w, "Error unmarshaling request", http.StatusBadRequest)
		return
	}

	now := time.Now()
	newReply := request.InsertReplyRequest{
		ReplyID:    fmt.Sprintf("r_%d", time.Now().UnixNano()),
		QuestionID: req.QuestionID,
		UserID:     req.UserID,
		Nickname:   req.Nickname,
		Content:    req.Content,
		CreatedAt:  &now,
	}

	err = h.Supabase.DB.From("replies").Insert(newReply).Execute(nil)
	if err != nil {
		fmt.Println("Supabase insert error:", err)
		http.Error(w, "failed to insert reply", http.StatusInternalServerError)
		return
	}

	// Fetch the original question to get the author's user_id for notification
	var questions []model.Question
	err = h.Supabase.DB.From("question").
		Select("*").
		Eq("question_id", req.QuestionID).
		Execute(&questions)

	if err == nil && len(questions) > 0 {
		questionAuthor := questions[0].UserID

		// Don't notify if replying to own question
		if questionAuthor != req.UserID {
			// Create notification for the question author
			actionUrl := fmt.Sprintf("/QuestionBoard?highlight=%s", req.QuestionID)
			notification := model.Notification{
				Id:        uuid.New(),
				UserId:    questionAuthor,
				Type:      "question_reply",
				Title:     "New Reply to Your Question",
				Message:   fmt.Sprintf("%s replied to your question: \"%s\"", req.Nickname, truncateContent(req.Content, 50)),
				IsRead:    false,
				CreatedAt: model.FlexibleTime{Time: now},
				ActionUrl: &actionUrl,
			}

			err = h.Supabase.DB.From("notifications").
				Insert(notification).
				Execute(nil)
			if err != nil {
				// Log error but don't fail the reply request
				fmt.Printf("PostReply: Failed to create notification: %v\n", err)
			} else {
				fmt.Printf("PostReply: Notification created for user %s\n", questionAuthor)
			}
		}
	} else if err != nil {
		// Log error but don't fail the reply request
		fmt.Printf("PostReply: Failed to fetch question for notification: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newReply)
}

func (h *QuestionBoardHandler) GetReplies(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("in get reply")
	questionID := mux.Vars(r)["question_id"]

	var replies []model.Reply
	err := h.Supabase.DB.From("replies").Select("*").Eq("question_id", questionID).Execute(&replies)
	if err != nil {
		http.Error(w, "failed to fetch replies", http.StatusInternalServerError)
		return
	}
	//attach votes to each reply
	for i := range replies {
		var votes []struct {
			IsUpvote bool `json:"is_upvote"`
		}
		err := h.Supabase.DB.From("votes").
			Select("is_upvote").
			Eq("target_id", replies[i].ReplyID).
			Execute(&votes)

		if err == nil {
			up, down := 0, 0
			for _, v := range votes {
				if v.IsUpvote {
					up++
				} else {
					down++
				}
			}
			replies[i].Upvotes = up
			replies[i].Downvotes = down
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(replies)
}

func (h *QuestionBoardHandler) DeleteReply(w http.ResponseWriter, r *http.Request) {

	fmt.Println("in delete reply")
	vars := mux.Vars(r)
	replyID := vars["reply_id"]
	userID := r.Header.Get("User-ID")
	isAdmin := r.Header.Get("Is-Admin") == "true"

	var replies []struct {
		ReplyID string `json:"reply_id"`
		UserID  string `json:"user_id"`
	}

	//get the reply
	err := h.Supabase.DB.From("replies").Select("reply_id,user_id").Eq("reply_id", replyID).Execute(&replies)
	if err != nil {
		fmt.Println("error fetching reply:", err)
		return
	}
	//extra check to prevent panic
	if len(replies) == 0 {
		fmt.Printf("Reply %s not found\n", replyID)
		return
	}

	if !isAdmin && replies[0].UserID != userID {
		fmt.Println("you cant do that")
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}

	//delete it
	err = h.Supabase.DB.From("replies").Delete().Eq("reply_id", replyID).Execute(nil)
	if err != nil {
		fmt.Println("error deleting reply:", err)
		http.Error(w, "failed to delete reply", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *QuestionBoardHandler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	questionID := vars["question_id"]
	userID := r.Header.Get("User-ID")
	isAdmin := r.Header.Get("Is-Admin") == "true"

	var questions []struct {
		QuestionID string `json:"question_id"`
		UserID     string `json:"user_id"`
	}
	// get the question
	err := h.Supabase.DB.From("question").Select("question_id,user_id").Eq("question_id", questionID).Execute(&questions)
	if err != nil {
		fmt.Println("Error fetching question:", err)
		http.Error(w, "failed to fetch question", http.StatusInternalServerError)
		return
	}
	//extra check to prevent panic
	if len(questions) == 0 {
		http.Error(w, "question not found", http.StatusNotFound)
		return
	}

	if !isAdmin && questions[0].UserID != userID {
		fmt.Println("you cant do that")
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}

	//delete replies first
	err = h.Supabase.DB.From("replies").
		Delete().
		Eq("question_id", questionID).
		Execute(nil)
	if err != nil {
		fmt.Println("cant delete replies", err)
	}

	err = h.Supabase.DB.From("question").Delete().Eq("question_id", questionID).Execute(nil)
	if err != nil {
		fmt.Println("Error deleting question:", err)
		http.Error(w, "failed to delete question", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// truncateContent truncates a string to maxLen characters and adds "..." if truncated
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

func (h *QuestionBoardHandler) EndorseReply(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in endorse reply")
	replyID := mux.Vars(r)["reply_id"]
	userID := r.Header.Get("User-ID")
	isAdmin := r.Header.Get("Is-Admin") == "true"

	if !isAdmin {
		http.Error(w, "ur not an instructor, sorry!", http.StatusForbidden)
		return
	}

	updateData := map[string]interface{}{
		"endorsed_by": userID,
	}

	err := h.Supabase.DB.From("replies").Update(updateData).Eq("reply_id", replyID).Execute(nil)
	if err != nil {
		fmt.Println("Error endorsing reply:", err)
		http.Error(w, "Failed to endorse reply", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"endorsed"}`))

}

func (h *QuestionBoardHandler) AddVote(w http.ResponseWriter, r *http.Request) {
	var req request.InsertVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	//check if alr voted on
	var existing []struct {
		IsUpvote bool `json:"is_upvote"`
	}
	err := h.Supabase.DB.From("votes").Select("is_upvote").Eq("user_id", req.UserID).Eq("target_id", req.TargetID).Eq("target_type", req.TargetType).Execute(&existing)

	if err != nil {
		http.Error(w, "failed to check existing votes", http.StatusInternalServerError)
		return
	}

	if len(existing) > 0 {
		// removing votes
		if existing[0].IsUpvote == req.IsUpvote {
			h.Supabase.DB.From("votes").Delete().Eq("user_id", req.UserID).Eq("target_id", req.TargetID).Eq("target_type", req.TargetType).Execute(nil)
		} else {
			// switching votes
			h.Supabase.DB.From("votes").Update(map[string]interface{}{"is_upvote": req.IsUpvote}).Eq("user_id", req.UserID).Eq("target_id", req.TargetID).Eq("target_type", req.TargetType).Execute(nil)
		}
	} else {
		// new one
		newVote := request.InsertVoteRequest{
			VoteID:     uuid.New(),
			UserID:     req.UserID,
			TargetID:   req.TargetID,
			TargetType: req.TargetType,
			IsUpvote:   req.IsUpvote,
		}
		h.Supabase.DB.From("votes").Insert(newVote).Execute(nil)
	}

	// updated counts
	var votes []struct {
		IsUpvote bool `json:"is_upvote"`
	}
	h.Supabase.DB.From("votes").Select("is_upvote").Eq("target_id", req.TargetID).Execute(&votes)

	//keeping track
	up, down := 0, 0
	for _, v := range votes {
		if v.IsUpvote {
			up++
		} else {
			down++
		}
	}

	resp := map[string]int{"upvotes": up, "downvotes": down}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
