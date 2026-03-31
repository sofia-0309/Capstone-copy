package ordersList

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	supabase "github.com/nedpals/supabase-go"
)

type OrdersListHandler struct {
	Supabase *supabase.Client
}

type Order struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Type string    `json:"type"`
}

// Grabs Orders_List from Supabase
func (h *OrdersListHandler) GetOrdersList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var orders []Order
	err := h.Supabase.DB.From("orders_list").Select("*").Execute(&orders)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to fetch orders: %v", err),
		})
		return
	}

	if err := json.NewEncoder(w).Encode(orders); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to encode response",
		})
		return
	}
}

// Handles Adding Orders to Orders_List
func (h *OrdersListHandler) AddOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var newOrder struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&newOrder); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	if newOrder.Name == "" || newOrder.Type == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Both Fields Need to be filled out",
		})
		return
	}

	readyOrder := Order{
		ID:   uuid.New(),
		Name: newOrder.Name,
		Type: newOrder.Type,
	}

	// Can you tell i'm running out of ways to say New Order ?
	var insertedOrder []Order
	err := h.Supabase.DB.From("orders_list").Insert(readyOrder).Execute(&insertedOrder)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to insert order: %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(insertedOrder[0])
}

func (h *OrdersListHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	orderId := vars["id"]

	var updatedData struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updatedData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid body",
		})
		return
	}

	// Checking that the user actually did make a change
	if updatedData.Name == "" && updatedData.Type == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "At least one field (name or type) must be provided",
		})
		return
	}

	newData := make(map[string]interface{})
	if updatedData.Name != "" {
		newData["name"] = updatedData.Name
	}
	if updatedData.Type != "" {
		newData["type"] = updatedData.Type
	}

	var updatedOrder []Order

	err := h.Supabase.DB.From("orders_list").Update(newData).Eq("id", orderId).Execute(&updatedOrder)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to update order %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedOrder[0])
}

func (h *OrdersListHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	orderId := vars["id"]

	var deletedOrder []Order
	err := h.Supabase.DB.From("orders_list").Delete().Eq("id", orderId).Execute(&deletedOrder)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to delete order: %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Order deleted successfully",
		"id":      orderId,
	})
}
