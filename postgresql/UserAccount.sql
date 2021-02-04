CREATE
DATABASE "UserAccount"
    WITH
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'C'
    LC_CTYPE = 'C'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;

CREATE TABLE tbl_Users
(
    userid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fname  VARCHAR(20),
    lname  VARCHAR(20),
    email  VARCHAR(30),
    dob    DATE,
    mobile VARCHAR(10)
);

CREATE TABLE tbl_Activity
(
    userid       UUID REFERENCES tbl_Users (userid),
    tranid       BIGSERIAL,
    created      TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC'),
    iscredit     BOOLEAN DEFAULT TRUE,
    amount       NUMERIC(10, 2) NOT NULL,
    usercreditid BIGINT REFERENCES tbl_UserCredits (usercreditid),
    PRIMARY KEY (userid, tranid)
);

CREATE TABLE tbl_UserCredits
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
);

INSERT INTO tbl_Users(fname, lanme, email, dob, mobile) VALUES('John', 'Doe', 'john.doe@gmail.com', '1987-11-10', '9994447878')
INSERT INTO tbl_Users(fname, lanme, email, dob, mobile) VALUES('Jane', 'Doe', 'jane.doe@gmail.com', '1989-10-09', '9995557878')
INSERT INTO tbl_Users(fname, lanme, email, dob, mobile) VALUES('Jonathan', 'Smith', 'jonathan.smith@gmail.com', '1988-08-09', '8885557878')