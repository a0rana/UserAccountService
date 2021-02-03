package models

type UserActivity struct {
	UserId       string  `json:"userid"`
	TranId       uint64  `json:"tranid,omitempty"`
	Created      string  `json:"created"`
	IsCredit     bool    `json:"iscredit"`
	Amount       float64 `json:"amount"`
	UserCreditId uint64  `json:"usercreditid,omitempty"`
}
