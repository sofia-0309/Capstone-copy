package orders

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
	request "gitlab.msu.edu/team-corewell-2025/routes/supabase"
)

type OrdersHandler struct {
	Supabase *supabase.Client
}

func (h *OrdersHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	var orders []model.Order
	err := h.Supabase.DB.From("orders").Select("*").Execute(&orders)
	if err != nil {
		msg := fmt.Sprintf("GetOrders: DB select error: %v", err)
		fmt.Println(msg)
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}
	ordersJSON, err := json.MarshalIndent(orders, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(ordersJSON)
}

func (h *OrdersHandler) LogOrder(w http.ResponseWriter, r *http.Request) {
	var req request.InsertOrdersRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		fmt.Println(err)
		http.Error(w, "Error unmarshaling request", http.StatusBadRequest)
		return
	}
	time := time.Now()
	newOrder := request.InsertOrdersRequest{
		ID:        uuid.New(),
		PatientID: req.PatientID,
		TaskID:    req.TaskID,
		Name:      req.Name,
		Date:      &time,
		Urgency:   req.Urgency,
		TimeFrame: req.TimeFrame,
	}

	err = h.Supabase.DB.From("orders").Insert(newOrder).Execute(nil)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error inserting new order row", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *OrdersHandler) GetOrderedOrders(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Backend", r.Method, "URL:", r.URL.Path)
	patientID := r.URL.Query().Get("patient_id")
	if patientID == "" {
		http.Error(w, "Missing patient_id parameter", http.StatusBadRequest)
		return
	}

	var orders []model.OrderedOrders

	// Query the database for orders for this patient
	err := h.Supabase.DB.From("orders").Select("*").Eq("patient_id", patientID).Execute(&orders)

	if err != nil {
		fmt.Printf("GetOrderedOrders: DB select error: %v\n", err)
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}

	// Marshal and return as JSON
	ordersJSON, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		fmt.Printf("GetOrderedOrders: JSON marshal error: %v\n", err)
		http.Error(w, "Failed to marshal orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(ordersJSON)
}
