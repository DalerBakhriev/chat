package main

import (
	"chat/config"
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
)

const welcomeMessage = "%s joined the room"

var ctx = context.Background()

// Room ...
type Room struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Private    bool      `json:"private"`
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
}

// NewRoom creates a new room
func NewRoom(name string, private bool) *Room {
	return &Room{
		ID:         uuid.New(),
		Name:       name,
		Private:    private,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
	}
}

// RunRoom runs our room, accepting various requests
func (room *Room) RunRoom() {

	// subscribe to pub/sub messages inside a new goroutine
	go room.subscribeToRoomMessages()

	for {
		select {

		case client := <-room.register:
			room.registerClientInRoom(client)

		case client := <-room.unregister:
			room.unregisterClientInRoom(client)

		case message := <-room.broadcast:
			room.publishRoomMessage(message.encode())
		}
	}
}

func (room *Room) registerClientInRoom(client *Client) {

	// by sending the message first the new user won't see his own message.
	if !room.Private {
		room.notifyClientJoined(client)
	}

	room.clients[client] = true
}

func (room *Room) unregisterClientInRoom(client *Client) {

	if _, ok := room.clients[client]; ok {
		delete(room.clients, client)
	}
}

func (room *Room) broadCastToClientsInRoom(message []byte) {
	for client := range room.clients {
		client.send <- message
	}
}

func (room *Room) notifyClientJoined(client *Client) {

	message := &Message{
		Action:  SendMessageAction,
		Target:  room,
		Message: fmt.Sprintf(welcomeMessage, client.GetName()),
	}

	room.publishRoomMessage(message.encode())
}

// GetName returns room's name
func (room *Room) GetName() string {
	return room.Name
}

// GetID returns room id as string
func (room *Room) GetID() string {
	return room.ID.String()
}

// GetPrivate returns private property of room
func (room *Room) GetPrivate() bool {
	return room.Private
}

func (room *Room) publishRoomMessage(message []byte) {

	err := config.Redis.Publish(ctx, room.GetName(), message).Err()

	if err != nil {
		log.Println(err)
	}
}

func (room *Room) subscribeToRoomMessages() {

	pubSub := config.Redis.Subscribe(ctx, room.GetName())
	ch := pubSub.Channel()

	for msg := range ch {
		room.broadCastToClientsInRoom([]byte(msg.Payload))
	}
}
