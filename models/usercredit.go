package models

type UserCredit struct {
	UserId          string  `json:"userid"`
	UserCreditId    uint64  `json:"usercreditid"`
	Updated         string  `json:"updated"`
	Created         string  `json:"created"`
	Amount          float64 `json:"amount"`
	TransactionType string  `json:"transactiontype"`
	Priority        int     `json:"priority"`
	Expiry          string  `json:"expiry"`
	IsExpired       bool    `json:"isexpired"`
}
