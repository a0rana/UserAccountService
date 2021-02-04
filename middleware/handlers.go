package middleware

import (
	"UserAccountService/models" // models package where User schema is defined
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/json" // package to encode and decode the json into struct and vice versa
	"errors"
	"fmt"
	"github.com/allegro/bigcache"
	_ "github.com/allegro/bigcache"
	"github.com/joho/godotenv" // package used to read the .env file
	_ "github.com/lib/pq"      // postgres golang driver
	"log"
	"math"
	"net/http" // used to access the request and response object of the api
	"os"       // used to read the environment variable
	"strings"
	"time"
)

//cache variable
var cache *bigcache.BigCache

//response format for Credit
type responseCredit struct {
	ID      uint64 `json:"id,omitempty"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

//response format for Debit
type responseDebit struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

//response format for Activity
type responseActivity struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// create connection with postgres db
func createConnection() *sql.DB {
	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DBNAME"), os.Getenv("POSTGRES_SSLMODE"))

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

//create cache instance and return BigCache type
func createCache() *bigcache.BigCache {
	if cache == nil {
		var initErr error
		cache, initErr = bigcache.NewBigCache(bigcache.DefaultConfig(10 * time.Minute))
		if initErr != nil {
			log.Fatalf("Error creating cache %v", initErr)
		}
	}
	return cache
}

// Fetches activity of the user's debits and credits
func GetAllTransactions(w http.ResponseWriter, r *http.Request) {
	cache = createCache()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var user models.User
	var res responseActivity

	//used for API pagination using limit and afterid
	limit := r.FormValue("limit")
	afterId := r.FormValue("afterid")
	//used for creating the key in the cache
	url := r.URL.String()

	fmt.Println(fmt.Sprint("URL: ", url, ", param limit: ", limit, ", param afterid: ", afterId))

	// decode the json request to user
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		res = responseActivity{
			Success: false,
			Message: fmt.Sprint("Unable to process the user's transaction history request. ", err.Error()),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(res)
		return
	}

	//check for an entry in cache first to reduce database load, format of the key is "userid_relative api url"
	entry, cacheErr := cache.Get(fmt.Sprint(user.UserId, "_", url))
	if cacheErr == nil {
		fmt.Println("Found key in cache: ", fmt.Sprint(user.UserId, "_", url))
		json.NewEncoder(w).Encode(decodeToUserActivity(entry))
	} else {
		// get all the activities from the db
		activities, err := getAllActivities(user, limit, afterId)

		if err != nil {
			res = responseActivity{
				Success: false,
				Message: fmt.Sprint("Unable to process the user's transaction history request. ", err.Error()),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(res)
			return
		}
		//set cache back using the key, for improving latency on subsequent calls
		cache.Set(fmt.Sprint(user.UserId, "_", url), encodeToBytes(activities))
		fmt.Println("Setting cache with key: ", fmt.Sprint(user.UserId, "_", url))

		if len(activities) == 0 {
			res = responseActivity{
				Success: true,
				Message: fmt.Sprint("Cannot find any transaction history for given user. "),
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(res)
			return
		}
		// send all the users as response
		json.NewEncoder(w).Encode(activities)
	}
}

// CreateUserCredit create a user-credit in the postgres db
func CreateUserCredit(w http.ResponseWriter, r *http.Request) {
	// Allow all origin to handle cors issue
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// create an empty user of type models.User
	var userCredit models.UserCredit
	var res responseCredit

	// decode the json request to user
	err := json.NewDecoder(r.Body).Decode(&userCredit)

	if err != nil {
		res = responseCredit{
			ID:      0,
			Success: false,
			Message: fmt.Sprint("Unable to process the user's credit. ", err.Error()),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(res)
		return
	}

	// call insert user function and pass the user
	insertID, err := insertUserCredit(userCredit)

	if err != nil {
		res = responseCredit{
			ID:      insertID,
			Success: false,
			Message: fmt.Sprint("Unable to process the user's credit. ", err.Error()),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(res)
		return
	}

	fmt.Println("\nCalling invalidate cache from CreateUserCredit")
	//invalidate the cache as new credit has been processed
	invalidateCache(userCredit.UserId)

	// format a response object
	res = responseCredit{
		ID:      insertID,
		Success: true,
		Message: "User credit created successfully",
	}

	// send the response
	json.NewEncoder(w).Encode(res)
}

//Process debit transaction for a user and log same in the activity table for future reporting
func CreateUserDebit(w http.ResponseWriter, r *http.Request) {
	// Allow all origin to handle cors issue
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// create an empty user of type models.User
	var userDebit models.UserDebit

	var res responseDebit

	// decode the json request to user
	err := json.NewDecoder(r.Body).Decode(&userDebit)

	if err != nil {
		res = responseDebit{
			Success: false,
			Message: fmt.Sprint("Unable to process the user's debit. ", err.Error()),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(res)
		return
	}

	// call insert debit function and pass the user
	err = insertUserDebit(userDebit)

	if err != nil {
		res = responseDebit{
			Success: false,
			Message: fmt.Sprint("Unable to process user's debit request. ", err.Error()),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(res)
		return
	}

	fmt.Println("\nCalling invalidate cache from CreateUserDebit")
	//invalidate the cache as new debit has been processed
	invalidateCache(userDebit.UserId)

	// format a response object
	res = responseDebit{
		Success: true,
		Message: "User debit has been processed successfully",
	}
	// send the response
	json.NewEncoder(w).Encode(res)
}

//------------------------- handler functions ---------------------

//get all activities for the user
func getAllActivities(user models.User, limit string, afterId string) ([]models.UserActivity, error) {
	// create the postgres db connection
	db := createConnection()

	// close the db connection
	defer db.Close()

	var activities []models.UserActivity
	var sqlStatement string

	if len(afterId) == 0 && len(limit) == 0 {
		sqlStatement = `SELECT userid, created, iscredit, amount FROM tbl_Activity WHERE userid=$1 ORDER BY iscredit DESC, created ASC`
	}
	if len(afterId) > 0 && len(limit) > 0 {
		sqlStatement = fmt.Sprint(`SELECT userid, created, iscredit, amount FROM tbl_Activity WHERE userid=$1 AND tranid > `, afterId, ` ORDER BY iscredit DESC, created ASC LIMIT `, limit)
	}
	if len(afterId) > 0 && len(limit) == 0 {
		sqlStatement = fmt.Sprint(`SELECT userid, created, iscredit, amount FROM tbl_Activity WHERE userid=$1 AND tranid > `, afterId, ` ORDER BY iscredit DESC, created ASC`)
	}
	if len(limit) > 0 && len(afterId) == 0 {
		sqlStatement = fmt.Sprint(`SELECT userid, created, iscredit, amount FROM tbl_Activity WHERE userid=$1 ORDER BY iscredit DESC, created ASC LIMIT `, limit)
	}
	// execute the sql statement
	rows, err := db.Query(sqlStatement, user.UserId)

	if err != nil {
		return activities, errors.New(fmt.Sprint("Unable to execute the query. ", err.Error()))
	}

	// close the statement
	defer rows.Close()

	// iterate over the rows
	for rows.Next() {
		var userActivity models.UserActivity

		// unmarshal the row object to user
		err = rows.Scan(&userActivity.UserId, &userActivity.Created, &userActivity.IsCredit, &userActivity.Amount)

		if err != nil {
			return activities, errors.New(fmt.Sprint("Unable to scan the row. ", err.Error()))
		}

		// append the user in the users slice
		activities = append(activities, userActivity)

	}

	// return empty user on error
	return activities, err
}

// inserts credit in the DB
func insertUserCredit(userCredit models.UserCredit) (uint64, error) {
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
		return 0, errors.New(err.Error())
	}
	err = tx.QueryRow(userCreditSqlStatement, userCredit.UserId, userCredit.Amount, userCredit.TransactionType,
		userCredit.Priority, userCredit.Expiry).Scan(&userCreditId)

	if err != nil {
		tx.Rollback()
		return 0, errors.New(err.Error())
	}

	// The next query is handled similarly
	_, err = tx.ExecContext(ctx, activitySqlStatement, userCredit.UserId, true, userCredit.Amount, userCreditId)
	if err != nil {
		tx.Rollback()
		return 0, errors.New(err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return 0, errors.New(err.Error())
	}

	fmt.Printf("Inserted a single record with id: %v and logged activity", userCreditId)

	// return the inserted id
	return userCreditId, err
}

//process debit and insert transaction in the activity table
func insertUserDebit(userDebit models.UserDebit) error {
	if userDebit.Amount <= 0.0 {
		return errors.New("please provide debit amount greater than zero")
	}
	// create the postgres db connection
	db := createConnection()

	// close the db connection
	defer db.Close()

	var rollbackError error
	// create the insert sql query
	// returning userid will return the id of the inserted user
	userCreditSqlStatement := `SELECT userid, usercreditid, amount, transactiontype, priority, expiry FROM tbl_UserCredits WHERE userid=$1 AND isexpired=false AND amount>0 ORDER BY priority DESC`

	// Create a new context, and begin a transaction
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New(err.Error())
	}
	var rows *sql.Rows
	rows, err = tx.Query(userCreditSqlStatement, userDebit.UserId)

	if err != nil {
		if rollbackError = tx.Rollback(); rollbackError != nil {
			return errors.New(fmt.Sprint("unable to rollback. ", rollbackError.Error()))
		}
		return errors.New(err.Error())
	}
	defer rows.Close()

	//create a slice to keep track of available credit(s) to consume
	m := make([]models.UserCredit, 0)
	var hasExpiredCredits bool

	for rows.Next() {
		var credit models.UserCredit
		if err := rows.Scan(&credit.UserId, &credit.UserCreditId, &credit.Amount, &credit.TransactionType, &credit.Priority, &credit.Expiry); err != nil {
			if rollbackError = tx.Rollback(); rollbackError != nil {
				return errors.New(fmt.Sprint("unable to rollback. ", rollbackError.Error()))
			}
			return errors.New(err.Error())
		}
		//check for expiry of the credit
		var t time.Time
		t, err = time.Parse(time.RFC3339, credit.Expiry)
		if err != nil {
			if rollbackError = tx.Rollback(); rollbackError != nil {
				return errors.New(fmt.Sprint("unable to rollback. ", rollbackError.Error()))
			}
			return errors.New(err.Error())
		}
		//if the expiry on credit is before or equal to current datetime, then ignore it
		if t.Before(time.Now()) || t.Equal(time.Now()) {
			hasExpiredCredits = true
			continue
		}
		m = append(m, credit)
	}

	if len(m) == 0 && hasExpiredCredits {
		if rollbackError = tx.Rollback(); rollbackError != nil {
			return errors.New(fmt.Sprint("unable to rollback. ", rollbackError.Error()))
		}
		return errors.New("some or all the credits have expired for the given user, cannot process further debits. please allocate new credit(s) for the user to resolve this issue")
	}

	if len(m) == 0 {
		if rollbackError = tx.Rollback(); rollbackError != nil {
			return errors.New(fmt.Sprint("unable to rollback. ", rollbackError.Error()))
		}
		return errors.New("trying to make a debit call before any credits are transacted for the given user. please allocate new credit(s) for the user to resolve this issue")
	}

	fmt.Println("Debug | insertUserDebit | data in the slice: ", m)

	canConsume, credits := canConsumeCredits(userDebit, m)

	fmt.Println("Debug | insertUserDebit | can consume: ", canConsume)
	fmt.Println("Debug | insertUserDebit | credits: ", credits)

	if !canConsume {
		if rollbackError = tx.Rollback(); rollbackError != nil {
			return errors.New(fmt.Sprint("unable to rollback. ", rollbackError.Error()))
		}
		return errors.New("cannot debit more amount than currently present as credit for the given user. please either create more credits or reduce the debit amount to resolve this issue")
	}

	var stmt *sql.Stmt
	stmt, err = tx.PrepareContext(ctx, `UPDATE tbl_UserCredits SET amount=$1, updated=(NOW() AT TIME ZONE 'UTC') WHERE userid=$2 AND usercreditid=$3`)
	if err != nil {
		return errors.New(err.Error())
	}
	defer stmt.Close()

	for _, credit := range credits {
		if _, err = stmt.ExecContext(ctx, credit.Amount, credit.UserId, credit.UserCreditId); err != nil {
			if rollbackError = tx.Rollback(); rollbackError != nil {
				return errors.New(fmt.Sprint("unable to rollback. ", rollbackError.Error()))
			}
			return errors.New(err.Error())
		}
	}

	stmt, err = tx.PrepareContext(ctx, `INSERT INTO tbl_Activity(userid, iscredit, amount, usercreditid) VALUES($1, $2, $3, $4)`)
	if err != nil {
		return errors.New(err.Error())
	}
	defer stmt.Close()

	for _, credit := range credits {
		if credit.Consumed == 0.0 {
			continue
		}
		if _, err = stmt.ExecContext(ctx, credit.UserId, false, credit.Consumed, credit.UserCreditId); err != nil {
			if rollbackError = tx.Rollback(); rollbackError != nil {
				return errors.New(fmt.Sprint("unable to rollback. ", rollbackError.Error()))
			}
			return errors.New(err.Error())
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.New(err.Error())
	}

	fmt.Printf("Debit request processed successfully")

	return err
}

//function containing core logic to process debit from multiple credits based on priority and availability
func canConsumeCredits(userDebit models.UserDebit, m []models.UserCredit) (bool, []models.UserCredit) {
	//processedCredits := make([]models.UserCredit, 0)
	debitAmount := userDebit.Amount
	totalAmount := getTotalAmountInUserCredits(m)

	remainingAmount := userDebit.Amount

	if debitAmount > totalAmount {
		return false, m
	}
	//loop to consume credit amount(when one or more credits are involved)
	/*
	 * It handles three cases:
	 *   (i) First credit in map can fulfill debit amount
	 *   (ii) All credits in map can fulfill debit amount
	 *   (iii) Any credit in between is consumed partially to achieve debit amount
	 */
	for i, credit := range m {
		//consume fully
		if remainingAmount >= credit.Amount {
			remainingAmount = remainingAmount - credit.Amount
			credit.Consumed = credit.Amount
			credit.Amount = 0.0
			m[i] = credit

			//check for case(ii)
			if remainingAmount == 0 {
				break
			}
		} else {
			//this will be the last credit that needs to consumed partially, so consume and break from loop
			credit.Amount = math.Abs(remainingAmount - credit.Amount)
			credit.Consumed = remainingAmount
			m[i] = credit
			break
		}
	}
	return true, m
}

//function to calculate total amount present in credits(expired one's are already filtered out)
func getTotalAmountInUserCredits(m []models.UserCredit) float64 {
	totalAmount := 0.0
	for _, credit := range m {
		totalAmount = totalAmount + credit.Amount
	}
	return totalAmount
}

//function to convert UserActivity slice to byte slice
func encodeToBytes(activity []models.UserActivity) []byte {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(activity)
	if err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

//function to convert []byte back to the []models.UserActivity
func decodeToUserActivity(s []byte) []models.UserActivity {
	var activities []models.UserActivity
	dec := gob.NewDecoder(bytes.NewReader(s))
	err := dec.Decode(&activities)
	if err != nil {
		log.Fatal(err)
	}
	return activities
}

//function to invalidate the cache whenever we receive a POST call for credit or debit
//as we need to pull the latest activities in subsequent transactions call
func invalidateCache(user string) bool {
	if len(user) == 0 {
		return false
	}
	cache = createCache()
	//create iterator for keys and values in cache
	iterator := cache.Iterator()
	for iterator.SetNext() {
		current, err := iterator.Value()
		if err != nil {
			return false
		}
		//if cache key contains specific user id then remove it
		if strings.HasPrefix(current.Key(), user) {
			fmt.Println(fmt.Sprint("debug | invalidateCache | user: ", user, " removing key: ", current.Key()))
			cache.Delete(current.Key())
			continue
		}
	}
	return true
}
