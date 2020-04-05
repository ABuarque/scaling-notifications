package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

var queueName string

// redis client
var client *redis.Client

// UserSSESession wraps user id and its message channel
type UserSSESession struct {
	ID             string
	messageChannel chan []byte
}

type dispatchMessage struct {
	UserID  string `json:"userID"`
	Payload string `json:"payload"`
}

// TODO change implemenation to keep number of connections
// and return 503 in case of 85%
var sessions = make(map[string]*UserSSESession)

// it formats the payload according with SSE protocol
func formatSSE(payloadAsString string) []byte {
	event := ""
	payloadLines := strings.Split(payloadAsString, "\n")
	for _, line := range payloadLines {
		event = event + "data: " + line + "\n"
	}
	return []byte(event + "\n")
}

func listenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	userID := r.FormValue("id")
	_, err := client.Set(userID, queueName, 0).Result()
	if err != nil {
		log.Println(fmt.Sprintf("failed to set id %s on redis", userID))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to persist"))
		return
	}
	currentSession := &UserSSESession{
		ID:             userID,
		messageChannel: make(chan []byte),
	}
	sessions[userID] = currentSession
	for {
		select {
		case message := <-currentSession.messageChannel:
			w.Write(formatSSE(string(message)))
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			delete(sessions, userID)
			log.Println(fmt.Sprintf("user %s has disconected", userID))
			return
		}
	}
}

func listenToEvents() {
	pubsub := client.Subscribe(queueName)
	ch := pubsub.Channel()
	for {
		for msg := range ch {
			//fmt.Println(msg.Channel, msg.Payload)
			dm := dispatchMessage{}
			err := json.Unmarshal([]byte(msg.Payload), &dm)
			if err != nil {
				log.Println("problem on parsing message: ", err)
			}
			s := sessions[dm.UserID]
			log.Println("dm: ", dm)
			s.messageChannel <- []byte(dm.Payload)
			log.Println(fmt.Sprintf("published message %s to user %s", dm.Payload, dm.UserID))
		}
	}
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
	queueName = uuid.New().String()
	log.Println("queueName: ", queueName)
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/listen", listenHandler)

	go listenToEvents()

	log.Println("server online at ", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
