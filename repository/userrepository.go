package repository

import (
	"chat/models"
	"database/sql"
	"log"
)

// User ...
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetID ...
func (user *User) GetID() string {
	return user.ID
}

// GetName ...
func (user *User) GetName() string {
	return user.Name
}

// UserRepository ...
type UserRepository struct {
	Db *sql.DB
}

// AddUser adds user to db
func (repo *UserRepository) AddUser(user models.User) {

	stmt, err := repo.Db.Prepare(
		`INSERT INTO user(id, name)
		 VALUES (?, ?)`,
	)

	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(user.GetID(), user.GetName())
	if err != nil {
		log.Fatal(err)
	}
}

// RemoveUser removes user from db
func (repo *UserRepository) RemoveUser(user models.User) {

	stmt, err := repo.Db.Prepare(
		`DELETE
		 FROM user
		 WHERE id = ? AND name = ?`,
	)

	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(user.GetID(), user.GetName())
	if err != nil {
		log.Fatal(err)
	}

}

// FindUserByID find user by id from database
func (repo *UserRepository) FindUserByID(id string) models.User {

	row := repo.Db.QueryRow(
		`SELECT id,
				name
		 FROM user
		 WHERE id = ?`,
		id,
	)

	var user User
	if err := row.Scan(&user.ID, &user.Name); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
	}

	return &user
}

// GetAllUsers gets all existingg users from database
func (repo *UserRepository) GetAllUsers() []models.User {

	rows, err := repo.Db.Query(
		`SELECT id,
				name
		 FROM user`,
	)

	if err != nil {
		log.Fatal(err)
	}

	var users []models.User
	defer rows.Close()

	for rows.Next() {
		var user User
		if err = rows.Scan(&user.ID, &user.Name); err != nil {
			log.Fatal(err)
		}
		users = append(users, &user)
	}

	return users
}
