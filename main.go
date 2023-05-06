package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
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

type Channel struct {
	connections map[*Connection]bool
	mu          sync.Mutex
}

func (ch *Channel) addConnection(conn *Connection) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.connections[conn] = true
}

func (ch *Channel) removeConnection(conn *Connection) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	delete(ch.connections, conn)
}

func (ch *Channel) broadcastMessage(messageType int, message []byte) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	for conn := range ch.connections {
		if err := conn.WriteMessage(messageType, message); err != nil {
			log.Println("Write message error:", err)
			conn.Conn.Close()
			delete(ch.connections, conn)
		}
	}
}

var channels = make(map[string]*Channel)
var channelsMu sync.Mutex

func getChannel(path string) *Channel {
	channelsMu.Lock()
	defer channelsMu.Unlock()

	ch, ok := channels[path]
	if !ok {
		ch = &Channel{
			connections: make(map[*Connection]bool),
			mu:          sync.Mutex{},
		}
		channels[path] = ch
	}

	return ch
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	channel := getChannel(r.URL.Path)
	connection := &Connection{Conn: conn}
	channel.addConnection(connection)

	log.Printf("Client connected: %s", conn.RemoteAddr())

	defer func() {
		channel.removeConnection(connection)
		conn.Close()
		log.Printf("Client disconnected: %s", conn.RemoteAddr())
	}()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read message error:", err)
			break
		}

		log.Printf("Received message from %s: %s\n", conn.RemoteAddr(), message)
		channel.broadcastMessage(messageType, message)
	}
}

func main() {
	http.HandleFunc("/", websocketHandler)

	// Use environment variable for the port
	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8080" // Set a default port if not provided
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid port number: %s\n", portStr)
	}

	// Use environment variable for the listening address
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	if listenAddress == "" {
		listenAddress = "localhost" // Set a default listening address if not provided
	}

	log.Printf("Starting WebSocket server without SSL on http://%s:%d\n", listenAddress, port)
	log.Fatal(http.ListenAndServe(listenAddress+":"+portStr, nil))
}
