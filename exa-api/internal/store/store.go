package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	DefaultUsername = "armin"
	DefaultPassword = "dopadopa123"
)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Store struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Expense struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	StoreID     string    `json:"store_id"`
	StoreName   string    `json:"store_name,omitempty"`
	Amount      float64   `json:"amount"`
	ExpenseDate string    `json:"expense_date"`
	Note        *string   `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

type SummaryStats struct {
	TotalAmount float64 `json:"total_amount"`
	Count       int     `json:"count"`
}

type StoreStat struct {
	StoreID     string  `json:"store_id"`
	StoreName   string  `json:"store_name"`
	TotalAmount float64 `json:"total_amount"`
	Count       int     `json:"count"`
}

func Open(databaseURL string) (*sql.DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

func Migrate(db *sql.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, name := range files {
		body, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}

func SeedDefaultUser(ctx context.Context, db *sql.DB, passwordHash string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash
	`, DefaultUsername, passwordHash)
	return err
}

func GetUserByUsername(ctx context.Context, db *sql.DB, username string) (*User, error) {
	var u User
	err := db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at
		FROM users WHERE username = $1
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func ListStores(ctx context.Context, db *sql.DB) ([]Store, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, created_at FROM stores ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Store, 0)
	for rows.Next() {
		var s Store
		if err := rows.Scan(&s.ID, &s.Name, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func CreateStore(ctx context.Context, db *sql.DB, name string) (*Store, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	var s Store
	err := db.QueryRowContext(ctx, `
		INSERT INTO stores (name) VALUES ($1)
		RETURNING id, name, created_at
	`, name).Scan(&s.ID, &s.Name, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func UpdateStore(ctx context.Context, db *sql.DB, id, name string) (*Store, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	var s Store
	err := db.QueryRowContext(ctx, `
		UPDATE stores SET name = $2 WHERE id = $1
		RETURNING id, name, created_at
	`, id, name).Scan(&s.ID, &s.Name, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func DeleteStore(ctx context.Context, db *sql.DB, id string) error {
	res, err := db.ExecContext(ctx, `DELETE FROM stores WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type ExpenseFilter struct {
	UserID  string
	StoreID string
	From    string
	To      string
	Limit   int
}

func ListExpenses(ctx context.Context, db *sql.DB, f ExpenseFilter) ([]Expense, error) {
	q := `
		SELECT e.id, e.user_id, e.store_id, s.name, e.amount, e.expense_date::text, e.note, e.created_at
		FROM expenses e
		JOIN stores s ON s.id = e.store_id
		WHERE e.user_id = $1
	`
	args := []any{f.UserID}
	n := 2
	if f.StoreID != "" {
		q += fmt.Sprintf(" AND e.store_id = $%d", n)
		args = append(args, f.StoreID)
		n++
	}
	if f.From != "" {
		q += fmt.Sprintf(" AND e.expense_date >= $%d::date", n)
		args = append(args, f.From)
		n++
	}
	if f.To != "" {
		q += fmt.Sprintf(" AND e.expense_date <= $%d::date", n)
		args = append(args, f.To)
		n++
	}
	q += " ORDER BY e.expense_date DESC, e.created_at DESC"
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", n)
		args = append(args, f.Limit)
	}

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Expense, 0)
	for rows.Next() {
		var e Expense
		if err := rows.Scan(&e.ID, &e.UserID, &e.StoreID, &e.StoreName, &e.Amount, &e.ExpenseDate, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func CreateExpense(ctx context.Context, db *sql.DB, userID, storeID, expenseDate string, amount float64, note *string) (*Expense, error) {
	var e Expense
	err := db.QueryRowContext(ctx, `
		INSERT INTO expenses (user_id, store_id, amount, expense_date, note)
		VALUES ($1, $2, $3, $4::date, $5)
		RETURNING id, user_id, store_id, amount, expense_date::text, note, created_at
	`, userID, storeID, amount, expenseDate, note).Scan(
		&e.ID, &e.UserID, &e.StoreID, &e.Amount, &e.ExpenseDate, &e.Note, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = db.QueryRowContext(ctx, `SELECT name FROM stores WHERE id = $1`, storeID).Scan(&e.StoreName)
	return &e, nil
}

func UpdateExpense(ctx context.Context, db *sql.DB, userID, id, storeID, expenseDate string, amount float64, note *string) (*Expense, error) {
	var e Expense
	err := db.QueryRowContext(ctx, `
		UPDATE expenses
		SET store_id = $3, amount = $4, expense_date = $5::date, note = $6
		WHERE id = $2 AND user_id = $1
		RETURNING id, user_id, store_id, amount, expense_date::text, note, created_at
	`, userID, id, storeID, amount, expenseDate, note).Scan(
		&e.ID, &e.UserID, &e.StoreID, &e.Amount, &e.ExpenseDate, &e.Note, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = db.QueryRowContext(ctx, `SELECT name FROM stores WHERE id = $1`, storeID).Scan(&e.StoreName)
	return &e, nil
}

func DeleteExpense(ctx context.Context, db *sql.DB, userID, id string) error {
	res, err := db.ExecContext(ctx, `DELETE FROM expenses WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func GetSummaryStats(ctx context.Context, db *sql.DB, userID, from, to string) (*SummaryStats, error) {
	q := `
		SELECT COALESCE(SUM(amount), 0), COUNT(*)
		FROM expenses WHERE user_id = $1
	`
	args := []any{userID}
	n := 2
	if from != "" {
		q += fmt.Sprintf(" AND expense_date >= $%d::date", n)
		args = append(args, from)
		n++
	}
	if to != "" {
		q += fmt.Sprintf(" AND expense_date <= $%d::date", n)
		args = append(args, to)
	}
	var s SummaryStats
	err := db.QueryRowContext(ctx, q, args...).Scan(&s.TotalAmount, &s.Count)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func StatsByStore(ctx context.Context, db *sql.DB, userID, from, to string) ([]StoreStat, error) {
	q := `
		SELECT s.id, s.name, COALESCE(SUM(e.amount), 0), COUNT(e.id)
		FROM stores s
		LEFT JOIN expenses e ON e.store_id = s.id AND e.user_id = $1
	`
	args := []any{userID}
	n := 2
	conds := make([]string, 0, 2)
	if from != "" {
		conds = append(conds, fmt.Sprintf("e.expense_date >= $%d::date", n))
		args = append(args, from)
		n++
	}
	if to != "" {
		conds = append(conds, fmt.Sprintf("e.expense_date <= $%d::date", n))
		args = append(args, to)
		n++
	}
	if len(conds) > 0 {
		q += " AND (" + strings.Join(conds, " AND ") + " OR e.id IS NULL)"
	}
	q += " GROUP BY s.id, s.name ORDER BY s.name"

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]StoreStat, 0)
	for rows.Next() {
		var s StoreStat
		if err := rows.Scan(&s.StoreID, &s.StoreName, &s.TotalAmount, &s.Count); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
