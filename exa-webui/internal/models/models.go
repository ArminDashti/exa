package models

type Shop struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Expense struct {
	ID       int     `json:"id,omitempty"`
	ShopID   int     `json:"shop_id"`
	ItemID   int     `json:"item_id"`
	Date     string  `json:"date"`
	Name     string  `json:"name"`
	Amount   float64 `json:"amount"`
	ShopName string  `json:"shop_name,omitempty"`
}

type CreateExpenseLine struct {
	Name   string  `json:"name" binding:"required"`
	Amount float64 `json:"amount" binding:"required"`
}

type CreateExpensesRequest struct {
	ShopID int                 `json:"shop_id" binding:"required"`
	Date   string              `json:"date" binding:"required"`
	Items  []CreateExpenseLine `json:"items" binding:"required,min=1,dive"`
}

type UpdateExpenseRequest struct {
	ShopID int     `json:"shop_id" binding:"required"`
	Date   string  `json:"date" binding:"required"`
	Name   string  `json:"name" binding:"required"`
	Amount float64 `json:"amount" binding:"required"`
}

type CreateShopRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateShopRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateItemRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateItemRequest struct {
	Name string `json:"name" binding:"required"`
}

type DuplicateCheckResponse struct {
	Exists bool `json:"exists"`
	Count  int  `json:"count"`
}

type MonthStats struct {
	Month string  `json:"month"`
	Total float64 `json:"total"`
}

type Stats struct {
	ByMonth []MonthStats `json:"by_month"`
}
