package database

import "database/sql"

func InitDB() *sql.DB {
	db, err := sql.Open("sqlite3", "database/db.db")
	if err != nil {
		panic(err)
	}

	return db
}
