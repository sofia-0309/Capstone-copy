package orders

import "net/http"

type OrdersService interface {
	GetOrders(w http.ResponseWriter, r *http.Request)
	LogOrder(w http.ResponseWriter, r *http.Request)
	GetOrderedOrders(w http.ResponseWriter, r *http.Request)
}
