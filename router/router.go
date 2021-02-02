package router

import (
	"UserAccountService/middleware"
	"github.com/gorilla/mux"
)

// Router is exported and used in main.go
func Router() *mux.Router {

	router := mux.NewRouter()

	router.HandleFunc("/users", middleware.GetAllUser).Methods("GET", "OPTIONS")
	router.HandleFunc("/credits", middleware.CreateUserCredit).Methods("POST", "OPTIONS")

	return router
}
