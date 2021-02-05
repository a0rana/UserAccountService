package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/a0rana/UserAccountService/models"
	"github.com/jasonlvhit/gocron"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // postgres golang driver
	"log"
	"os"
)

func main() {
	// Do jobs without params
	s := gocron.NewScheduler()
	//run this job twice every day
	s.Every(12).Hours().Do(updateExpiredCredits)
	<-s.Start()
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

//update isexpired attribute to true for expired user credits
func updateExpiredCredits() error {
	// create the postgres db connection
	db := createConnection()

	// close the db connection
	defer db.Close()

	var rollbackError error
	// create the insert sql query
	// returning userid will return the id of the inserted user
	userCreditSqlStatement := `SELECT userid, usercreditid, amount, transactiontype, priority, expiry FROM tbl_UserCredits WHERE expiry<=(NOW() AT TIME ZONE 'UTC')`

	// Create a new context, and begin a transaction
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return errors.New(err.Error())
	}
	var rows *sql.Rows
	rows, err = tx.Query(userCreditSqlStatement)

	if err != nil {
		if rollbackError = tx.Rollback(); rollbackError != nil {
			return errors.New(fmt.Sprint("unable to rollback. ", rollbackError.Error()))
		}
		return errors.New(err.Error())
	}
	defer rows.Close()

	//create a slice to keep track of available credit(s) to consume
	m := make([]models.UserCredit, 0)

	for rows.Next() {
		var credit models.UserCredit
		if err := rows.Scan(&credit.UserId, &credit.UserCreditId, &credit.Amount, &credit.TransactionType, &credit.Priority, &credit.Expiry); err != nil {
			if rollbackError = tx.Rollback(); rollbackError != nil {
				return errors.New(fmt.Sprint("unable to rollback. ", rollbackError.Error()))
			}
			return errors.New(err.Error())
		}
		m = append(m, credit)
	}

	var stmt *sql.Stmt
	stmt, err = tx.PrepareContext(ctx, `UPDATE tbl_UserCredits SET isexpired=true, updated=(NOW() AT TIME ZONE 'UTC') WHERE userid=$1 AND usercreditid=$2`)
	if err != nil {
		return errors.New(err.Error())
	}
	defer stmt.Close()

	for _, credit := range m {
		if _, err = stmt.ExecContext(ctx, credit.UserId, credit.UserCreditId); err != nil {
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

	fmt.Printf("Credit expiry job completed, updated %d rows in tbl_UserCredits", len(m))

	return err
}
