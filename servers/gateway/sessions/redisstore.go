package sessions

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis"
)

//RedisStore represents a session.Store backed by redis.
type RedisStore struct {
	//Redis client used to talk to redis server.
	Client *redis.Client
	//Used for key expiry time on redis.
	SessionDuration time.Duration
}

//NewRedisStore constructs a new RedisStore
func NewRedisStore(client *redis.Client, sessionDuration time.Duration) *RedisStore {
	//initialize and return a new RedisStore struct
	myReddisStore := RedisStore{
		Client:          client,
		SessionDuration: sessionDuration,
	}

	return &myReddisStore
}

//Store implementation

//Save saves the provided `sessionState` and associated SessionID to the store.
//The `sessionState` parameter is typically a pointer to a struct containing
//all the data you want to associated with the given SessionID.
func (rs *RedisStore) Save(sid SessionID, sessionState interface{}) error {
	//TODO: marshal the `sessionState` to JSON and save it in the redis database,
	//using `sid.getRedisKey()` for the key.
	value, err := json.Marshal(sessionState)
	if err != nil {
		return errors.New("Problem during marshal of session state.")
	}

	key := sid.getRedisKey()
	err2 := rs.Client.Set(key, value, rs.SessionDuration).Err()
	if err2 != nil {
		return errors.New("Problem adding key and value: " + err2.Error())
	}

	//return any errors that occur along the way.
	return nil
}

//Get populates `sessionState` with the data previously saved
//for the given SessionID
func (rs *RedisStore) Get(sid SessionID, sessionState interface{}) error {
	//TODO: get the previously-saved session state data from redis,
	//unmarshal it back into the `sessionState` parameter
	//and reset the expiry time, so that it doesn't get deleted until
	//the SessionDuration has elapsed.
	key := sid.getRedisKey()
	value, err := rs.Client.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrStateNotFound
		}
		return err
	}

	json.Unmarshal([]byte(value), sessionState)

	err3 := rs.Client.Set(key, value, rs.SessionDuration).Err()
	if err3 != nil {
		return errors.New("Problem adding key and value and resetting expiry time.")
	}

	//for extra-credit using the Pipeline feature of the redis
	//package to do both the get and the reset of the expiry time
	//in just one network round trip!

	return nil
}

//Delete deletes all state data associated with the SessionID from the store.
func (rs *RedisStore) Delete(sid SessionID) error {
	//TODO: delete the data stored in redis for the provided SessionID
	err := rs.Client.Del(sid.getRedisKey()).Err()
	if err != nil {
		return errors.New("Problem with deleting value with given key.")
	}

	return nil
}

//getRedisKey() returns the redis key to use for the SessionID
func (sid SessionID) getRedisKey() string {
	//convert the SessionID to a string and add the prefix "sid:" to keep
	//SessionID keys separate from other keys that might end up in this
	//redis instance
	return "sid:" + sid.String()
}
