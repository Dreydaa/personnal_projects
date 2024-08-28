package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var clients = make(map[*Client]bool)
var broadcast = make(chan []byte)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	defer ws.Close()

	client := &Client{conn: ws, send: make(chan []byte)}
	clients[client] = true

	go handleMessages(client)

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("ReadMessage error: %v", err)
			delete(clients, client)
			break
		}
		broadcast <- msg
	}
}

func handleMessages(client *Client) {
	for {
		msg := <-client.send
		for c := range clients {
			if c != client {
				err := c.conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					log.Printf("WriteMessage error: %v", err)
					c.conn.Close()
					delete(clients, c)
				}
			}
		}
	}
}

func broadcastMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				delete(clients, client)
			}
		}
	}
}

func main() {
	fs := http.FileServer(http.Dir("./templates"))
	http.Handle("/", fs)
	http.HandleFunc("/ws", handleConnections)

	go broadcastMessages()

	log.Println("Server started on :8000")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe error: ", err)
	}
}
