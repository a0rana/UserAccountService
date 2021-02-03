package models

type UserDebit struct {
	UserId string  `json:"userid"`
	Amount float64 `json:"amount"`
}
