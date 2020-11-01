package models

// Room ...
type Room interface {
	GetID() string
	GetName() string
	GetPrivate() bool
}

// RoomRepository ...
type RoomRepository interface {
	AddRoom(room Room)
	FindRoomByName(name string) Room
}
