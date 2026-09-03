package handlers

import (
	"database/sql"
	"errors"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/armin/exa/backend/internal/database"
	"github.com/armin/exa/backend/internal/jalali"
	"github.com/armin/exa/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *database.DB
}

func New(db *database.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) ListShops(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q != "" && len([]rune(q)) < 3 {
		c.JSON(http.StatusOK, []models.Shop{})
		return
	}

	var (
		rows *sql.Rows
		err  error
	)
	if q != "" {
		rows, err = h.db.Query(
			`SELECT id, name FROM shops WHERE name LIKE ? COLLATE NOCASE ORDER BY name`,
			"%"+q+"%",
		)
	} else {
		rows, err = h.db.Query(`SELECT id, name FROM shops ORDER BY name`)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list shops"})
		return
	}
	defer rows.Close()

	shops := make([]models.Shop, 0)
	for rows.Next() {
		var s models.Shop
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read shops"})
			return
		}
		shops = append(shops, s)
	}

	c.JSON(http.StatusOK, shops)
}

func (h *Handler) CreateShop(c *gin.Context) {
	var req models.CreateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	result, err := h.db.Exec(`INSERT INTO shops (name) VALUES (?)`, name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"error": "shop already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shop"})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, models.Shop{ID: int(id), Name: name})
}

func (h *Handler) UpdateShop(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shop id"})
		return
	}

	var req models.UpdateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	result, err := h.db.Exec(`UPDATE shops SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"error": "shop already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update shop"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "shop not found"})
		return
	}

	c.JSON(http.StatusOK, models.Shop{ID: id, Name: name})
}

func (h *Handler) DeleteShop(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shop id"})
		return
	}

	var inUse int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM expenses WHERE shop_id = ?`, id).Scan(&inUse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check shop usage"})
		return
	}
	if inUse > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "shop is used by expenses"})
		return
	}

	result, err := h.db.Exec(`DELETE FROM shops WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete shop"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "shop not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) ListItems(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q != "" && len([]rune(q)) < 3 {
		c.JSON(http.StatusOK, []models.Item{})
		return
	}

	var (
		rows *sql.Rows
		err  error
	)
	if q != "" {
		rows, err = h.db.Query(
			`SELECT id, name FROM items WHERE name LIKE ? COLLATE NOCASE ORDER BY name`,
			"%"+q+"%",
		)
	} else {
		rows, err = h.db.Query(`SELECT id, name FROM items ORDER BY name`)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list items"})
		return
	}
	defer rows.Close()

	items := make([]models.Item, 0)
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read items"})
			return
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

func (h *Handler) CreateItem(c *gin.Context) {
	var req models.CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	result, err := h.db.Exec(`INSERT INTO items (name) VALUES (?)`, name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"error": "item already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create item"})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, models.Item{ID: int(id), Name: name})
}

func (h *Handler) UpdateItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req models.UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	result, err := h.db.Exec(`UPDATE items SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"error": "item already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update item"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	c.JSON(http.StatusOK, models.Item{ID: id, Name: name})
}

func (h *Handler) DeleteItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var inUse int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM expenses WHERE item_id = ?`, id).Scan(&inUse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check item usage"})
		return
	}
	if inUse > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "item is used by expenses"})
		return
	}

	result, err := h.db.Exec(`DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete item"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GetStats(c *gin.Context) {
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")

	where := []string{"1=1"}
	args := make([]any, 0)
	if fromDate != "" {
		where = append(where, "e.date >= ?")
		args = append(args, fromDate)
	}
	if toDate != "" {
		where = append(where, "e.date <= ?")
		args = append(args, toDate)
	}
	whereSQL := strings.Join(where, " AND ")

	query := `SELECT e.date, e.amount FROM expenses e WHERE ` + whereSQL

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load expenses for stats"})
		return
	}
	defer rows.Close()

	byMonth := make(map[string]*models.MonthStats)
	for rows.Next() {
		var date string
		var amount float64
		if err := rows.Scan(&date, &amount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read expense stats"})
			return
		}
		month, err := jalali.MonthKeyFromGregorian(date)
		if err != nil {
			continue
		}
		row, ok := byMonth[month]
		if !ok {
			row = &models.MonthStats{Month: month}
			byMonth[month] = row
		}
		row.Total += amount
	}

	months := make([]models.MonthStats, 0, len(byMonth))
	for _, row := range byMonth {
		months = append(months, *row)
	}
	sort.Slice(months, func(i, j int) bool {
		return months[i].Month > months[j].Month
	})

	c.JSON(http.StatusOK, models.Stats{ByMonth: months})
}

func validateWholeAmount(amount float64) error {
	if amount < 0 {
		return errors.New("amount must be >= 0")
	}
	if amount != math.Trunc(amount) {
		return errors.New("amount must be a whole number")
	}
	return nil
}

func upsertItem(tx *sql.Tx, name string) (int64, error) {
	return database.UpsertItemTx(tx, name)
}

