package httpserver

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArminDashti/exa-api/internal/auth"
	"github.com/ArminDashti/exa-api/internal/config"
	"github.com/ArminDashti/exa-api/internal/store"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg config.Config
	db  *sql.DB
}

func New(cfg config.Config, db *sql.DB) *Server {
	return &Server{cfg: cfg, db: db}
}

func (s *Server) Router() *gin.Engine {
	r := gin.Default()
	allow := map[string]struct{}{}
	for _, o := range s.cfg.CORSOrigins {
		allow[o] = struct{}{}
	}
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if len(allow) == 0 {
				return true
			}
			_, ok := allow[origin]
			return ok
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", s.login)

		authed := api.Group("")
		authed.Use(auth.Middleware(s.cfg.JWTSecret))
		{
			authed.GET("/stores", s.listStores)
			authed.POST("/stores", s.createStore)
			authed.PATCH("/stores/:id", s.updateStore)
			authed.DELETE("/stores/:id", s.deleteStore)

			authed.GET("/expenses", s.listExpenses)
			authed.POST("/expenses", s.createExpense)
			authed.PATCH("/expenses/:id", s.updateExpense)
			authed.DELETE("/expenses/:id", s.deleteExpense)

			authed.GET("/stats/summary", s.statsSummary)
			authed.GET("/stats/by-store", s.statsByStore)
		}
	}
	return r
}

func writeError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password" binding:"required"`
}

func (s *Server) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request")
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		writeError(c, http.StatusBadRequest, "username is required")
		return
	}
	user, err := store.GetUserByUsername(c.Request.Context(), s.db, username)
	if err == sql.ErrNoRows {
		writeError(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "could not load user")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	token, err := auth.IssueToken(s.cfg.JWTSecret, user.ID, user.Username)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "could not issue token")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "username": user.Username})
}

func (s *Server) listStores(c *gin.Context) {
	list, err := store.ListStores(c.Request.Context(), s.db)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "could not list stores")
		return
	}
	c.JSON(http.StatusOK, gin.H{"stores": list})
}

type storeBody struct {
	Name string `json:"name" binding:"required"`
}

func (s *Server) createStore(c *gin.Context) {
	var req storeBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "name is required")
		return
	}
	st, err := store.CreateStore(c.Request.Context(), s.db, req.Name)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeError(c, http.StatusConflict, "store already exists")
			return
		}
		writeError(c, http.StatusInternalServerError, "could not create store")
		return
	}
	c.JSON(http.StatusCreated, st)
}

func (s *Server) updateStore(c *gin.Context) {
	var req storeBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "name is required")
		return
	}
	st, err := store.UpdateStore(c.Request.Context(), s.db, c.Param("id"), req.Name)
	if err == sql.ErrNoRows {
		writeError(c, http.StatusNotFound, "store not found")
		return
	}
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeError(c, http.StatusConflict, "store already exists")
			return
		}
		writeError(c, http.StatusInternalServerError, "could not update store")
		return
	}
	c.JSON(http.StatusOK, st)
}

func (s *Server) deleteStore(c *gin.Context) {
	err := store.DeleteStore(c.Request.Context(), s.db, c.Param("id"))
	if err == sql.ErrNoRows {
		writeError(c, http.StatusNotFound, "store not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "could not delete store")
		return
	}
	c.Status(http.StatusNoContent)
}

func expenseFilter(c *gin.Context) store.ExpenseFilter {
	limit := 0
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	return store.ExpenseFilter{
		UserID:  auth.UserIDFromContext(c),
		StoreID: strings.TrimSpace(c.Query("store_id")),
		From:    strings.TrimSpace(c.Query("from")),
		To:      strings.TrimSpace(c.Query("to")),
		Limit:   limit,
	}
}

func (s *Server) listExpenses(c *gin.Context) {
	list, err := store.ListExpenses(c.Request.Context(), s.db, expenseFilter(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "could not list expenses")
		return
	}
	c.JSON(http.StatusOK, gin.H{"expenses": list})
}

type expenseBody struct {
	StoreID     string  `json:"store_id" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	ExpenseDate string  `json:"expense_date" binding:"required"`
	Note        *string `json:"note"`
}

func (s *Server) createExpense(c *gin.Context) {
	var req expenseBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid expense payload")
		return
	}
	if req.Amount < 0 {
		writeError(c, http.StatusBadRequest, "amount must be non-negative")
		return
	}
	exp, err := store.CreateExpense(
		c.Request.Context(), s.db,
		auth.UserIDFromContext(c),
		req.StoreID, req.ExpenseDate, req.Amount, req.Note,
	)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "could not create expense")
		return
	}
	c.JSON(http.StatusCreated, exp)
}

func (s *Server) updateExpense(c *gin.Context) {
	var req expenseBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid expense payload")
		return
	}
	if req.Amount < 0 {
		writeError(c, http.StatusBadRequest, "amount must be non-negative")
		return
	}
	exp, err := store.UpdateExpense(
		c.Request.Context(), s.db,
		auth.UserIDFromContext(c), c.Param("id"),
		req.StoreID, req.ExpenseDate, req.Amount, req.Note,
	)
	if err == sql.ErrNoRows {
		writeError(c, http.StatusNotFound, "expense not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "could not update expense")
		return
	}
	c.JSON(http.StatusOK, exp)
}

func (s *Server) deleteExpense(c *gin.Context) {
	err := store.DeleteExpense(c.Request.Context(), s.db, auth.UserIDFromContext(c), c.Param("id"))
	if err == sql.ErrNoRows {
		writeError(c, http.StatusNotFound, "expense not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "could not delete expense")
		return
	}
	c.Status(http.StatusNoContent)
}

func dateRange(c *gin.Context) (from, to string) {
	return strings.TrimSpace(c.Query("from")), strings.TrimSpace(c.Query("to"))
}

func (s *Server) statsSummary(c *gin.Context) {
	from, to := dateRange(c)
	stats, err := store.GetSummaryStats(c.Request.Context(), s.db, auth.UserIDFromContext(c), from, to)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "could not load summary")
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (s *Server) statsByStore(c *gin.Context) {
	from, to := dateRange(c)
	stats, err := store.StatsByStore(c.Request.Context(), s.db, auth.UserIDFromContext(c), from, to)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "could not load store stats")
		return
	}
	c.JSON(http.StatusOK, gin.H{"stores": stats})
}
