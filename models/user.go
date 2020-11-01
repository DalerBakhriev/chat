package models

// User ..
type User interface {
	GetID() string
	GetName() string
}

// UserRepository ...
type UserRepository interface {
	AddUser(user User)
	RemoveUser(user User)
	FindUserByID(id string) User
	GetAllUsers() []User
}
