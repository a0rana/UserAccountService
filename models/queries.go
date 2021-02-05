package models

const (
	UserCreditSelectStatement   string = `SELECT userid, usercreditid, amount, transactiontype, priority, expiry FROM tbl_UserCredits WHERE userid=$1 AND isexpired=false AND amount>0 ORDER BY priority DESC`
	UserCreditInsertStatement   string = `INSERT INTO tbl_UserCredits(userid, amount, transactiontype, priority, expiry) VALUES ($1, $2, $3, $4, $5) RETURNING usercreditid`
	UserActivityInsertStatement string = `INSERT INTO tbl_Activity(userid, iscredit, amount, usercreditid) VALUES ($1, $2, $3, $4)`
	UserCreditUpdateStatement   string = `UPDATE tbl_UserCredits SET amount=$1, updated=(NOW() AT TIME ZONE 'UTC') WHERE userid=$2 AND usercreditid=$3`
)
