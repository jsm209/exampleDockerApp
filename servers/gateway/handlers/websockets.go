package handlers

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"

	"github.com/UW-Info-441-Winter-Quarter-2020/homework-jsm209/servers/gateway/sessions"
	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
)

// A simple store to store all the connections
type SocketStore struct {
	Connections map[int64]*websocket.Conn
	RedisStore  sessions.RedisStore
	lock        sync.Mutex
}

// Constructs a new socketstore
func NewSocketStore() *SocketStore {
	//initialize and return a new RedisStore struct
	mySocketStore := SocketStore{
		Connections: map[int64]*websocket.Conn{},
	}

	return &mySocketStore
}

// Thread-safe method for inserting a connection
func (s *SocketStore) InsertConnection(conn *websocket.Conn, userid int64) {
	s.lock.Lock()
	// insert socket connection
	s.Connections[userid] = conn
	s.lock.Unlock()
}

// Thread-safe method for inserting a connection
func (s *SocketStore) RemoveConnection(userid int64) {
	s.lock.Lock()
	// remove socket connection
	delete(s.Connections, userid)
	s.lock.Unlock()
}

//TODO: add a handler that upgrades clients to a WebSocket connection
//and adds that to a list of WebSockets to notify when events are
//read from the RabbitMQ server. Remember to synchronize changes
//to this list, as handlers are called concurrently from multiple
//goroutines.

// Go has a websocket module called gorilla that can handle this for us
// it returns a connection that we can write to

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// This function's purpose is to reject websocket upgrade requests if the
		// origin of the websockete handshake request is coming from unknown domains.
		// This prevents some random domain from opening up a socket with your server.
		// TODO: make sure you modify this for your HW to check if r.Origin is your host
		return true
	},
}

func (s *SocketStore) WebSocketConnectionHandler(w http.ResponseWriter, r *http.Request) {
	// handle the websocket handshake
	// get auth query string parameter, but only if there isn't a authorization header
	sessionToken := r.Header.Get("Authorization")
	if sessionToken == "" { // header was not set
		sessionToken = r.URL.Query().Get("auth")
	}

	// TODO: use token to get session state
	sessionState := SessionState{}
	sessionid := sessions.SessionID(sessionToken)
	s.RedisStore.Get(sessionid, sessionState)
	if reflect.DeepEqual(sessionState, SessionState{}) { // if sessionState is empty
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to open websocket connection", 401)
	}

	// do something with connection
	// like probably add it to some collection
	// start a goroutine on the connection to read incoming messages
	// for each new websocket, start a goroutine to read incoming messages
	// if you run into an error while reading incoming messages, close the websocket and remove it from the list

	s.InsertConnection(conn, sessionState.User.ID)
	go s.read(conn, sessionState.User.ID)

}

func (s *SocketStore) read(conn *websocket.Conn, userid int64) {
	for { // infinite loop
		var m map[string]interface{}

		err := conn.ReadJSON(&m)
		if err != nil {
			fmt.Println("Error reading json.", err)
			s.RemoveConnection(userid)
			conn.Close()
			break
		}

		fmt.Printf("Got message: %#v\n", m)

		if err = conn.WriteJSON(m); err != nil {
			fmt.Println(err)
		}
	}
}

//TODO: start a goroutine that connects to the RabbitMQ server,
//reads events off the queue, and broadcasts them to all of
//the existing WebSocket connections that should hear about
//that event. If you get an error writing to the WebSocket,
//just close it and remove it from the list
//(client went away without closing from
//their end). Also make sure you start a read pump that
//reads incoming control messages, as described in the
//Gorilla WebSocket API documentation:
//http://godoc.org/github.com/gorilla/websocket

func Updater() {
	// connect to rabbitMQ server
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	// also open up a channel
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// also open up a queue with that channel,
	// and connect it to the same queue the sender sends to
	// the sender also creates a queue- it's ok to have it twice
	// RabbitMQ is smart enough to only create it if it doesn't exist already.
	q, err := ch.QueueDeclare(
		"messages", // name
		true,       // durable (queue stored in local storage in case rabbitMQ crashes)
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// register a consumer
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack (set false, so that tasks aren't automatically marked for deletion upon being sent)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	// broadcast it to all existing websocket connections
	// that *should hear about that event
	for d := range msgs {
		log.Printf("Recieved a message: %s", d.Body)

		d.Ack(false) // acknolwedgew a single delivery, allowing rabbitmq to safely deleted the tasks (it's actually done)
	}

	// if while writing to a websocket, it errors, close it.

	// also start a read pump to read incoming control messages
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
