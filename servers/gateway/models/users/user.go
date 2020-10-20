package users

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/badoux/checkmail"
	"golang.org/x/crypto/bcrypt"
)

//gravatarBasePhotoURL is the base URL for Gravatar image requests.
//See https://id.gravatar.com/site/implement/images/ for details
const gravatarBasePhotoURL = "https://www.gravatar.com/avatar/"

//bcryptCost is the default bcrypt cost to use when hashing passwords
var bcryptCost = 13

//User represents a user account in the database
type User struct {
	ID        int64  `json:"id"`
	Email     string `json:"-"` //never JSON encoded/decoded
	PassHash  []byte `json:"-"` //never JSON encoded/decoded
	UserName  string `json:"userName"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	PhotoURL  string `json:"photoURL"`
}

//Credentials represents user sign-in credentials
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

//NewUser represents a new user signing up for an account
type NewUser struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	PasswordConf string `json:"passwordConf"`
	UserName     string `json:"userName"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
}

//Updates represents allowed updates to a user profile
type Updates struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

//Validate validates the new user and returns an error if
//any of the validation rules fail, or nil if its valid
func (nu *NewUser) Validate() error {
	//TODO: validate the new user according to these rules:
	//- Email field must be a valid email address (hint: see mail.ParseAddress)
	//- Password must be at least 6 characters
	//- Password and PasswordConf must match
	//- UserName must be non-zero length and may not contain spaces
	//use fmt.Errorf() to generate appropriate error messages if
	//the new user doesn't pass one of the validation rules

	err := checkmail.ValidateFormat(nu.Email)
	if err != nil {
		fmt.Errorf("Invalid user email address.")
		return errors.New("Invalid user email address.")
	}

	if len(nu.Password) < 6 {
		fmt.Errorf("Password must be at least 6 characters.")
		return errors.New("Password must be at least 6 characters.")
	}

	if nu.Password != nu.PasswordConf {
		fmt.Errorf("Password and confirmed passwords must match.")
		return errors.New("Password and confirmed passwords must match.")
	}

	if len(nu.UserName) <= 0 || strings.Contains(nu.UserName, " ") {
		fmt.Errorf("UserName must not contain spaces and not be zero length.")
		return errors.New("UserName must not contain spaces and not be zero length.")
	}

	return nil
}

//ToUser converts the NewUser to a User, setting the
//PhotoURL and PassHash fields appropriately
func (nu *NewUser) ToUser() (*User, error) {
	//TODO: call Validate() to validate the NewUser and
	//return any validation errors that may occur.
	//if valid, create a new *User and set the fields
	//based on the field values in `nu`.
	//Leave the ID field as the zero-value; your Store
	//implementation will set that field to the DBMS-assigned
	//primary key value.
	//Set the PhotoURL field to the Gravatar PhotoURL
	//for the user's email address.
	//see https://en.gravatar.com/site/implement/hash/
	//and https://en.gravatar.com/site/implement/images/

	//TODO: also call .SetPassword() to set the PassHash
	//field of the User to a hash of the NewUser.Password

	err := nu.Validate()
	if err != nil {
		return nil, err
	}

	// Format the email to md5 hash for gravatar
	formattedEmail := nu.Email
	formattedEmail = strings.TrimSpace(formattedEmail)
	formattedEmail = strings.ToLower(formattedEmail)

	// actually md5 hash it
	m := md5.New()
	m.Write([]byte(formattedEmail))
	emailHash := hex.EncodeToString(m.Sum(nil))

	gravatarPhotoURL := append([]byte(gravatarBasePhotoURL), []byte(emailHash)...)

	newUser := User{
		ID:        0,
		Email:     nu.Email,
		PassHash:  nil,
		UserName:  nu.UserName,
		FirstName: nu.FirstName,
		LastName:  nu.LastName,
		PhotoURL:  string(gravatarPhotoURL),
	}
	newUser.SetPassword(nu.Password)

	return &newUser, nil
}

//FullName returns the user's full name, in the form:
// "<FirstName> <LastName>"
//If either first or last name is an empty string, no
//space is put between the names. If both are missing,
//this returns an empty string
func (u *User) FullName() string {
	//TODO: implement according to comment above
	if len(u.FirstName) == 0 || len(u.LastName) == 0 {
		// if they're both empty it'll go into this branch anyways and work
		return u.FirstName + u.LastName
	} else {
		return u.FirstName + " " + u.LastName
	}
}

//SetPassword hashes the password and stores it in the PassHash field
func (u *User) SetPassword(password string) error {
	//TODO: use the bcrypt package to generate a new hash of the password
	//https://godoc.org/golang.org/x/crypto/bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 13)
	if err != nil {
		fmt.Printf("Encountered error generating bcrypt hash: %v\n", err)
		return errors.New("Error generating bcrypt hash.")
	}

	u.PassHash = hash

	return nil
}

//Authenticate compares the plaintext password against the stored hash
//and returns an error if they don't match, or nil if they do
func (u *User) Authenticate(password string) error {
	//TODO: use the bcrypt package to compare the supplied
	//password with the stored PassHash
	//https://godoc.org/golang.org/x/crypto/bcrypt
	if err := bcrypt.CompareHashAndPassword(u.PassHash, []byte(password)); err != nil {
		return errors.New("Password doesn't match the stored hash.")
	}

	return nil
}

//ApplyUpdates applies the updates to the user. An error
//is returned if the updates are invalid
func (u *User) ApplyUpdates(updates *Updates) error {
	//TODO: set the fields of `u` to the values of the related
	//field in the `updates` struct
	if len(updates.FirstName) == 0 && len(updates.LastName) == 0 {
		return errors.New("Invalid first and last names.")
	} else {
		u.FirstName = updates.FirstName
		u.LastName = updates.LastName
	}
	return nil
}
