package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis"
)

var client *redis.Client

type dispatchMessage struct {
	UserID  string `json:"userID"`
	Payload string `json:"payload"`
}

func dispatchHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.FormValue("id")
	payload := r.FormValue("payload")
	log.Println(fmt.Sprintf("id: %s, payload: %s", userID, payload))
	queueToSend, err := client.Get(userID).Result()
	if err != nil {
		log.Println("error on getting queue name from redis: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Pau\n"))
		return
	}
	m := dispatchMessage{
		UserID:  userID,
		Payload: payload,
	}
	messageAsBytes, err := json.Marshal(m)
	if err != nil {
		log.Println("failed to marshal message: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("pau"))
		return
	}
	err = client.Publish(queueToSend, string(messageAsBytes)).Err()
	if err != nil {
		fmt.Println("problem on publishing: ", err)
	}
	log.Println("published message to queue ", queueToSend)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("done"))
	return
}

func main() {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", redisHost),
		Password: "",
		DB:       0,
	})
	_, err := client.Ping().Result()
	if err != nil {
		log.Fatal("problem on connect ro redis: %q", err)
	}
	log.Println("connected to redis at host: ", redisHost)
	http.HandleFunc("/dispatch", dispatchHandler)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