func (h *Handler) CheckDuplicateExpense(c *gin.Context) {
	date := strings.TrimSpace(c.Query("date"))
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}

	excludeID := strings.TrimSpace(c.Query("exclude_id"))
	itemIDStr := strings.TrimSpace(c.Query("item_id"))
	name := strings.TrimSpace(c.Query("name"))

	var (
		count int
		err   error
	)

	switch {
	case itemIDStr != "":
		itemID, parseErr := strconv.Atoi(itemIDStr)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item_id"})
			return
		}
		if excludeID != "" {
			err = h.db.QueryRow(
				`SELECT COUNT(*) FROM expenses WHERE item_id = ? AND date = ? AND id != ?`,
				itemID, date, excludeID,
			).Scan(&count)
		} else {
			err = h.db.QueryRow(
				`SELECT COUNT(*) FROM expenses WHERE item_id = ? AND date = ?`,
				itemID, date,
			).Scan(&count)
		}
	case name != "":
		if excludeID != "" {
			err = h.db.QueryRow(
				`SELECT COUNT(*) FROM expenses e
				 JOIN items i ON i.id = e.item_id
				 WHERE i.name = ? COLLATE NOCASE AND e.date = ? AND e.id != ?`,
				name, date, excludeID,
			).Scan(&count)
		} else {
			err = h.db.QueryRow(
				`SELECT COUNT(*) FROM expenses e
				 JOIN items i ON i.id = e.item_id
				 WHERE i.name = ? COLLATE NOCASE AND e.date = ?`,
				name, date,
			).Scan(&count)
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_id or name is required"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check duplicate"})
		return
	}

	c.JSON(http.StatusOK, models.DuplicateCheckResponse{Exists: count > 0, Count: count})
}

func (h *Handler) ListExpenses(c *gin.Context) {
	query := `
		SELECT e.id, e.shop_id, s.name, e.item_id, i.name, e.date, e.amount
		FROM expenses e
		JOIN shops s ON s.id = e.shop_id
		JOIN items i ON i.id = e.item_id
		WHERE 1=1
	`
	args := []any{}

	if from := c.Query("from_date"); from != "" {
		query += ` AND e.date >= ?`
		args = append(args, from)
	}
	if to := c.Query("to_date"); to != "" {
		query += ` AND e.date <= ?`
		args = append(args, to)
	}

	query += ` ORDER BY e.date DESC, e.id DESC`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list expenses"})
		return
	}
	defer rows.Close()

	expenses := make([]models.Expense, 0)
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(
			&e.ID, &e.ShopID, &e.ShopName,
			&e.ItemID, &e.Name, &e.Date, &e.Amount,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read expenses"})
			return
		}
		expenses = append(expenses, e)
	}

	c.JSON(http.StatusOK, expenses)
}

func (h *Handler) GetExpense(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return
	}

	exp, err := h.fetchExpense(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "expense not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get expense"})
		return
	}

	c.JSON(http.StatusOK, exp)
}

func (h *Handler) fetchExpense(id int) (models.Expense, error) {
	var e models.Expense
	err := h.db.QueryRow(`
		SELECT e.id, e.shop_id, s.name, e.item_id, i.name, e.date, e.amount
		FROM expenses e
		JOIN shops s ON s.id = e.shop_id
		JOIN items i ON i.id = e.item_id
		WHERE e.id = ?`, id,
	).Scan(
		&e.ID, &e.ShopID, &e.ShopName,
		&e.ItemID, &e.Name, &e.Date, &e.Amount,
	)
	return e, err
}

func (h *Handler) CreateExpenses(c *gin.Context) {
	var req models.CreateExpensesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, item := range req.Items {
		if strings.TrimSpace(item.Name) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "item name is required"})
			return
		}
		if err := validateWholeAmount(item.Amount); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	var shopExists int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM shops WHERE id = ?`, req.ShopID).Scan(&shopExists); err != nil || shopExists == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shop_id"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	createdIDs := make([]int, 0, len(req.Items))
	for _, item := range req.Items {
		name := strings.TrimSpace(item.Name)
		itemID, err := upsertItem(tx, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve item"})
			return
		}
		result, err := tx.Exec(
			`INSERT INTO expenses (shop_id, item_id, date, amount) VALUES (?, ?, ?, ?)`,
			req.ShopID, itemID, req.Date, item.Amount,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create expense"})
			return
		}
		expenseID, _ := result.LastInsertId()
		createdIDs = append(createdIDs, int(expenseID))
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save expenses"})
		return
	}

	created := make([]models.Expense, 0, len(createdIDs))
	for _, id := range createdIDs {
		exp, err := h.fetchExpense(id)
		if err != nil {
			continue
		}
		created = append(created, exp)
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) UpdateExpense(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return
	}

	var req models.UpdateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validateWholeAmount(req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	var shopExists int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM shops WHERE id = ?`, req.ShopID).Scan(&shopExists); err != nil || shopExists == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shop_id"})
		return
	}

	var exists int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM expenses WHERE id = ?`, id).Scan(&exists); err != nil || exists == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "expense not found"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	itemID, err := upsertItem(tx, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve item"})
		return
	}

	if _, err := tx.Exec(
		`UPDATE expenses SET shop_id = ?, item_id = ?, date = ?, amount = ? WHERE id = ?`,
		req.ShopID, itemID, req.Date, req.Amount, id,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update expense"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save expense"})
		return
	}

	exp, err := h.fetchExpense(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"id": id})
		return
	}

	c.JSON(http.StatusOK, exp)
}

func (h *Handler) DeleteExpense(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return
	}

	result, err := h.db.Exec(`DELETE FROM expenses WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete expense"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "expense not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
