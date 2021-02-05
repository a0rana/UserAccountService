package router

import (
	"github.com/a0rana/UserAccountService/middleware"
	"github.com/gorilla/mux"
)

// Router is exported and used in main.go
func Router() *mux.Router {

	router := mux.NewRouter()

	router.HandleFunc("/transactions", middleware.GetAllTransactions).Methods("GET", "OPTIONS")
	router.HandleFunc("/credit", middleware.CreateUserCredit).Methods("POST", "OPTIONS")
	router.HandleFunc("/debit", middleware.CreateUserDebit).Methods("POST", "OPTIONS")

	return router
}
