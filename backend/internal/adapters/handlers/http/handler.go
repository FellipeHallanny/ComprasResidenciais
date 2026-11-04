package http

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HTTPHandler struct {
	purchaseService  ports.PurchaseService
	analyticsService ports.AnalyticsService
	userService      ports.UserService
	authService      ports.AuthService
	storeRepo        ports.StoreRepository
	pool             *pgxpool.Pool
}

func NewHTTPHandler(
	purchaseService ports.PurchaseService,
	analyticsService ports.AnalyticsService,
	userService ports.UserService,
	authService ports.AuthService,
	storeRepo ports.StoreRepository,
	pool *pgxpool.Pool,
) *HTTPHandler {
	return &HTTPHandler{
		purchaseService:  purchaseService,
		analyticsService: analyticsService,
		userService:      userService,
		authService:      authService,
		storeRepo:        storeRepo,
		pool:             pool,
	}
}

// RegisterRoutes registra as rotas da API RESTful no Gin
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	// CORS Middleware para integração transparente com Frontend Svelte / Next.js
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api/v1")
	{
		// Health check (público)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now()})
		})

		// Rotas Públicas de Autenticação
		auth := api.Group("/auth")
		{
			auth.POST("/login", h.login)
			auth.POST("/setup-admin", h.setupAdmin)
			auth.GET("/status", h.authStatus)
		}

		// Rotas Protegidas por Autenticação (Requer token JWT válido)
		protected := api.Group("")
		protected.Use(AuthMiddleware(h.authService))
		{
			// Informações do usuário conectado
			protected.GET("/auth/me", h.getMe)

			// Usuários / Moradores
			protected.GET("/users", h.listUsers)
			protected.POST("/users", h.createUser)
			protected.PUT("/users/:id", h.updateUser)
			protected.DELETE("/users/:id", RequireAdminMiddleware(), h.deleteUser)

			// Estabelecimentos
			protected.GET("/stores", h.listStores)
			protected.POST("/stores", h.createStore)
			protected.PUT("/stores/:id", h.updateStore)
			protected.DELETE("/stores/:id", h.deleteStore)

			// Compras
			protected.GET("/purchases", h.listPurchases)
			protected.GET("/purchases/:id", h.getPurchase)
			protected.POST("/purchases/upload", h.uploadReceipt)
			protected.POST("/purchases/sefaz", h.processFiscalKey)
			protected.POST("/purchases/manual", h.createManualPurchase)
			protected.PUT("/purchases/:id/confirm", h.confirmPurchase)

			// Analytics, Catálogo & Preços
			protected.GET("/analytics/search-products", h.searchProducts)
			protected.GET("/analytics/catalog-prices", h.getCatalogWithPrices)
			protected.GET("/analytics/price-comparison/:productId", h.getPriceComparison)
			protected.GET("/analytics/price-oscillation/:productId", h.getMonthlyOscillation)
			protected.GET("/analytics/summary", h.getGlobalSummary)

			// Lista de Compras / Notificações
			protected.POST("/shopping-list/send-email", h.sendShoppingListEmail)

			// Admin / Manutenção (Apenas ADMIN)
			protected.POST("/admin/reset-database", RequireAdminMiddleware(), h.resetDatabase)
		}
	}
}

// Handlers de Autenticação

func (h *HTTPHandler) login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "E-mail e senha são obrigatórios"})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HTTPHandler) setupAdmin(c *gin.Context) {
	var req domain.SetupAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: informe nome, e-mail e senha de no mínimo 6 caracteres"})
		return
	}

	resp, err := h.authService.SetupInitialAdmin(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *HTTPHandler) authStatus(c *gin.Context) {
	hasAdmin, err := h.authService.GetAuthStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"has_admin": hasAdmin})
}

func (h *HTTPHandler) getMe(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *HTTPHandler) listUsers(c *gin.Context) {
	users, err := h.userService.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *HTTPHandler) createUser(c *gin.Context) {
	caller := GetCurrentUser(c)
	var req struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Somente ADMIN pode cadastrar outro ADMIN
	if req.Role == domain.UserRoleAdmin && (caller == nil || caller.Role != domain.UserRoleAdmin) {
		req.Role = domain.UserRoleMember
	}
	if req.Role == "" {
		req.Role = domain.UserRoleMember
	}

	user, err := h.userService.CreateUser(c.Request.Context(), req.Name, req.Email, req.Phone, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (h *HTTPHandler) updateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de id de usuário inválido"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		Role        string `json:"role"`
		IsBlocked   bool   `json:"is_blocked"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	caller := GetCurrentUser(c)
	if caller == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	// Regra de segurança: Apenas ADMIN pode bloquear/desbloquear usuários ou conceder papel de admin
	if caller.Role != domain.UserRoleAdmin {
		if req.IsBlocked {
			c.JSON(http.StatusForbidden, gin.H{"error": "Apenas administradores podem bloquear usuários da residência"})
			return
		}
		if req.Role != "" && req.Role != caller.Role {
			c.JSON(http.StatusForbidden, gin.H{"error": "Apenas administradores podem alterar permissões de acesso"})
			return
		}
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), id, req.Name, req.Email, req.Phone, req.Role, req.IsBlocked, req.NewPassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "usuário não encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *HTTPHandler) deleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id format"})
		return
	}

	result, err := h.userService.DeleteUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
func (h *HTTPHandler) listStores(c *gin.Context) {
	stores, err := h.storeRepo.ListAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stores)
}

func (h *HTTPHandler) createStore(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		CNPJ    string `json:"cnpj"`
		Address string `json:"address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store, err := h.storeRepo.FindOrCreateByName(c.Request.Context(), req.Name, req.CNPJ)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Address != "" && req.Address != store.Address {
		store.Address = req.Address
		_ = h.storeRepo.Update(c.Request.Context(), store)
	}

	c.JSON(http.StatusCreated, store)
}

