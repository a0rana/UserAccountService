package middleware

import (
	"UserAccountService/models" // models package where User schema is defined
	"context"
	"database/sql"
	"encoding/json" // package to encode and decode the json into struct and vice versa
	"fmt"
	"github.com/joho/godotenv" // package used to read the .env file
	_ "github.com/lib/pq"      // postgres golang driver
	"log"
	"net/http" // used to access the request and response object of the api
	"os"       // used to read the environment variable
)

//response format
type response struct {
	ID      uint64 `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

// create connection with postgres db
func createConnection() *sql.DB {
	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Open the connection
	db, err := sql.Open("postgres", os.Getenv("POSTGRES_URL"))

	if err != nil {
		panic(err)
	}

	// check the connection
	err = db.Ping()

	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected!")
	// return the connection
	return db
}

// CreateUserCredit create a user-credit in the postgres db
func CreateUserCredit(w http.ResponseWriter, r *http.Request) {
	// set the header to content type x-www-form-urlencoded
	// Allow all origin to handle cors issue
	w.Header().Set("Context-Type", "application/x-www-form-urlencoded")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// create an empty user of type models.User
	var userCredit models.UserCredit

	// decode the json request to user
	err := json.NewDecoder(r.Body).Decode(&userCredit)

	if err != nil {
		log.Fatalf("Unable to decode the request body.  %v", err)
	}

	// call insert user function and pass the user
	insertID := insertUserCredit(userCredit)

	// format a response object
	res := response{
		ID:      insertID,
		Message: "User credit created successfully",
	}

	// send the response
	json.NewEncoder(w).Encode(res)
}

// GetAllUser will return all the users
func GetAllUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Context-Type", "application/x-www-form-urlencoded")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// get all the users in the db
	users, err := getAllUsers()

	if err != nil {
		log.Fatalf("Unable to get all user. %v", err)
	}

	// send all the users as response
	json.NewEncoder(w).Encode(users)
}

//------------------------- handler functions ----------------
// inserts credit in the DB
func insertUserCredit(userCredit models.UserCredit) uint64 {

	// create the postgres db connection
	db := createConnection()

	// close the db connection
	defer db.Close()

	// create the insert sql query
	// returning userid will return the id of the inserted user
	userCreditSqlStatement := `INSERT INTO tbl_UserCredits(userid, amount, transactiontype, priority, expiry)
 					 VALUES ($1, $2, $3, $4, $5) RETURNING usercreditid`

	activitySqlStatement := `INSERT INTO tbl_Activity(userid, iscredit, amount, usercreditid)
 					 VALUES ($1, $2, $3, $4)`

	// the inserted id will store in this id
	var userCreditId uint64

	// Create a new context, and begin a transaction
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	err = tx.QueryRow(userCreditSqlStatement, userCredit.UserId, userCredit.Amount, userCredit.TransactionType,
		userCredit.Priority, userCredit.Expiry).Scan(&userCreditId)

	if err != nil {
		tx.Rollback()
		log.Fatal(err)
		return 0
	}

	// The next query is handled similarly
	_, err = tx.ExecContext(ctx, activitySqlStatement, userCredit.UserId, true, userCredit.Amount, userCreditId)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
		return 0
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Inserted a single record with id: %v and logged activity", userCreditId)

	// return the inserted id
	return userCreditId
}

// get one user from the DB by its userid
func getAllUsers() ([]models.User, error) {
	// create the postgres db connection
	db := createConnection()

	// close the db connection
	defer db.Close()

	var users []models.User

	// create the select sql query
	sqlStatement := `SELECT userid,fname,lname,email,dob,mobile FROM tbl_Users`

	// execute the sql statement
	rows, err := db.Query(sqlStatement)

	if err != nil {
		log.Fatalf("Unable to execute the query. %v", err)
	}

	// close the statement
	defer rows.Close()

	// iterate over the rows
	for rows.Next() {
		var user models.User

		// unmarshal the row object to user
		err = rows.Scan(&user.UserId, &user.FirstName, &user.LastName, &user.Email, &user.DOB, &user.Mobile)

		if err != nil {
			log.Fatalf("Unable to scan the row. %v", err)
		}

		// append the user in the users slice
		users = append(users, user)

	}

	// return empty user on error
	return users, err
}
