package sqlite

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// InitDB opens the DB and creates the table if missing.
func InitDB(path string) error {
	var err error
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS telegram_history (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        message TEXT NOT NULL,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
    )`)
	return err
}

// CloseDB closes the DB connection (call on shutdown).
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

func SaveMessage(userID int64, msg string) error {
	_, err := db.Exec("INSERT INTO telegram_history (user_id, message) VALUES (?, ?)", userID, msg)
	return err
}

// DeleteHistory removes all messages for a given user (used by /forget command).
func DeleteHistory(userID int64) error {
	_, err := db.Exec("DELETE FROM telegram_history WHERE user_id = ?", userID)
	return err
}

func GetHistory(userID int64, limit int) ([]string, error) {
	rows, err := db.Query("SELECT message FROM telegram_history WHERE user_id=? ORDER BY timestamp DESC LIMIT ?", userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []string
	for rows.Next() {
		var msg string
		if err := rows.Scan(&msg); err != nil {
			return nil, err
		}
		history = append(history, msg)
	}
	return history, nil
}
