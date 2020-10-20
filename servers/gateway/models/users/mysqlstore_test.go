package users

import (
	"regexp"
	"testing"

	"github.com/UW-Info-441-Winter-Quarter-2020/homework-jsm209/servers/gateway/indexes"

	"github.com/DATA-DOG/go-sqlmock"
)

const sqlColumnListNoID = "Email, PassHash, UserName, FirstName, LastName, PhotoURL"
const sqlInsertUser = "insert into USERS(" + sqlColumnListNoID + ") values(?,?,?,?,?,?)"
const sqlGetByID = "select ID," + sqlColumnListNoID + " from USERS where id = ?"
const sqlGetByEmail = "select ID," + sqlColumnListNoID + " from USERS where Email = ?"
const sqlGetByUserName = "select ID," + sqlColumnListNoID + " from USERS where UserName = ?"

func TestInsert(t *testing.T) {
	// =======================
	// Step 1: Create User
	// =======================

	// set up sqlMock, New() returns (*sql.DB, Sqlmock, error)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating sqlmock: %v", err)
	}

	// use the *sql.DB to construct a new sql store
	// remember that the `db` that New() returns conforms to *sql.DB
	sqlStore := NewSQLStore(db, indexes.NewTrieNode()) // <- this might be confusing.
	// All we're doing here is to plug in an "object" that satisfies *sql.DB requirements -- like an ***interface***
	// NewSqlStore wants an *sql.DB to construct the `SqlStore` struct, and sqlmock gives us that object

	// I keep calling them "objects", but strictly speaking there are only "interfaces" and "structs" in golang.
	// Though functionally speaking, you can consider them as the same

	// create new mock user
	signUp := NewUser{
		Email:        "jsm209@uw.edu",
		Password:     "password",
		PasswordConf: "password",
		UserName:     "jsm209",
		FirstName:    "Joshua",
		LastName:     "Maza",
	}
	// server-side validation
	err = signUp.Validate()
	if err != nil {
		t.Fatalf("cannot validate sign up: %v", err)
	}
	// transform that signup struct to an actual profile that we want to save
	profile, err := signUp.ToUser()
	if err != nil {
		t.Fatalf("cannot convert to profile: %v", err)
	}

	// =======================
	// Step 2: Test Insertion
	// =======================

	// must use regexp.QuoteMeta() to escape the regexp
	// special characters in SQL statements
	expectedSQL := regexp.QuoteMeta(sqlInsertUser)

	// simple success case
	var newID int64 = 1

	// expect to exec the following sql statement
	mock.ExpectExec(expectedSQL).
		// with these arguments
		WithArgs(
			profile.Email,
			profile.PassHash,
			profile.UserName,
			profile.FirstName,
			profile.LastName,
			profile.PhotoURL,
		).
		// with these results
		WillReturnResult(sqlmock.NewResult(newID, 1))

	// now execute the insertion
	insertedUser, err := sqlStore.Insert(profile)
	if err != nil {
		t.Fatalf("unexpected error during successful insert: %v", err)
	}
	if insertedUser == nil {
		t.Fatal("nil user returned from insert")
	} else if insertedUser.ID != newID {
		t.Fatalf("incorrect new ID: expected %d but got %d", newID, insertedUser.ID)
	}
}
