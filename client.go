package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Max wait time when writing message to peer
	writeWait = 10 * time.Second

	// Max time till next pong from peer
	pongWait = 60 * time.Second

	// Send ping interval, must be less then pong wait time
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 10000
)

var (
	newLine = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// Client represents the websocket client at the server
type Client struct {
	ID       uuid.UUID `json:"id"`
	conn     *websocket.Conn
	wsServer *WsServer
	send     chan []byte
	rooms    map[*Room]bool
	Name     string `json:"name"`
}

// GetName gets client name
func (client *Client) GetName() string {
	return client.Name
}

// GetID returns client id as string
func (client *Client) GetID() string {
	return client.ID.String()
}

func newClient(conn *websocket.Conn, wsServer *WsServer, name string) *Client {

	return &Client{
		ID:       uuid.New(),
		Name:     name,
		conn:     conn,
		wsServer: wsServer,
		send:     make(chan []byte),
		rooms:    make(map[*Room]bool),
	}
}

func (client *Client) readPump() {

	defer func() {
		client.disconnect()
	}()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(
		func(string) error {
			client.conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		},
	)

	// Start endless read loop, waiting for messages from client
	for {
		_, jsonMessage, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("unexpected close error %v", err)
			}
			break
		}

		client.handleNewMessage(jsonMessage)
	}
}

func (client *Client) writePump() {

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {

		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				// The WsServer closed the channel.
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Attach queued chat messages to the current websocket message.
			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write(newLine)
				w.Write(<-client.send)

			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		}
	}
}

func (client *Client) disconnect() {

	client.wsServer.unregister <- client
	for room := range client.rooms {
		room.unregister <- client
	}
	close(client.send)
	client.conn.Close()

}

func (client *Client) handleNewMessage(jsonMessage []byte) {

	var message Message
	if err := json.Unmarshal(jsonMessage, &message); err != nil {
		log.Printf("Error while unmarshaling json %s", err)
		return
	}

	// Attach the client object as the sender of the messsage.
	message.Sender = client

	switch message.Action {
	// The send-message action, this will send messages to a specific room now.
	// Which room wil depend on the message Target
	case SendMessageAction:
		roomID := message.Target.GetID()
		// Use the ChatServer method to find the room, and if found, broadcast!
		if room := client.wsServer.findRoomByID(roomID); room != nil {
			room.broadcast <- &message
		}

		// We delegate the join and leave actions
	case JoinRoomAction:
		client.handleJoinRoomMessage(message)

	case LeaveRoomAction:
		client.handleLeaveRoomMessage(message)

	case JoinRoomPrivateAction:
		client.handleJoinRoomPrivateMessage(message)

	}

}

func (client *Client) notifyRoomJoined(room *Room, sender *Client) {

	message := &Message{
		Action: JoinRoomAction,
		Target: room,
		Sender: sender,
	}

	client.send <- message.encode()
}

// Joining a room both for public and private roooms
// When joining a private room a sender is passed as the opposing party
func (client *Client) joinRoom(roomName string, sender *Client) {

	room := client.wsServer.findRoomByName(roomName)
	if room == nil {
		room = client.wsServer.createRoom(roomName, sender != nil)
	}

	// Don't allow to join private rooms through public room message
	if sender == nil && room.Private {
		return
	}

	if !client.IsInRoom(room) {
		client.rooms[room] = true
		room.register <- client
		client.notifyRoomJoined(room, sender)
	}
}

// IsInRoom returns true if client is already in room
// Otherwise returns false
func (client *Client) IsInRoom(room *Room) bool {

	if _, ok := room.clients[client]; ok {
		return true
	}

	return false
}

// When joining a private room we will combine the IDs of the users
// Then we will bothe join the client and the target.
func (client *Client) handleJoinRoomPrivateMessage(message Message) {

	target := client.wsServer.findClientByID(message.Message)
	if target == nil {
		return
	}

	// create unique room name combined to the two IDs
	roomName := message.Message + client.ID.String()

	client.joinRoom(roomName, target)
	target.joinRoom(roomName, client)
}

func (client *Client) handleJoinRoomMessage(message Message) {

	roomName := message.Message

	client.joinRoom(roomName, nil)

}

func (client *Client) handleLeaveRoomMessage(message Message) {

	room := client.wsServer.findRoomByID(message.Message)
	if room == nil {
		return
	}

	if _, ok := client.rooms[room]; ok {
		delete(client.rooms, room)
	}

	room.unregister <- client
}

// ServeWs ...
func ServeWs(wsServer *WsServer, w http.ResponseWriter, r *http.Request) {

	name, ok := r.URL.Query()["name"]

	if !ok || len(name[0]) < 1 {
		log.Println("Url param 'name' is missing")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(conn, wsServer, name[0])
	go client.writePump()
	go client.readPump()

	fmt.Println("New client joined the hub")
	fmt.Println(client)
}
