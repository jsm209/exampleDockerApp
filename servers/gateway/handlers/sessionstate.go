package handlers

import (
	"time"

	"github.com/UW-Info-441-Winter-Quarter-2020/homework-jsm209/servers/gateway/models/users"
)

//TODO: define a session state struct for this web server
//see the assignment description for the fields you should include
//remember that other packages can only see exported fields!
type SessionState struct {
	Curtime time.Time
	User    users.User
}
