package main

import (
	"encoding/json"
	"log"
)

const (
	SendMessageAction     = "send-message"
	JoinRoomAction        = "join-room"
	LeaveRoomAction       = "leave-room"
	UserJoinedAction      = "user-join"
	UserLeftAction        = "user-left"
	JoinRoomPrivateAction = "join-room-private"
	RoomJoinedAction      = "room-joined"
)

// Message ...
type Message struct {
	Action  string  `json:"action"`
	Message string  `json:"message"`
	Target  *Room   `json:"target"`
	Sender  *Client `json:"sender"`
}

func (message *Message) encode() []byte {

	json, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
	}

	return json
}
