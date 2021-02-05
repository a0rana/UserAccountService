# UserAccountService

Assignment to create a REST API in Go language, which will process credits and debits of a user and will also provide the 
activities for all the transactions happened so far.

**Go Packages used:**
1. BigCache: For caching the responses, in order to reduce latency and database hits.
2. Gorilla/mux: Implements a request router and dispatcher for matching incoming requests to their respective handler.
3. GoDotEnv: Loads env vars from a .env file
4. Pq: Pure Go Postgres driver for the database/sql package.
5. goCron: Golang job scheduling package which lets you run Go functions periodically at pre-determined interval.

**Endpoints Exposed:**
1. POST /credit : To process credit for the user.
2. POST /debit : To process a debit request for the user.
3. GET /transactions : To show user activity containing both credits and debits

**Endpoints Request/Response Format(example):**
1. POST /credit 
   <br/>
   Request: `{"userid":"7507decb-0f2d-4510-8202-c78699ed3153","amount":5,"transactiontype":"Gift Card","priority":5,"expiry":"2021-10-19 10:23:54"}` 
   <br/>
   Response:
   For success scenarios: `{"id":11,"success":true,"message":"User credit created successfully"}`
   For error use cases: `{"success":false,"message":"Unable to process the user's credit."}`

2. POST /debit
   <br/>
   Request: `{"userid":"7507decb-0f2d-4510-8202-c78699ed3153","amount":5}`
   <br/>
   Response:
   For success scenarios: `{"success":true,"message":"User debit has been processed successfully"}`
   For error use cases: `{"success":false,"message":"Unable to process user's debit request."}`
   We are handling expired credits during processing the debits(ignore those) and also we have a scheduled job to mark them as expired.

3. GET /transactions
   <br/>
   Request: `{"userid":"7507decb-0f2d-4510-8202-c78699ed3153"}`
   <br/>
   Response: `[{"userid":"7507decb-0f2d-4510-8202-c78699ed3153","created":"2021-02-04T21:22:18.856783Z","iscredit":true,"amount":5},{"userid":"7507decb-0f2d-4510-8202-c78699ed3153","created":"2021-02-04T21:22:30.202332Z","iscredit":false,"amount":1},{"userid":"7507decb-0f2d-4510-8202-c78699ed3153","created":"2021-02-04T21:22:31.791192Z","iscredit":false,"amount":1}]`
   
   Pagination is supported for this endpoint using limit and afterid params:
   <br/>
   Request URI will look like: /transactions?limit=2&afterid=1
   <br/>
   Above request will provide 2 JSON objects on each call, with tranid > 1. It's the responsibility of the caller to keep track of last tranid sent to the server    for paginating the results. 
   <br/>
   By using pagination and caching together we will be able to reduce load on the database server. We are also invalidating the cache when any credit or debit is posted for a user, so that we can fetch the latest user activities.

**Credit Expiry Job**

Placed in "./scheduledjob/creditexpiryjob.go"
This job uses a cron scheduler to run it twice every day(periodically) and contains logic to mark user credits as expired if expiry date is before or equal to the current datetime(when job runs).
Should be run as a separate standalone project(will need to copy ".env" file and "./models/usercredit.go" model).

**SQL Database used:** PostgreSQL 13.1

SQL script to create database objects included in "./postgresql/useraccount.sql"
Tables:
1. tbl_Users: Containing information of the user, userid is of type uuid.
2. tbl_UserCredits: Holds user credit info, stores updated credits after the debit transaction has been executed.
3. tbl_Activity: Contains history of user credits and debits.

**Assumption/Limitation(s):**
1. REST/JSON API
2. Authentication/Authorization mechanism have not been considered in this assignment.
3. Considering we might need the details about the user credit for which debit was done, the tables in the database are designed in that way.
4. Unit test cases needs to be implemented.
5. Assuming that the PostgreSQL instance will already have two databases:
   
   a. UserAccount: Production database used by the REST API.

   b. TestUserAccount: Sandbox database used for integration testing.

**Setup:**
1. Clone the repo in local.
2. Create ".env" file inside cloned directory with below params:<br/>
POSTGRES_HOST="localhost"<br/>
POSTGRES_PORT="5432"<br/>
POSTGRES_USER="XXXXX"<br/>
POSTGRES_PASSWORD="XXXXX"<br/>
POSTGRES_DBNAME="UserAccount"<br/>
POSTGRES_SSLMODE="disable"

3. Execute "go run main.go" in terminal to start the rest api in the local machine at port 8080
4. Use any REST client(like Postman) to make API calls.


**Running integration test cases:**
1. Clone the repo in local.
2. Make sure to update "POSTGRES_DBNAME" param value in ".env" file to "TestUserAccount"(test database).
3. Run "go test -v" in terminal(in same directory)

**Future enhancement for better performance that can be implemented:**
1) Partitioning table: We can partition the table in future based on userid and created timestamp(for a period of month etc). 
2) Moving user credits to history table: We can move expired user credits to a separate archive table for the given user
   so that the user credit table size does not largely increase.
