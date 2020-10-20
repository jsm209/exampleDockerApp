package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/UW-Info-441-Winter-Quarter-2020/homework-jsm209/servers/gateway/models/users"
	"github.com/UW-Info-441-Winter-Quarter-2020/homework-jsm209/servers/gateway/sessions"
)

//TODO: define a handler context struct that
//will be a receiver on any of your HTTP
//handler functions that need access to
//globals, such as the key used for signing
//and verifying SessionIDs, the session store
//and the user store

type HandlerContext struct {
	Key          string
	SessionStore sessions.Store
	UserStore    *users.SQLStore
}

func (h *HandlerContext) Search(w http.ResponseWriter, r *http.Request) {
	// first check if the user is authenticated
	// 1) get sessionID from token
	token := r.Header.Get("Authorization")
	validSessionID, err5 := sessions.ValidateID(token, h.Key)
	if err5 != nil {
		http.Error(w, "Failed to validate authorization token: "+err5.Error(), 500)
	}

	// 2) get the current session state by getting the sessionState corresponding to our sessionID
	sessionState := SessionState{}
	err10 := h.SessionStore.Get(validSessionID, sessionState)
	if err10 != nil {
		http.Error(w, "Problem with getting the current session: "+err10.Error(), 500)
	}

	// 3) ask redisStore if that session is still valid.
	// if invalid, err != nil
	_, err4 := sessions.GetState(r, h.Key, h.SessionStore, sessionState)
	if err4 != nil {
		http.Error(w, "You're not authorized to do that: "+err4.Error(), http.StatusUnauthorized)
	}

	// check that the query parameter "q" is not empty
	query := r.URL.Query()["q"]
	if len(query) == 0 {
		http.Error(w, "Query parameter cannot be empty.", http.StatusBadRequest)
	}

	// Get first 20 UserIDs
	searchedIDs := h.UserStore.Query(strings.Join(query, ""), 20)

	// Fetch profiles
	fetchedUsers := []*users.User{} // ?
	for _, element := range searchedIDs {
		user, err := h.UserStore.GetByID(element)
		if err == nil {
			fetchedUsers = append(fetchedUsers, user)
		}
	}

	// Order by userName ascending (A to Z)
	sort.Slice(fetchedUsers, func(i, j int) bool {
		return fetchedUsers[i].UserName < fetchedUsers[j].UserName
	})

	// if all is well up to this point,
	// respond to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	myjson, err7 := json.Marshal(fetchedUsers)
	if err7 != nil {
		http.Error(w, "Failed to properly marshal user object to a JSON format: "+err7.Error(), 500)
	}
	w.Write(myjson)

}

func (h *HandlerContext) UsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		http.Error(w, "Request body must be in JSON.", http.StatusUnsupportedMediaType)
	}

	var potentialUser users.NewUser

	// reading out request body into a user struct
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "Failed to properly read request body: "+err.Error(), 500)
	}

	err2 := json.Unmarshal(body, &potentialUser)
	if err2 != nil {
		http.Error(w, "Failed to unmarshal request body into a struct: "+err2.Error(), 500)
	}

	err3 := potentialUser.Validate()
	if err3 != nil {
		http.Error(w, "Failed to validate user: "+err3.Error(), 500)
	}

	validUser, err4 := potentialUser.ToUser()
	if err4 != nil {
		http.Error(w, "Failed to convert valid user to a user object: "+err4.Error(), 500)
	}

	// insert valid user
	insertedUser, err5 := h.UserStore.Insert(validUser)
	if err5 != nil {
		http.Error(w, "Failed to insert user object to the store: "+err5.Error(), 500)
	}

	// Add user's username, firstname, lastname, to the trie
	h.UserStore.AddUserToTrie(insertedUser)

	// make a new sessionState for the valid user
	newSessionState := SessionState{
		Curtime: time.Now(),
		User:    *insertedUser,
	}

	// make a new sessionID for valid user
	mySessionID, err8 := sessions.NewSessionID(h.Key)
	if err8 != nil {
		http.Error(w, "Failed to create a new sessionID: "+err8.Error(), 500)
	}

	// begin new session for user
	err6 := h.SessionStore.Save(mySessionID, newSessionState)
	if err6 != nil {
		http.Error(w, "Failed to save a new session for the user: "+err6.Error(), 500)
	}

	// if all is well up to this point,
	// respond to the client
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cookie", mySessionID.String())
	w.WriteHeader(http.StatusCreated)
	myjson, err7 := json.Marshal(insertedUser)
	if err7 != nil {
		http.Error(w, "Failed to properly marshal user object to a JSON format: "+err7.Error(), 500)
	}
	w.Write(myjson)
}

