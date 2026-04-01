package orders

import "net/http"

type OrdersService interface {
	GetOrders(w http.ResponseWriter, r *http.Request)
	LogOrder(w http.ResponseWriter, r *http.Request)
	GetOrderedOrders(w http.ResponseWriter, r *http.Request)
	GetOrdersFeedback(w http.ResponseWriter, r *http.Request)
	GetSavedOrdersFeedback(w http.ResponseWriter, r *http.Request)
}
