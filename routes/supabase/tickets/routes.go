package tickets

import (
	"net/http"
	// "github.com/gorilla/mux"
	// model "gitlab.msu.edu/team-corewell-2025/models"
	"encoding/json"
	"fmt"
	"os"
    "bytes"
    "io"
	"time"
	"github.com/gorilla/mux"
	supabase "github.com/nedpals/supabase-go"

	
)

func getExplainURL() string {
	url := os.Getenv("FLASK_EXPLAIN_URL")
	if url == "" {
		return "http://127.0.0.1:5001/api/explain-request" // default for local dev
	}
	return url
}


type TicketHandler struct {
	Supabase *supabase.Client
}

func (h* TicketHandler) SaveTicket(w http.ResponseWriter, r *http.Request) {

	fmt.Println("go save ticket")

	type Report struct {
		Browser     string       `json:"browser"`
		Platform    string       `json:"platform"`
		Language    string       `json:"language"`
		Timezone    string       `json:"timezone"`
		Network     any          `json:"network"`
		URL         string       `json:"url"`
		Environment string       `json:"environment"`
		Timestamp   string       `json:"timestamp"`
		Screen      any          `json:"screen"`
		Viewport    any          `json:"viewport"`
		API         any          `json:"api"`       
		Image       string       `json:"image"`     
		Message     string       `json:"message"`   


	}

	var data Report

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	other := map[string]interface{}{
		"platform":    data.Platform,
		"language":    data.Language,
		"timezone":    data.Timezone,
		"network":     data.Network,
		"url":         data.URL,
		"environment": data.Environment,
	}

	flaskURL := getExplainURL()

	llmRequest := map[string]interface{}{
    "task_type": "analyze",
    "message":    fmt.Sprintf(
        "based on the user message and the data, analyze the following issue from the student and return only one of the following risk levels high moderate or low for how the application malfunction effecting the student work. Be critical. Answer in one word only (high or low or moderate) all lower case no punctuation no symbols no dot no exclamation no question mark user message %s browser %s platform %s language %s timezone %s network %v url %s environment %s screen %v viewport %v api %v image %s",
        data.Message,
        data.Browser,
        data.Platform,
        data.Language,
        data.Timezone,
        data.Network,
        data.URL,
        data.Environment,
        data.Screen,
        data.Viewport,
    ),
	}

	// convert to slice bytes
	json_data, err := json.Marshal(llmRequest)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
    	fmt.Println("failed to decode json:", err)
	}

	// make request
	req, err:= http.NewRequest("POST",flaskURL, bytes.NewBuffer(json_data))
	if err != nil {
    	fmt.Println("failed to send stuff:", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("request failed")
	}
	defer response.Body.Close()
	
	// read request 
	b, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("failed to read response")
	}

	fmt.Println("Flask Response:")
	fmt.Println(string(b))

	var ai_resp map[string]interface{}
	if err := json.Unmarshal(b, &ai_resp); err != nil {
		fmt.Println("failed to parse AI response: %w", err)
	}

	risk := ai_resp["sample_response"]




	insert_data := map[string]interface{}{
		"browser":    data.Browser,
		"screen":     data.Screen,
		"viewport":   data.Viewport,
		"screenshot": data.Image,
		"message":    data.Message,
		"api":        data.API,
		"timestamp":  data.Timestamp,
		"other":      other,
		"status":	  "open",
		"risk": 	  risk,
	}

	
	if err := h.Supabase.DB.From("tickets").Insert(insert_data).Execute(nil); err != nil {
		http.Error(w, "Error saving ticket"+err.Error(), http.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}

}

func (h *TicketHandler) GetTickets(w http.ResponseWriter, r *http.Request) {
    (w).Header().Set("Access-Control-Allow-Origin", "*")
	(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	w.Header().Set("Content-Type", "application/json")

	type Ticket struct {
		ID        int                    `json:"ticket_id"`
		Browser   string                 `json:"browser"`
		Screen    interface{}            `json:"screen"`
		Viewport  interface{}            `json:"viewport"`
		Screenshot string                `json:"screenshot"`
		Message   string                 `json:"message"`
		API       interface{}            `json:"api"`
		Timestamp string                 `json:"timestamp"`
		Other     map[string]interface{} `json:"other"`
		Status    string                 `json:"status"`
		Risk      string                 `json:"risk"`
		CLosedAt  string                 `json:"closed_at"`
	}


    var tickets []Ticket
    if err := h.Supabase.DB.From("tickets").Select("*").Execute(&tickets); err != nil {
        http.Error(w, "Cannot fetch tickets", http.StatusInternalServerError)
        fmt.Println("Supabase error:", err)
        return
    }

    if err := json.NewEncoder(w).Encode(map[string]interface{}{"tickets": tickets}); err != nil {
        http.Error(w, "Cannot encode tickets", http.StatusInternalServerError)
        fmt.Println("JSON encode error:", err)
    }

}

func (h* TicketHandler) CloseTicket(w http.ResponseWriter, r *http.Request) {
	fmt.Println("😥Oh ahaa...")
	vars := mux.Vars(r)
	id := vars["ticket_id"]

	update_data := map[string]interface{}{
        "status": "closed",
        "closed_at": time.Now().Format(time.RFC3339),
    }

	if err := h.Supabase.DB.From("tickets").Update(update_data).Eq("ticket_id", id).Execute(nil); err != nil {
        http.Error(w, "Failed to close ticket", http.StatusInternalServerError)
		fmt.Println(err)
        return
    }



}