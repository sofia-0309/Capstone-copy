package tickets
import (
	"net/http"

)

type TicketService interface {
	SaveTicket(w http.ResponseWriter, r *http.Request)
	GetTickets(w http.ResponseWriter, r *http.Request)
	CloseTicket(w http.ResponseWriter, r *http.Request)
}