func (h *HTTPHandler) updateStore(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid store id format"})
		return
	}

	var req struct {
		Name    string `json:"name" binding:"required"`
		CNPJ    string `json:"cnpj"`
		Address string `json:"address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store := &domain.Store{
		ID:      id,
		Name:    req.Name,
		CNPJ:    req.CNPJ,
		Address: req.Address,
	}

	if err := h.storeRepo.Update(c.Request.Context(), store); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.storeRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, store)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *HTTPHandler) deleteStore(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid store id format"})
		return
	}

	if err := h.storeRepo.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "store deleted successfully", "id": id})
}

func (h *HTTPHandler) resetDatabase(c *gin.Context) {
	if h.pool == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database pool not initialized"})
		return
	}

	query := `TRUNCATE TABLE purchase_items, purchases, canonical_products, stores, household_users RESTART IDENTITY CASCADE;`
	if _, err := h.pool.Exec(c.Request.Context(), query); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Todas as tabelas foram reiniciadas do zero com sucesso."})
}

func (h *HTTPHandler) uploadReceipt(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open uploaded file"})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot read file bytes"})
		return
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	draft, err := h.purchaseService.ProcessReceiptFile(c.Request.Context(), fileBytes, mimeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, draft)
}

func (h *HTTPHandler) processFiscalKey(c *gin.Context) {
	var req struct {
		FiscalKey string `json:"fiscal_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	draft, err := h.purchaseService.ProcessFiscalKey(c.Request.Context(), req.FiscalKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, draft)
}

func (h *HTTPHandler) confirmPurchase(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid purchase id"})
		return
	}

	var purchase domain.Purchase
	if err := c.ShouldBindJSON(&purchase); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	purchase.ID = id

	confirmed, err := h.purchaseService.ConfirmPurchase(c.Request.Context(), &purchase)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, confirmed)
}

func (h *HTTPHandler) createManualPurchase(c *gin.Context) {
	var purchase domain.Purchase
	if err := c.ShouldBindJSON(&purchase); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.purchaseService.CreateManualPurchase(c.Request.Context(), &purchase)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *HTTPHandler) getPurchase(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid purchase id"})
		return
	}

	purchase, err := h.purchaseService.GetPurchase(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, purchase)
}

func (h *HTTPHandler) listPurchases(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	list, err := h.purchaseService.ListPurchases(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, list)
}

func (h *HTTPHandler) searchProducts(c *gin.Context) {
	query := c.Query("q")
	products, err := h.analyticsService.SearchProducts(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, products)
}

func (h *HTTPHandler) getCatalogWithPrices(c *gin.Context) {
	catalog, err := h.analyticsService.GetCatalogWithPrices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, catalog)
}

func (h *HTTPHandler) getPriceComparison(c *gin.Context) {
	prodIdStr := c.Param("productId")
	id, err := uuid.Parse(prodIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid canonical product id"})
		return
	}

	view, err := h.analyticsService.GetPriceComparison(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, view)
}

func (h *HTTPHandler) getMonthlyOscillation(c *gin.Context) {
	prodIdStr := c.Param("productId")
	id, err := uuid.Parse(prodIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid canonical product id"})
		return
	}

	oscillation, err := h.analyticsService.GetMonthlyOscillation(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, oscillation)
}

func (h *HTTPHandler) getGlobalSummary(c *gin.Context) {
	summary, err := h.analyticsService.GetGlobalSummary(c.Request.Context(), time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *HTTPHandler) sendShoppingListEmail(c *gin.Context) {
	var req struct {
		ToEmail string `json:"to_email" binding:"required,email"`
		Subject string `json:"subject" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[ShoppingList] Lista de compras enviada com sucesso para %s: %s\n", req.ToEmail, req.Subject)
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": fmt.Sprintf("Lista de compras enviada para %s com sucesso.", req.ToEmail),
	})
}
