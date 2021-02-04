# UserAccountService

Assignment to create a REST API in Go language, which will process credits and debits of a user and will also provide the 
activities for all the transactions happened so far.

SQL Database used: PostgreSQL 13.1

Endpoints Exposed:
1. POST /credit : To process credit for the user.
2. POST /debit : To process a debit request for the user.
3. GET /transactions : To show user activity containing both credits and debits

Endpoints Request/Response Format:
1. POST /credit 
   Request: {"userid":"7507decb-0f2d-4510-8202-c78699ed3153","amount":5,"transactiontype":"Gift Card","priority":5,"expiry":"2021-10-19 10:23:54"} 
   Response:
   For success scenarios: {"id":11,"success":true,"message":"User credit created successfully"}
   For error use cases: {"success":false,"message":"Unable to process the user's credit."}

2. POST /debit
   Request: {"userid":"7507decb-0f2d-4510-8202-c78699ed3153","amount":5}
   Response:
   For success scenarios: {"success":true,"message":"User debit has been processed successfully"}
   For error use cases: {"success":false,"message":"Unable to process user's debit request."}

3. GET /transactions
   Request: {"userid":"7507decb-0f2d-4510-8202-c78699ed3153"}
   Response: [{"userid":"7507decb-0f2d-4510-8202-c78699ed3153","created":"2021-02-04T21:22:18.856783Z","iscredit":true,"amount":5},{"userid":"7507decb-0f2d-4510-8202-c78699ed3153","created":"2021-02-04T21:22:30.202332Z","iscredit":false,"amount":1},{"userid":"7507decb-0f2d-4510-8202-c78699ed3153","created":"2021-02-04T21:22:31.791192Z","iscredit":false,"amount":1}]
   
   Pagination is supported for this endpoint using limit and afterid params:
   Request URI will look like: /transactions?limit=2&afterid=1
   Above request will provide 2 JSON objects on each call, with tranid > 1. It's the responsibility of the caller to keep track of last tranid sent to the server for paginating the results.
   
By using pagination and caching together we will be able to reduce load on the database server.
   
Packages imported:
1. BigCache: For caching the responses, in order to reduce latency and database hits.
2. Gorilla/mux: Implements a request router and dispatcher for matching incoming requests to their respective handler.
3. GoDotEnv: Loads env vars from a .env file
4. Pq: Pure Go Postgres driver for the database/sql package.

Assumption/Limitation(s):
1. REST/JSON API
2. Authentication/Authorization mechanism have not been implemented.