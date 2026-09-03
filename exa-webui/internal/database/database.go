package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &DB{DB: db}, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS shops (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS expenses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		shop_id INTEGER NOT NULL,
		item_id INTEGER NOT NULL,
		date TEXT NOT NULL,
		amount REAL NOT NULL,
		FOREIGN KEY (shop_id) REFERENCES shops(id),
		FOREIGN KEY (item_id) REFERENCES items(id)
	);

	CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(date);
	CREATE INDEX IF NOT EXISTS idx_expenses_shop ON expenses(shop_id);
	CREATE INDEX IF NOT EXISTS idx_expenses_item ON expenses(item_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	return nil
}

func UpsertItemTx(tx *sql.Tx, name string) (int64, error) {
	var id int64
	err := tx.QueryRow(`SELECT id FROM items WHERE name = ? COLLATE NOCASE`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("lookup item %q: %w", name, err)
	}

	result, err := tx.Exec(`INSERT INTO items (name) VALUES (?)`, name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			if err := tx.QueryRow(`SELECT id FROM items WHERE name = ? COLLATE NOCASE`, name).Scan(&id); err != nil {
				return 0, fmt.Errorf("resolve item after conflict %q: %w", name, err)
			}
			return id, nil
		}
		return 0, fmt.Errorf("insert item %q: %w", name, err)
	}
	return result.LastInsertId()
}
