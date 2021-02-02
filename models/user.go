package models

type User struct {
	UserId    string `json:"userid"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
	DOB       string `json:"dateofbirth"`
	Mobile    string `json:"mobile"`
}
