package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/UW-Info-441-Winter-Quarter-2020/homework-jsm209/servers/gateway/handlers"
	"github.com/UW-Info-441-Winter-Quarter-2020/homework-jsm209/servers/gateway/indexes"
	"github.com/UW-Info-441-Winter-Quarter-2020/homework-jsm209/servers/gateway/models/users"
	"github.com/UW-Info-441-Winter-Quarter-2020/homework-jsm209/servers/gateway/sessions"
	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/streadway/amqp"
)

func main() {

	// read addr environment variable
	addr := os.Getenv("ADDR")

	// if empty, default to ":443", the standard port for HTTPS
	if len(addr) == 0 {
		addr = ":443"
	}

	// read some more environmental variables
	tlsCertPath := os.Getenv("TLSCERT")
	tlsKeyPath := os.Getenv("TLSKEY")
	sessionKey := os.Getenv("SESSIONKEY")
	redisAddr := os.Getenv("REDISADDR")
	dsn := "root:password@tcp(mysqldemo:3306)/users"
	//dsn := fmt.Sprintf("root:databasepassword@tcp(mysql:3306)/users", "password")
	//api.infoclass.me
	if len(tlsCertPath) == 0 && len(tlsKeyPath) == 0 {
		log.Fatal("Requires both a TLSCERT and a TLSKEY environmental variables")
	}

	// creating a new redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	// using it to create a new redisstore
	redisStore := sessions.NewRedisStore(redisClient, time.Hour)

	// open the sql database with the dsn
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("error opening database: %v\n", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	// check db conneciton
	if err := sqlDB.Ping(); err != nil {
		fmt.Printf("error pinging database: %v\n", err)
	} else {
		fmt.Printf("successfully connected!\n")
	}

	// initializing user store
	userStore := users.NewSQLStore(sqlDB, indexes.NewTrieNode())

	// Add existing users to the trie in the user store
	userStore.AddAllUsersToTrie()

	// connect to rabbitMQ server
	conn, err2 := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err2, "Failed to connect to RabbitMQ")
	defer conn.Close()

	// also open up a channel
	ch, err3 := conn.Channel()
	failOnError(err3, "Failed to open a channel")
	defer ch.Close()

	// also open up a queue with that channel,
	// and connect it to the same queue the sender sends to
	// the sender also creates a queue- it's ok to have it twice
	// RabbitMQ is smart enough to only create it if it doesn't exist already.

	q, err4 := ch.QueueDeclare(
		"messages", // name
		true,       // durable (queue stored in local storage in case rabbitMQ crashes)
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	failOnError(err4, "Failed to declare a queue")

	fmt.Printf("connected successfully to rabbitmq")

	// create a socketstore to manage websocket connections
	socketHandler := handlers.NewSocketStore()

	// start a go routine to constantly consume from the queue

	// create a new context handler
	contextHandler := handlers.HandlerContext{
		Key:          sessionKey,
		SessionStore: redisStore,
		UserStore:    userStore,
	}

	// Microservice related environmental variables
	messageAddrs := strings.Split(os.Getenv("MESSAGEADDR"), ",")
	summaryAddrs := strings.Split(os.Getenv("SUMMARYADDR"), ",")

	fmt.Print(messageAddrs)
	fmt.Print(summaryAddrs)

	// Converting address strings to urls
	messageUrls := []*url.URL{}
	summaryUrls := []*url.URL{}

	for _, addr := range messageAddrs {
		url, err := url.Parse(addr)
		if err != nil {
			fmt.Printf("Provided URL address is invalid: %v\n", err)
		}
		fmt.Printf("Messaging address: %v", url)
		messageUrls = append(messageUrls, url)
	}

	for _, addr := range summaryAddrs {
		url, err := url.Parse(addr)
		if err != nil {
			fmt.Printf("Provided URL address is invalid: %v\n", err)
		}
		fmt.Printf("Summary address: %v", url)
		summaryUrls = append(summaryUrls, url)
	}

	// Making proxies for microservices
	messageProxy := &httputil.ReverseProxy{Director: CustomDirector(messageUrls, contextHandler)}
	summaryProxy := &httputil.ReverseProxy{Director: CustomDirector(summaryUrls, contextHandler)}

	// create a new mux
	mux := http.NewServeMux()

	// HANDLERS
	// tell mux to call handlers.SummaryHandler function
	// when "/v1/summary" URL path is requested
	// mux.HandleFunc("/v1/summary", handlers.SummaryHandler)

	// more handlers
	mux.HandleFunc("/v1/users", contextHandler.UsersHandler)
	mux.HandleFunc("/v1/users/", contextHandler.SpecificUserHandler)
	mux.HandleFunc("/v1/sessions", contextHandler.SessionsHandler)
	mux.HandleFunc("/v1/sessions/", contextHandler.SpecificSessionHandler)
	mux.HandleFunc("/v1/users?q=", contextHandler.Search)

	mux.Handle("/v1/channels/", messageProxy)
	mux.Handle("/v1/messages/", messageProxy)
	mux.Handle("/v1/summary", summaryProxy)

	mux.HandleFunc("/v1/ws", socketHandler.WebSocketConnectionHandler)

	// wrap mux in handler
	wrappedMux := handlers.CORS{mux}

	// now start consuming messages
	// register a consumer
	msgs, err := ch.Consume( // change _ to msgs
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack (set false, so that tasks aren't automatically marked for deletion upon being sent)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	go Broadcast(msgs, socketHandler)

	// start web server on address from addr variable
	// and if there is an error, log it
	fmt.Printf("listening on %s...\n", addr)
	log.Fatal(http.ListenAndServeTLS(addr, tlsCertPath, tlsKeyPath, &wrappedMux))
}

// Go routine to constantly consume messages
// Upon consumption, will send the message to
// the appropriate websocket connection
func Broadcast(msgs <-chan amqp.Delivery, socketStore *handlers.SocketStore) {

	// broadcast it to all existing websocket connections
	// that *should hear about that event
	for d := range msgs {
		log.Printf("Recieved a message: %s", d.Body)
		var eventObject map[string]interface{}
		json.Unmarshal([]byte(d.Body), &eventObject)
		if eventObject["userIDs"] != nil {
			// for each user id
			// find the connection belnging to the userid
			// and send them the message if they exist
			allUserIds := eventObject["userIDs"].(map[string]interface{})
			for _, userID := range allUserIds {
				if _, ok := socketStore.Connections[userID.(int64)]; ok { // if websocket exists in map for given id
					//https://godoc.org/github.com/streadway/amqp#Delivery
					err := socketStore.Connections[userID.(int64)].WriteJSON(d.Body)
					if err != nil {
						// if there was a problem sending
						// websocket is invalid, so should be deleted
						delete(socketStore.Connections, userID.(int64))
					}
				}
			}
		} else {
			// send to all channels
			for _, conn := range socketStore.Connections {
				conn.WriteJSON(d.Body)
			}
		}

		d.Ack(false) // acknolwedgew a single delivery, allowing rabbitmq to safely deleted the tasks (it's actually done)
	}
}

// Making a director to attach potential authenticated user and to use HTTP scheme
type Director func(r *http.Request)

func CustomDirector(targets []*url.URL, contextHandler handlers.HandlerContext) Director {
	// getting the authenticated user if there is one
	sessionState := handlers.SessionState{}
	sessionid, err := sessions.NewSessionID(contextHandler.Key)
	if err != nil {
		fmt.Printf("Problem while generating sessionID: %s", err)
	}
	contextHandler.SessionStore.Get(sessionid, sessionState)
	user, err2 := contextHandler.UserStore.GetByID(sessionState.User.ID)
	if err2 != nil {
		fmt.Printf("Problem while getting user from store: %s", err2)
	}

	var counter int32
	counter = 0
	return func(r *http.Request) {
		targ := targets[int(counter)%len(targets)]
		atomic.AddInt32(&counter, 1)

		if user.UserName == "" {
			// would be unset because there is no valid session
			r.Header.Del("X-User")
		} else {
			userData, err := json.Marshal(user)
			if err != nil {
				fmt.Printf("Problem while marshalling user data: %s", err)
			}
			r.Header.Add("X-User", string(userData))
		}
		r.Host = targ.Host
		r.URL.Host = targ.Host
		r.URL.Scheme = targ.Scheme
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
