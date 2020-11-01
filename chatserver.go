package main

import (
	"chat/config"
	"chat/models"
	"encoding/json"
	"log"

	"github.com/google/uuid"
)

// PubSubGeneralChannel ...
const PubSubGeneralChannel = "general"

// WsServer structure for client web sockets connections
type WsServer struct {
	clients    map[*Client]bool
	rooms      map[*Room]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	users          []models.User
	roomRepository models.RoomRepository
	userRepository models.UserRepository
}

// NewWebsocketServer creates a new WsServer type
func NewWebsocketServer(roomRepository models.RoomRepository, userRepository models.UserRepository) *WsServer {

	wsServer := &WsServer{
		clients:        make(map[*Client]bool),
		rooms:          make(map[*Room]bool),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		broadcast:      make(chan []byte),
		roomRepository: roomRepository,
		userRepository: userRepository,
	}

	// Add users from database to server
	wsServer.users = userRepository.GetAllUsers()

	return wsServer
}

// Run our websocket server, accepting various requests
func (server *WsServer) Run() {

	go server.listenPubSubChannel()
	for {
		select {

		case client := <-server.register:
			server.registerClient(client)

		case client := <-server.unregister:
			server.unregisterClient(client)

		case message := <-server.broadcast:
			server.broadcastToClients(message)
		}

	}
}

func (server *WsServer) findRoomByName(name string) *Room {

	var foundRoom *Room
	for room := range server.rooms {
		if room.GetName() == name {
			foundRoom = room
			break
		}
	}

	if foundRoom == nil {
		foundRoom = server.runRoomFromRepository(name)
	}

	return foundRoom
}

func (server *WsServer) runRoomFromRepository(name string) *Room {

	var room *Room
	dbRoom := server.roomRepository.FindRoomByName(name)

	if dbRoom != nil {
		room = NewRoom(dbRoom.GetName(), dbRoom.GetPrivate())
		room.ID, _ = uuid.Parse(dbRoom.GetID())
		go room.RunRoom()

		server.rooms[room] = true
	}

	return room
}

func (server *WsServer) findRoomByID(ID string) *Room {

	var foundRoom *Room
	for room := range server.rooms {
		if room.GetID() == ID {
			foundRoom = room
			break
		}
	}

	return foundRoom
}

func (server *WsServer) findClientByID(ID string) *Client {

	var foundClient *Client
	for client := range server.clients {
		if client.GetID() == ID {
			foundClient = client
			break
		}
	}

	return foundClient
}

func (server *WsServer) registerClient(client *Client) {

	// Add user to the repo
	server.userRepository.AddUser(client)

	// Publish user in pubsub
	server.publishClientJoined(client)

	server.listOnlineClients(client)
	server.clients[client] = true
}

func (server *WsServer) unregisterClient(client *Client) {

	if _, ok := server.clients[client]; ok {

		delete(server.clients, client)

		for i, user := range server.users {
			if user.GetID() == client.GetID() {
				server.users[i] = server.users[len(server.users)-1]
				server.users = server.users[:len(server.users)-1]
			}
		}

		// Remove user from repo
		server.userRepository.RemoveUser(client)

		// Publish user left in PubSub
		server.publishClientLeft(client)
	}
}

func (server *WsServer) broadcastToClients(message []byte) {
	for client := range server.clients {
		client.send <- message
	}
}

func (server *WsServer) createRoom(name string, private bool) *Room {

	room := NewRoom(name, private)
	server.roomRepository.AddRoom(room)

	go room.RunRoom()
	server.rooms[room] = true

	return room
}

func (server *WsServer) notifyClientJoined(client *Client) {

	message := &Message{
		Action: UserJoinedAction,
		Sender: client,
	}

	server.broadcastToClients(message.encode())
}

func (server *WsServer) notifyClientLeft(client *Client) {

	message := &Message{
		Action: UserLeftAction,
		Sender: client,
	}

	server.broadcastToClients(message.encode())
}

func (server *WsServer) listOnlineClients(client *Client) {

	for _, user := range server.users {
		message := &Message{
			Action: UserJoinedAction,
			Sender: user,
		}
		client.send <- message.encode()
	}
}

func (server *WsServer) publishClientJoined(client *Client) {

	message := &Message{
		Action: UserJoinedAction,
		Sender: client,
	}

	if err := config.Redis.Publish(ctx, PubSubGeneralChannel, message.encode()).Err(); err != nil {
		log.Println(err)
	}
}

func (server *WsServer) publishClientLeft(client *Client) {

	message := &Message{
		Action: UserLeftAction,
		Sender: client,
	}

	if err := config.Redis.Publish(ctx, PubSubGeneralChannel, message.encode()).Err(); err != nil {
		log.Println(err)
	}
}

// Listen to pub/sub general channels
func (server *WsServer) listenPubSubChannel() {

	pubSub := config.Redis.Subscribe(ctx, PubSubGeneralChannel)
	ch := pubSub.Channel()

	for msg := range ch {
		var message Message
		if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
			log.Printf("Error on unmarshal json message %s", err)
			return
		}

		switch message.Action {
		case UserJoinedAction:
			server.handleUserJoined(message)
		case UserLeftAction:
			server.handleUserLeft(message)
		case JoinRoomPrivateAction:
			server.handleUserJoinPrivate(message)
		}
	}
}

func (server *WsServer) handleUserJoinPrivate(message Message) {

	// Find client for given user, if found add the user to the room.
	targetClient := server.findClientByID(message.Message)
	if targetClient != nil {
		targetClient.joinRoom(message.Target.GetName(), message.Sender)
	}
}

func (server *WsServer) findUserByID(ID string) models.User {

	var foundUser models.User
	for _, user := range server.users {
		if user.GetID() == ID {
			foundUser = user
			break
		}
	}
	return foundUser
}

func (server *WsServer) handleUserJoined(message Message) {
	// Add the user to the slice
	server.users = append(server.users, message.Sender)
	server.broadcastToClients(message.encode())
}

func (server *WsServer) handleUserLeft(message Message) {

	// Remove the user from the slice
	for i, user := range server.users {
		if user.GetID() == message.Sender.GetID() {
			server.users[i] = server.users[len(server.users)-1]
			server.users = server.users[:len(server.users)-1]
		}
	}

	server.broadcastToClients(message.encode())
}
