package http

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type HTTPHandler struct {
	purchaseService  ports.PurchaseService
	analyticsService ports.AnalyticsService
	userService      ports.UserService
	storeRepo        ports.StoreRepository
}

func NewHTTPHandler(
	purchaseService ports.PurchaseService,
	analyticsService ports.AnalyticsService,
	userService ports.UserService,
	storeRepo ports.StoreRepository,
) *HTTPHandler {
	return &HTTPHandler{
		purchaseService:  purchaseService,
		analyticsService: analyticsService,
		userService:      userService,
		storeRepo:        storeRepo,
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
		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now()})
		})

		// Usuários / Moradores
		users := api.Group("/users")
		{
			users.GET("", h.listUsers)
			users.POST("", h.createUser)
			users.DELETE("/:id", h.deleteUser)
		}

		// Estabelecimentos
		stores := api.Group("/stores")
		{
			stores.GET("", h.listStores)
			stores.POST("", h.createStore)
		}

		// Compras
		purchases := api.Group("/purchases")
		{
			purchases.GET("", h.listPurchases)
			purchases.GET("/:id", h.getPurchase)
			purchases.POST("/upload", h.uploadReceipt)
			purchases.POST("/sefaz", h.processFiscalKey)
			purchases.POST("/manual", h.createManualPurchase)
			purchases.PUT("/:id/confirm", h.confirmPurchase)
		}

		// Analytics & Preços
		analytics := api.Group("/analytics")
		{
			analytics.GET("/search-products", h.searchProducts)
			analytics.GET("/price-comparison/:productId", h.getPriceComparison)
			analytics.GET("/price-oscillation/:productId", h.getMonthlyOscillation)
			analytics.GET("/summary", h.getGlobalSummary)
		}
	}
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
	var req struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), req.Name, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, user)
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
		Name string `json:"name" binding:"required"`
		CNPJ string `json:"cnpj"`
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

	c.JSON(http.StatusCreated, store)
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
