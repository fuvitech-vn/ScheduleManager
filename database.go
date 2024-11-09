package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./tasks.db")
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	createTables()
}

func createTables() {
	userQuery := `CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT UNIQUE,
        token TEXT
    )`
	_, err := db.Exec(userQuery)
	if err != nil {
		log.Fatal("Error creating users table:", err)
	}

	taskQuery := `CREATE TABLE IF NOT EXISTS tasks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER,
        name TEXT,
        message TEXT,
        url TEXT,
        interval INTEGER,
        start INTEGER,
        end INTEGER,
        is_recurring BOOLEAN,
        enabled BOOLEAN DEFAULT FALSE,  -- Default value for Enabled
        FOREIGN KEY (user_id) REFERENCES users(id),
        UNIQUE(user_id, name)  -- Ensure task name is unique per user
    )`
	_, err = db.Exec(taskQuery)
	if err != nil {
		log.Fatal("Error creating tasks table:", err)
	}
}
