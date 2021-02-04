package main

import (
	"UserAccountService/router"
	"bytes"
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var db *sql.DB

//integration test cases for the rest-api
func TestMain(m *testing.M) {
	db = createConnection()
	ensureTableExists()
	clearTable()
	createUser()
	code := m.Run()
	os.Exit(code)
}

//test case to verify if activities are provided before any credit/debit's are made
func TestEmptyActivity(t *testing.T) {
	userid := getUser()
	var jsonStr = []byte(fmt.Sprint(`{"userid":"`, userid, `"}`))
	req, _ := http.NewRequest("GET", "/transactions", bytes.NewBuffer(jsonStr))
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); !strings.Contains(body, "Cannot find any transaction history for given user") {
		t.Errorf("Expected no activity for a user. Got %s", body)
	}
}

//test case to verify credit for a user is processed or not
func TestPostUserCredit(t *testing.T) {
	userid := getUser()
	var jsonStr = []byte(fmt.Sprint(`{"userid":"`, userid, `","amount":5,"transactiontype":"Refund","priority":5,"expiry":"2021-10-19 10:23:54"}`))
	req, _ := http.NewRequest("POST", "/credit", bytes.NewBuffer(jsonStr))
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); !strings.Contains(body, "User credit created successfully") {
		t.Errorf("Expected user credit to be created successfully. Got %s", body)
	}
}

//test case to verify if debit for a user is processed or not
func TestPostUserDebit(t *testing.T) {
	userid := getUser()
	var jsonStr = []byte(fmt.Sprint(`{"userid":"`, userid, `","amount":5}`))
	req, _ := http.NewRequest("POST", "/debit", bytes.NewBuffer(jsonStr))
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); !strings.Contains(body, "User debit has been processed successfully") {
		t.Errorf("Expected user debit to be created successfully. Got %s", body)
	}
}

//test case to verify if user activity is generated after processing credits and debits
func TestUserActivity(t *testing.T) {
	userid := getUser()
	var jsonStr = []byte(fmt.Sprint(`{"userid":"`, userid, `"}`))
	req, _ := http.NewRequest("GET", "/transactions", bytes.NewBuffer(jsonStr))
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); !strings.Contains(body, userid) {
		t.Errorf("Expected activity for the user. Got %s", body)
	}
}

//----------------------------- helper methods ------------------------------------
//function to execute the http request, after invoking the matched route's handler
func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.Router().ServeHTTP(rr, req)

	return rr
}

//function to match expected and actual response code after running the test case
func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

// create connection with postgres db
func createConnection() *sql.DB {
	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=TestUserAccount sslmode=%s",
		os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_SSLMODE"))

	// Open the connection
	db, err := sql.Open("postgres", connectionString)

	if err != nil {
		panic(err)
	}

	// check the connection
	err = db.Ping()

	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected to the database!")
	// return the connection
	return db
}

//function to check for tables in db, if not present then create same, if present then ignore
func ensureTableExists() {
	for _, query := range getTableCreationQueries() {
		if _, err := db.Exec(query); err != nil {
			log.Fatal(err)
		}
	}
}

//function to fetch the singular user created while executing the test cases
func getUser() string {
	var userid string
	rows, err := db.Query(`SELECT userid FROM tbl_users LIMIT 1`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&userid)
	}
	return userid
}

//function to insert a single user in the table
func createUser() {
	fmt.Print("create user called")
	for _, query := range getUserInsertStatement() {
		if _, err := db.Exec(query); err != nil {
			log.Fatal(err)
		}
	}
}

//function to delete the tables and reset the sequences for the auto increment ids
func clearTable() {
	db.Exec("DELETE FROM tbl_activity")
	db.Exec("DELETE FROM tbl_usercredits")
	db.Exec("DELETE FROM tbl_Users")
	db.Exec("ALTER SEQUENCE tbl_users_userid_seq RESTART")
	db.Exec("ALTER SEQUENCE tbl_usercredits_usercreditid_seq RESTART")
	db.Exec("ALTER SEQUENCE tbl_activity_tranid_seq RESTART")
}

//function to fetch create table queries
func getTableCreationQueries() []string {
	tableCreationQuery := make([]string, 3)
	tableCreationQuery[0] = `CREATE TABLE IF NOT EXISTS tbl_Users
	(
		userid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		fname  VARCHAR(20),
		lname  VARCHAR(20),
		email  VARCHAR(30),
		dob    DATE,
		mobile VARCHAR(10)
	)`
	tableCreationQuery[1] = `CREATE TABLE IF NOT EXISTS tbl_UserCredits
	(
		userid          UUID REFERENCES tbl_Users (userid),
		usercreditid    BIGSERIAL UNIQUE,
		updated         TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC'),
		created         TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC'),
		amount          NUMERIC(10, 2) NOT NULL,
		transactiontype VARCHAR(10),
		priority        INTEGER,
		expiry          TIMESTAMP WITHOUT TIME ZONE NOT NULL,
		isexpired       BOOLEAN DEFAULT FALSE,
		PRIMARY KEY (userid, usercreditid)
	)`
	tableCreationQuery[2] = `CREATE TABLE IF NOT EXISTS tbl_Activity
	(
		userid       UUID REFERENCES tbl_Users (userid),
		tranid       BIGSERIAL,
		created      TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC'),
		iscredit     BOOLEAN DEFAULT TRUE,
		amount       NUMERIC(10, 2) NOT NULL,
		usercreditid BIGINT REFERENCES tbl_UserCredits (usercreditid),
		PRIMARY KEY (userid, tranid)
	)`
	return tableCreationQuery
}

//function to fetch query to insert a single user in table
func getUserInsertStatement() []string {
	query := make([]string, 1)
	query[0] = `INSERT INTO tbl_Users(fname, lname, email, dob, mobile) VALUES('John', 'Doe', 'john.doe@gmail.com', '1987-11-10', '9994447878')`
	return query
}
