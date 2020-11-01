package main

import (
	"chat/models"
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
	Action  string      `json:"action"`
	Message string      `json:"message"`
	Target  *Room       `json:"target"`
	Sender  models.User `json:"sender"`
}

// UnmarshalJSON ...
func (message *Message) UnmarshalJSON(data []byte) error {

	type Alias Message
	msg := &struct {
		Sender Client `json:"sender"`
		*Alias
	}{
		Alias: (*Alias)(message),
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	message.Sender = &msg.Sender

	return nil
}

func (message *Message) encode() []byte {

	json, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
	}

	return json
}