func (h *HandlerContext) SpecificUserHandler(w http.ResponseWriter, r *http.Request) {
	// first check if the user is authenticated

	// 1) get sessionID from token
	token := r.Header.Get("Authorization")
	validSessionID, err := sessions.ValidateID(token, h.Key)
	if err != nil {
		http.Error(w, "Failed to validate authorization token: "+err.Error(), 500)
	}

	// 2) get the current session state by getting the sessionState corresponding to our sessionID
	sessionState := SessionState{}
	err10 := h.SessionStore.Get(validSessionID, sessionState)
	if err10 != nil {
		http.Error(w, "Problem with getting the current session: "+err10.Error(), 500)
	}

	// 3) ask redisStore if that session is still valid.
	// if invalid, err != nil
	_, err4 := sessions.GetState(r, h.Key, h.SessionStore, sessionState)
	if err4 != nil {
		http.Error(w, "You're not authorized to do that: "+err4.Error(), http.StatusUnauthorized)
	}

	if r.Method == http.MethodGet {
		// get requested user id, parsing url path
		userID := r.URL.Query()["UserID"]

		// parsing int
		userIDint, _ := strconv.ParseInt(strings.Join(userID, ""), 10, 64)

		// get user from store
		user, err2 := h.UserStore.GetByID(int64(userIDint))
		if err2 != nil {
			http.Error(w, "Failed to get user from store from their ID: "+err2.Error(), 415)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("User with that ID cannot be found."))
		}

		// user at this point was found in the store
		// if all is well up to this point,
		// respond to the client
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		myjson, err3 := json.Marshal(user)
		if err3 != nil {
			http.Error(w, "Failed to properly marshal user object to a JSON format: "+err3.Error(), 500)
		}
		w.Write(myjson)

	} else if r.Method == http.MethodPatch {
		// get requested user id, parsing url path
		userID := r.URL.Query()["UserID"]

		// parsing int
		userIDint, _ := strconv.ParseInt(strings.Join(userID, ""), 10, 64)

		// if userID doesn't match "me"
		// TODO: or the currently authorized user
		// tell them they can't do that
		if strings.Join(userID, "") != "me" || sessionState.User.ID != userIDint {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

		// check if response body is JSON
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "Request body must be in JSON.", http.StatusUnsupportedMediaType)
		}

		// get user from store
		user, err := h.UserStore.GetByID(int64(userIDint))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			http.Error(w, "User with that ID cannot be found: "+err.Error(), http.StatusNotFound)
		}

		var userUpdates users.Updates

		// reading out request body into a user struct
		body, err2 := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err2 != nil {
			http.Error(w, "Failed to read request body: "+err2.Error(), 500)
		}

		// putting the json into an updates struct
		err3 := json.Unmarshal(body, &userUpdates)
		if err3 != nil {
			http.Error(w, "Failed to unmarshal the JSON: "+err3.Error(), 500)
		}

		// applying the updates
		err4 := user.ApplyUpdates(&userUpdates)
		if err4 != nil {
			http.Error(w, "Failed to apply given updates: "+err4.Error(), 500)
		}

		// user at this point was updated.
		// if all is well up to this point,
		// respond to the client
		updatedUser, _ := h.UserStore.GetByID(userIDint)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		myjson, err5 := json.Marshal(updatedUser)
		if err5 != nil {
			http.Error(w, "Failed to remarshal JSON: "+err5.Error(), 500)
		}
		w.Write(myjson)

	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

}

func (h *HandlerContext) SessionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			http.Error(w, "Request body must be in JSON.", http.StatusUnsupportedMediaType)
		}

		var userCredentials users.Credentials

		// reading out request body into a user credentials struct
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, "Failed to properly read request body: "+err.Error(), 500)
		}

		err2 := json.Unmarshal(body, &userCredentials)
		if err2 != nil {
			http.Error(w, "Failed to unmarshal request body into a struct: "+err2.Error(), 500)
		}

		// get the user from the store by email
		user, err3 := h.UserStore.GetByEmail(userCredentials.Email)
		if err3 != nil {
			// pretend to validate, then return an error.
			time.Sleep(1 * time.Second)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		}

		// authenticate the user
		err4 := user.Authenticate(userCredentials.Password)
		if err4 != nil {
			// incorrect password
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		}

		// authorized, so we begin a new session.
		// make a new sessionState for the valid user
		newSessionState := SessionState{
			Curtime: time.Now(),
			User:    *user,
		}

		// make a new sessionID for valid user
		mySessionID, err8 := sessions.NewSessionID(h.Key)
		if err8 != nil {
			http.Error(w, "Failed to create a new sessionID: "+err8.Error(), 500)
		}

		// begin new session for user
		err5 := h.SessionStore.Save(mySessionID, newSessionState)
		if err5 != nil {
			http.Error(w, "Failed to save a new session for the user: "+err5.Error(), 500)
		}

		// if all is well up to this point,
		// respond to the client
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		myjson, err6 := json.Marshal(user)
		if err6 != nil {
			http.Error(w, "Failed to properly marshal user object to a JSON format: "+err6.Error(), 500)
		}
		w.Write(myjson)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HandlerContext) SpecificSessionHandler(w http.ResponseWriter, r *http.Request) {
	// first check if the user is authenticated

	// 1) get sessionID from token
	token := r.Header.Get("Authorization")
	validSessionID, err5 := sessions.ValidateID(token, h.Key)
	if err5 != nil {
		http.Error(w, "Failed to validate authorization token: "+err5.Error(), 500)
	}

	// 2) get the current session state by getting the sessionState corresponding to our sessionID
	sessionState := SessionState{}
	err10 := h.SessionStore.Get(validSessionID, sessionState)
	if err10 != nil {
		http.Error(w, "Problem with getting the current session: "+err10.Error(), 500)
	}

	// 3) ask redisStore if that session is still valid.
	// if invalid, err != nil
	sessionID, err4 := sessions.GetState(r, h.Key, h.SessionStore, sessionState)
	if err4 != nil {
		http.Error(w, "You're not authorized to do that: "+err4.Error(), http.StatusUnauthorized)
	}

	if r.Method == http.MethodDelete {
		if path.Base(r.URL.Path) != "mine" {
			http.Error(w, "You can't do that.", http.StatusForbidden)
		}
		// deletes the session
		err := h.SessionStore.Delete(sessionID)
		if err != nil {
			http.Error(w, "Something went wrong while deleting the session: "+err.Error(), 500)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("signed out"))
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Write a function later to authenticate tokens
