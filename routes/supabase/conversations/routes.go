package conversations

import (
	"net/http"
	"github.com/gorilla/mux"
	"encoding/json"
	"fmt"
	"strconv"
	model "gitlab.msu.edu/team-corewell-2025/models"
	supabase "github.com/nedpals/supabase-go"

	
)
type ConversationHandler struct {
	Supabase *supabase.Client
}
func (h* ConversationHandler) GetConversationForQuiz(w http.ResponseWriter, r *http.Request) {
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

	var conversation_query []model.Conversation
	if err := h.Supabase.DB.From("conversations").Select("*").Eq("quiz_id", id).Eq("user_id", sid).Execute(&conversation_query); err != nil {
		http.Error(w, "Cannot fetch question", http.StatusInternalServerError)
		return


	}

	if len(conversation_query) == 0 {
        json.NewEncoder(w).Encode([]model.Message{})
		return
    }

    conversation_id := conversation_query[0].ConversationId
	fmt.Println(conversation_query)

	var messages_query []model.Message
    if err := h.Supabase.DB.From("messages").Select("*").Eq("conversation_id", strconv.Itoa(conversation_id)).Execute(&messages_query); err != nil {
        http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		fmt.Println(err)
        return
    }

	fmt.Println(messages_query)
	if err := json.NewEncoder(w).Encode(messages_query); err != nil {
        http.Error(w, "Failed to encode messages", http.StatusInternalServerError)
        return
    }


}