package main

import (
	"database/sql"
	"errors"
	"log"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var ErrUsernameTaken = errors.New("username already exists")

func initDB() *sql.DB {
	db, err := sql.Open("sqlite", "people.db")

	if err != nil {
		log.Fatal(err)
	}

	createTable := `
		CREATE TABLE IF NOT EXISTS people (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL
		);
	`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	createUsersTable := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL
		);
	`

	_, err = db.Exec(createUsersTable)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func createPerson(db *sql.DB, person Person) (Person, error) {

	result, err := db.Exec(
		"INSERT INTO people (name) VALUES (?)",
		person.Name,
	)

	if err != nil {
		return Person{}, err
	}

	id, err := result.LastInsertId()

	if err != nil {
		return Person{}, err
	}

	person.ID = int(id)

	return person, nil
}

func getPeople(db *sql.DB) ([]Person, error) {

	rows, err := db.Query(
		"SELECT id, name FROM people",
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var people []Person

	for rows.Next() {

		var person Person

		err := rows.Scan(
			&person.ID,
			&person.Name,
		)

		if err != nil {
			return nil, err
		}

		people = append(people, person)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return people, nil

}

func getPerson(db *sql.DB, id int) (Person, error) {

	var person Person

	err := db.QueryRow(
		"SELECT id, name FROM people WHERE id = ?",
		id,
	).Scan(
		&person.ID,
		&person.Name,
	)

	if err != nil {
		return Person{}, err
	}

	return person, nil

}

func deletePerson(db *sql.DB, id int) (bool, error) {

	result, err := db.Exec(
		"DELETE FROM people WHERE id = ?",
		id,
	)

	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil

}

func createUser(db *sql.DB, user User) (User, error) {

	result, err := db.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		user.Username,
		user.PasswordHash,
	)

	if err != nil {

		var sqliteErr *sqlite.Error

		if errors.As(err, &sqliteErr) {
			if sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return User{}, ErrUsernameTaken
			}
		}

		return User{}, err
	}

	id, err := result.LastInsertId()

	if err != nil {
		return User{}, err
	}

	user.ID = int(id)

	return user, nil
}

func getUserByUsername(db *sql.DB, username string) (User, error) {

	var user User

	err := db.QueryRow(
		"SELECT id, username, password_hash FROM users WHERE username = ?",
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
	)

	if err != nil {
		return User{}, err
	}

	return user, nil
}

func getUserByID(db *sql.DB, id int) (User, error) {

	var user User

	err := db.QueryRow(
		"SELECT id, username, password_hash FROM users WHERE id = ?",
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
	)

	if err != nil {
		return User{}, err
	}

	return user, nil
}
