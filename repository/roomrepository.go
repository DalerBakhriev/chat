package repository

import (
	"chat/models"
	"database/sql"
	"log"
)

// Room ...
type Room struct {
	ID      string
	Name    string
	Private bool
}

// GetID returns id property
func (room *Room) GetID() string {
	return room.ID
}

// GetName returns name property
func (room *Room) GetName() string {
	return room.Name
}

// GetPrivate returns private property
func (room *Room) GetPrivate() bool {
	return room.Private
}

// RoomRepository for db interaction
type RoomRepository struct {
	Db *sql.DB
}

// AddRoom adds room into database
func (repo *RoomRepository) AddRoom(room models.Room) {

	stmt, err := repo.Db.Prepare(
		`INSERT INTO room(id, name, private)
		 VALUES (?, ?, ?)`,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(room.GetID(), room.GetName(), room.GetPrivate())
	if err != nil {
		log.Fatal(err)
	}
}

// FindRoomByName finds room by name in database
func (repo *RoomRepository) FindRoomByName(name string) models.Room {

	row := repo.Db.QueryRow(
		`SELECT id,
				name,
				private
		 FROM room
		 WHERE name = ?
		 LIMIT 1`,
		name,
	)

	var room Room
	if err := row.Scan(&room.ID, &room.Name, &room.Private); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Fatal(err)
	}

	return &room
}
