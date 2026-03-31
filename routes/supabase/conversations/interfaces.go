package conversations
import (
	"net/http"

)

type ConversationService interface {
	GetConversationForQuiz(w http.ResponseWriter, r *http.Request)

}