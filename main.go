package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Connection struct {
	Conn *websocket.Conn
	mu   sync.Mutex
}

func (c *Connection) WriteMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteMessage(messageType, data)
}

var connections = make(map[*Connection]bool)
var connectionsMu sync.Mutex

func addConnection(conn *Connection) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	connections[conn] = true
}

func removeConnection(conn *Connection) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	delete(connections, conn)
}

func broadcastMessage(messageType int, message []byte) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	for conn := range connections {
		if err := conn.WriteMessage(messageType, message); err != nil {
			log.Println("Write message error:", err)
			conn.Conn.Close()
			delete(connections, conn)
		}
	}
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	connection := &Connection{Conn: conn}
	addConnection(connection)
	defer func() {
		removeConnection(connection)
		conn.Close()
	}()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read message error:", err)
			break
		}

		log.Printf("Received message: %s\n", message)
		broadcastMessage(messageType, message)
	}
}

func main() {
	http.HandleFunc("/ws", websocketHandler)

	// Use your own certificate and key files
	certFile := "cert.pem"
	keyFile := "privkey.pem"

	log.Println("Starting WebSocket server on https://localhost:8080/ws")
	log.Fatal(http.ListenAndServeTLS(":9080", certFile, keyFile, nil))
}
