package users

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/UW-Info-441-Winter-Quarter-2020/homework-jsm209/servers/gateway/indexes"
)

//RedisStore represents a session.Store backed by redis.
type SQLStore struct {
	// SQL database object
	db *sql.DB

	// Trie for searching users by UserName, FirstName, LastName
	searchIndex *indexes.TrieNode
}

//NewSQLStore constructs a new SQLStore
func NewSQLStore(db *sql.DB, searchIndex *indexes.TrieNode) *SQLStore {
	mySQLStore := SQLStore{
		db:          db,
		searchIndex: searchIndex,
	}
	return &mySQLStore
}

func (ss *SQLStore) Query(prefix string, max int) []int64 {
	return ss.searchIndex.Find(prefix, max)
}

func (ss *SQLStore) AddAllUsersToTrie() error {
	rows, err := ss.db.Query("select * from USERS")
	if err != nil {
		return errors.New("Failed to select all users when adding to trie")
	}

	defer rows.Close()

	users := User{}

	for rows.Next() {
		// Scans row into users struct.
		if err := rows.Scan(&users.ID, &users.Email,
			&users.PassHash, &users.UserName, &users.FirstName,
			&users.LastName, &users.PhotoURL); err != nil {
			return errors.New("Error scanning row.")
		}
		ss.AddUserToTrie(&users)
	}

	return nil
}

func (ss *SQLStore) AddUserToTrie(users *User) {
	id := users.ID
	// make lower case
	// if there are spaces, break it up
	username := strings.Split(strings.ToLower(users.UserName), " ")
	firstname := strings.Split(strings.ToLower(users.FirstName), " ")
	lastname := strings.Split(strings.ToLower(users.LastName), " ")

	for _, element := range username {
		ss.searchIndex.Add(element, id)
	}

	for _, element := range firstname {
		ss.searchIndex.Add(element, id)
	}

	for _, element := range lastname {
		ss.searchIndex.Add(element, id)
	}
}

func (ss *SQLStore) DeleteUserFromTrie(users *User) {
	id := users.ID
	// make lower case
	// if there are spaces, break it up
	firstname := strings.Split(strings.ToLower(users.FirstName), " ")
	lastname := strings.Split(strings.ToLower(users.LastName), " ")

	for _, element := range firstname {
		ss.searchIndex.Remove(element, id)
	}

	for _, element := range lastname {
		ss.searchIndex.Remove(element, id)
	}
}

//GetByID returns the User with the given ID
func (ss *SQLStore) GetByID(id int64) (*User, error) {
	rows, err := ss.db.Query("select id, Email, PassHash, UserName, FirstName, LastName, PhotoURL from USERS")
	if err != nil {
		return nil, errors.New("Failed to query.")
	}

	defer rows.Close()

	users := User{}

	for rows.Next() {
		// Scans row into users struct.
		if err := rows.Scan(&users.ID, &users.Email,
			&users.PassHash, &users.UserName, &users.FirstName,
			&users.LastName, &users.PhotoURL); err != nil {
			return nil, errors.New("Error scanning row.")
		}

		// checks the ID, and if it matches, return that user.
		if users.ID == id {
			return &users, nil
		}
	}

	return nil, errors.New("User with ID not found.")
}

//GetByEmail returns the User with the given email
func (ss *SQLStore) GetByEmail(email string) (*User, error) {
	rows, err := ss.db.Query("select id, Email, PassHash, UserName, FirstName, LastName, PhotoURL from USERS")
	if err != nil {
		return nil, errors.New("Failed to query.")
	}

	defer rows.Close()

	users := User{}

	for rows.Next() {
		if err := rows.Scan(&users.ID, &users.Email,
			&users.PassHash, &users.UserName, &users.FirstName,
			&users.LastName, &users.PhotoURL); err != nil {
			return nil, errors.New("Error scanning row.")
		}

		if users.Email == email {
			return &users, nil
		}
	}
	return nil, nil
}

//GetByUserName returns the User with the given Username
func (ss *SQLStore) GetByUserName(username string) (*User, error) {
	rows, err := ss.db.Query("select id, Email, PassHash, UserName, FirstName, LastName, PhotoURL from USERS")
	if err != nil {
		return nil, errors.New("Failed to query.")
	}

	defer rows.Close()

	users := User{}

	for rows.Next() {
		if err := rows.Scan(&users.ID, &users.Email,
			&users.PassHash, &users.UserName, &users.FirstName,
			&users.LastName, &users.PhotoURL); err != nil {
			return nil, errors.New("Error scanning row.")
		}

		if users.UserName == username {
			return &users, nil
		}
	}

	return nil, nil
}

//Insert inserts the user into the database, and returns
//the newly-inserted User, complete with the DBMS-assigned ID
func (ss *SQLStore) Insert(user *User) (*User, error) {
	insq := "insert into USERS(Email, PassHash, UserName, FirstName, LastName, PhotoURL) values(?,?,?,?,?,?)"
	res, err := ss.db.Exec(insq, user.Email, user.PassHash, user.UserName, user.FirstName, user.LastName, user.PhotoURL)
	if err != nil {
		fmt.Printf("error inserting row: %v\n", err)
		return user, errors.New("Error inserting row: " + err.Error())
	} else {
		id, err := res.LastInsertId()
		if err != nil {
			fmt.Printf("error getting new id: %v\n", err)
			return user, errors.New("Error getting new ID")
		} else {
			user.ID = id
			return user, nil
		}
	}
}

//Update applies UserUpdates to the given user ID
//and returns the newly-updated user
func (ss *SQLStore) Update(id int64, updates *Updates) (*User, error) {
	upsq := "update USERS set firstname = ?, lastname = ?, where id = ?"
	res, err := ss.db.Exec(upsq, updates.FirstName, updates.LastName, id)
	if err != nil {
		return nil, errors.New("Error updating row")
	} else {
		id, err := res.LastInsertId()
		if err != nil {
			return nil, errors.New("Error getting new ID")
		} else {
			updatedUser, err := ss.GetByID(id)
			if err != nil {
				return nil, errors.New("Failed to get new user after updating.")
			}

			// delete user from trie
			ss.DeleteUserFromTrie(updatedUser)

			// add in updated user
			ss.AddUserToTrie(updatedUser)

			return updatedUser, nil
		}
	}
}

//Delete deletes the user with the given ID
func (ss *SQLStore) Delete(id int64) error {
	desq := "delete from USERS where id = ?"
	_, err := ss.db.Exec(desq, id)
	if err != nil {
		return errors.New("Error deleting row, given ID might be invalid.")
	}
	return nil
}
