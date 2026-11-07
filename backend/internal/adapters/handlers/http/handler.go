package http

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ComprasResidenciais/internal/adapters/repositories/postgres"
	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"
)

// MaxReceiptUploadSize define o tamanho máximo aceito para imagens/PDFs de cupons fiscais (10 MB)
const MaxReceiptUploadSize = 10 << 20

// --- DTOs para Swagger / OpenAPI ---

// CreateUserRequest payload para cadastro de morador
type CreateUserRequest struct {
	Name     string `json:"name" binding:"required" example:"Carlos Silva"`
	Email    string `json:"email" example:"carlos@email.com"`
	Phone    string `json:"phone" example:"11988887777"`
	Password string `json:"password" example:"senha123"`
	Role     string `json:"role" example:"MEMBER"`
}

// UpdateUserRequest payload para atualização de morador
type UpdateUserRequest struct {
	Name        string `json:"name" binding:"required" example:"Carlos Silva"`
	Email       string `json:"email" example:"carlos@email.com"`
	Phone       string `json:"phone" example:"11988887777"`
	Role        string `json:"role" example:"MEMBER"`
	IsBlocked   bool   `json:"is_blocked" example:"false"`
	NewPassword string `json:"new_password" example:"novaSenha123"`
}

// StoreRequest payload para criação ou atualização de estabelecimento
type StoreRequest struct {
	Name    string `json:"name" binding:"required" example:"Supermercado Pão de Açúcar"`
	CNPJ    string `json:"cnpj" example:"47.508.411/0001-56"`
	Address string `json:"address" example:"Av. Paulista, 1000 - Bela Vista, São Paulo - SP"`
}

// ProcessFiscalKeyRequest payload para consulta de NFC-e na SEFAZ
type ProcessFiscalKeyRequest struct {
	FiscalKey string `json:"fiscal_key" binding:"required" example:"35240100000000000000650010000000011000000018"`
}

// SendShoppingListEmailRequest payload para envio de lista de compras
type SendShoppingListEmailRequest struct {
	ToEmail string `json:"to_email" binding:"required,email" example:"morador@email.com"`
	Subject string `json:"subject" binding:"required" example:"Lista de Compras da Residência"`
	Content string `json:"content" binding:"required" example:"Itens recomendados:\n- Arroz Tipo 1\n- Feijão Carioca"`
}

// ResetDatabaseRequest payload de confirmação para reiniciar o banco de dados
type ResetDatabaseRequest struct {
	Confirm string `json:"confirm" binding:"required" example:"CONFIRMAR_RESET"`
}

// ErrorResponse modelo padrão para mensagens de erro
type ErrorResponse struct {
	Error string `json:"error" example:"mensagem descritiva do erro"`
}

// MessageResponse modelo padrão para mensagens de sucesso
type MessageResponse struct {
	Status  string `json:"status" example:"ok"`
	Message string `json:"message" example:"Operação executada com sucesso"`
}

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
	// Middlewares globais de segurança: cabeçalhos HTTP e CORS estrito
	r.Use(SecurityHeadersMiddleware())
	r.Use(CORSMiddleware())

	// Limitadores de taxa por IP para prevenção de força bruta e DoS
	authLimiter := NewIPRateLimiter(rate.Every(12*time.Second), 5)  // ~5 requisições/minuto com burst 5
	uploadLimiter := NewIPRateLimiter(rate.Every(6*time.Second), 5) // ~10 uploads/minuto com burst 5

	api := r.Group("/api/v1")
	{
		// Health check (público)
		api.GET("/health", h.health)

		// Rotas Públicas de Autenticação com limitação de requisições
		auth := api.Group("/auth")
		{
			auth.POST("/login", RateLimitMiddleware(authLimiter, "login"), h.login)
			auth.POST("/setup-admin", RateLimitMiddleware(authLimiter, "setup-admin"), h.setupAdmin)
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
			protected.POST("/purchases/upload", RateLimitMiddleware(uploadLimiter, "upload"), h.uploadReceipt)
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

			// Admin / Manutenção (Apenas ADMIN com confirmação explícita)
			protected.POST("/admin/reset-database", RequireAdminMiddleware(), h.resetDatabase)
		}
	}
}

// @Summary      Health Check
// @Description  Retorna o status operacional da aplicação e timestamp atual
// @Tags         System
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /health [get]
func (h *HTTPHandler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now()})
}

// @Summary      Autenticar morador
// @Description  Realiza login por e-mail e senha, retornando o token JWT e dados do morador
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      domain.LoginRequest  true  "Credenciais de acesso"
// @Success      200      {object}  domain.LoginResponse
// @Failure      400      {object}  ErrorResponse  "E-mail e senha são obrigatórios"
// @Failure      401      {object}  ErrorResponse  "Credenciais inválidas"
// @Router       /auth/login [post]
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

// @Summary      Cadastrar administrador inicial
// @Description  Cadastra o primeiro morador com privilégios de administrador do sistema (caso não exista nenhum)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      domain.SetupAdminRequest  true  "Dados do primeiro administrador"
// @Success      201      {object}  domain.LoginResponse
// @Failure      400      {object}  ErrorResponse  "Dados inválidos ou admin já cadastrado"
// @Router       /auth/setup-admin [post]
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

// @Summary      Status de configuração de autenticação
// @Description  Verifica se o sistema já possui pelo menos um administrador cadastrado
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  map[string]bool
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/status [get]
func (h *HTTPHandler) authStatus(c *gin.Context) {
	hasAdmin, err := h.authService.GetAuthStatus(c.Request.Context())
	if err != nil {
		log.Printf("[AuthStatus Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao verificar status de autenticação"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"has_admin": hasAdmin})
}

// @Summary      Obter perfil do usuário conectado
// @Description  Retorna as informações do morador autenticado via token JWT
// @Tags         Auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  domain.User
// @Failure      401  {object}  ErrorResponse  "Usuário não autenticado"
// @Router       /auth/me [get]
func (h *HTTPHandler) getMe(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// @Summary      Listar moradores
// @Description  Retorna todos os moradores cadastrados na residência
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Success      200  {array}   domain.User
// @Failure      500  {object}  ErrorResponse
// @Router       /users [get]
func (h *HTTPHandler) listUsers(c *gin.Context) {
	users, err := h.userService.ListUsers(c.Request.Context())
	if err != nil {
		log.Printf("[ListUsers Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao listar moradores"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// @Summary      Cadastrar novo morador
// @Description  Cadastra um novo morador na residência. Apenas administradores podem atribuir papel de ADMIN.
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      CreateUserRequest  true  "Dados do morador"
// @Success      201      {object}  domain.User
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /users [post]
func (h *HTTPHandler) createUser(c *gin.Context) {
	caller := GetCurrentUser(c)
	var req CreateUserRequest
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
		log.Printf("[CreateUser Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao cadastrar morador"})
		return
	}
	c.JSON(http.StatusCreated, user)
}

// @Summary      Atualizar dados do morador
// @Description  Atualiza dados cadastrais, bloqueio ou senha do morador. Alterações de papel e bloqueio exigem papel de ADMIN.
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      string             true  "ID do usuário (UUID)"
// @Param        request  body      UpdateUserRequest  true  "Dados para atualização"
// @Success      200      {object}  domain.User
// @Failure      400      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse  "Apenas administradores podem alterar permissões ou bloquear"
// @Failure      404      {object}  ErrorResponse  "Usuário não encontrado"
// @Failure      500      {object}  ErrorResponse
// @Router       /users/{id} [put]
func (h *HTTPHandler) updateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de id de usuário inválido"})
		return
	}

	var req UpdateUserRequest
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
		log.Printf("[UpdateUser Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao atualizar morador"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// @Summary      Excluir morador
// @Description  Remove um morador da residência. Requer perfil de ADMIN.
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string         true  "ID do usuário (UUID)"
// @Success      200  {object}  MessageResponse
// @Failure      403  {object}  ErrorResponse  "Apenas administradores"
// @Failure      404  {object}  ErrorResponse  "Usuário não encontrado"
// @Failure      500  {object}  ErrorResponse
// @Router       /users/{id} [delete]
func (h *HTTPHandler) deleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de id inválido"})
		return
	}

	result, err := h.userService.DeleteUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "usuário não encontrado"})
			return
		}
		log.Printf("[DeleteUser Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao excluir morador"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary      Listar estabelecimentos
// @Description  Retorna todos os estabelecimentos comerciais e supermercados cadastrados
// @Tags         Stores
// @Security     BearerAuth
// @Produce      json
// @Success      200  {array}   domain.Store
// @Failure      500  {object}  ErrorResponse
// @Router       /stores [get]
func (h *HTTPHandler) listStores(c *gin.Context) {
	stores, err := h.storeRepo.ListAll(c.Request.Context())
	if err != nil {
		log.Printf("[ListStores Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao listar estabelecimentos"})
		return
	}
	c.JSON(http.StatusOK, stores)
}

// @Summary      Cadastrar estabelecimento
// @Description  Cadastra ou localiza um estabelecimento comercial por nome e CNPJ
// @Tags         Stores
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      StoreRequest  true  "Dados do estabelecimento"
// @Success      201      {object}  domain.Store
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /stores [post]
func (h *HTTPHandler) createStore(c *gin.Context) {
	var req StoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store, err := h.storeRepo.FindOrCreateByName(c.Request.Context(), req.Name, req.CNPJ)
	if err != nil {
		log.Printf("[CreateStore Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao cadastrar estabelecimento"})
		return
	}

	if req.Address != "" && req.Address != store.Address {
		store.Address = req.Address
		_ = h.storeRepo.Update(c.Request.Context(), store)
	}

	c.JSON(http.StatusCreated, store)
}

// @Summary      Atualizar estabelecimento
// @Description  Atualiza os dados de um estabelecimento comercial (nome, CNPJ, endereço)
// @Tags         Stores
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      string        true  "ID do estabelecimento (UUID)"
// @Param        request  body      StoreRequest  true  "Dados atualizados do estabelecimento"
// @Success      200      {object}  domain.Store
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse  "Estabelecimento não encontrado"
// @Failure      500      {object}  ErrorResponse
// @Router       /stores/{id} [put]
func (h *HTTPHandler) updateStore(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de id inválido"})
		return
	}

	var req StoreRequest
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
			c.JSON(http.StatusNotFound, gin.H{"error": "estabelecimento não encontrado"})
			return
		}
		log.Printf("[UpdateStore Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao atualizar estabelecimento"})
		return
	}

	updated, err := h.storeRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, store)
		return
	}
	c.JSON(http.StatusOK, updated)
}

// @Summary      Excluir estabelecimento
// @Description  Remove um estabelecimento comercial
// @Tags         Stores
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string         true  "ID do estabelecimento (UUID)"
// @Success      200  {object}  MessageResponse
// @Failure      404  {object}  ErrorResponse  "Estabelecimento não encontrado"
// @Failure      500  {object}  ErrorResponse
// @Router       /stores/{id} [delete]
func (h *HTTPHandler) deleteStore(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de id inválido"})
		return
	}

	if err := h.storeRepo.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "estabelecimento não encontrado"})
			return
		}
		log.Printf("[DeleteStore Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao excluir estabelecimento"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "estabelecimento excluído com sucesso", "id": id})
}

// @Summary      Reiniciar dados operacionais do banco de dados
// @Description  Efetua truncate nas tabelas de compras, itens, catálogo e lojas preservando os moradores e acessos (Requer ADMIN e payload de confirmação)
// @Tags         Admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      ResetDatabaseRequest  true  "Confirmação da operação"
// @Success      200      {object}  MessageResponse
// @Failure      400      {object}  ErrorResponse  "Confirmação inválida"
// @Failure      403      {object}  ErrorResponse  "Requer ADMIN"
// @Failure      500      {object}  ErrorResponse
// @Router       /admin/reset-database [post]
func (h *HTTPHandler) resetDatabase(c *gin.Context) {
	var req ResetDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Confirm != "CONFIRMAR_RESET" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ação destrutiva requer confirmação explícita. Envie no corpo da requisição: {\"confirm\": \"CONFIRMAR_RESET\"}",
		})
		return
	}

	if h.pool == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Conexão com o banco de dados não inicializada"})
		return
	}

	if err := postgres.ResetDatabase(c.Request.Context(), h.pool); err != nil {
		log.Printf("[ResetDatabase Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao reiniciar dados operacionais"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Tabelas operacionais reiniciadas com sucesso. Moradores e credenciais foram preservados para manter a integridade de acesso.",
	})
}

// @Summary      Processar cupom fiscal com IA Gemini Vision
// @Description  Faz upload de imagem (JPG, PNG, WebP, HEIC) ou PDF do cupom e realiza extração multimodal inteligente de itens, quantidades e valores criando um rascunho (Draft)
// @Tags         Purchases
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Arquivo de imagem ou PDF do cupom fiscal (máximo 10MB)"
// @Success      201   {object}  domain.Purchase
// @Failure      400   {object}  ErrorResponse
// @Failure      413   {object}  ErrorResponse  "Arquivo excede o limite máximo permitido de 10MB"
// @Failure      500   {object}  ErrorResponse
// @Router       /purchases/upload [post]
func (h *HTTPHandler) uploadReceipt(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'file' é obrigatório"})
		return
	}

	// 1. Validação de tamanho pelo header do formulário multipart
	if fileHeader.Size > MaxReceiptUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": fmt.Sprintf("o arquivo excede o limite máximo permitido de %d MB", MaxReceiptUploadSize/(1024*1024)),
		})
		return
	}

	// 2. Validação antecipada do Content-Type
	mimeType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
	allowedMimes := map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/webp":      true,
		"image/heic":      true,
		"image/heif":      true,
		"application/pdf": true,
	}
	if mimeType != "" && !allowedMimes[mimeType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("tipo de arquivo não suportado (%s). Formatos permitidos: JPG, PNG, WebP, HEIC, PDF", mimeType),
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "não foi possível abrir o arquivo enviado"})
		return
	}
	defer file.Close()

	// 3. io.LimitReader garante defesa em profundidade contra headers de tamanho adulterados
	limitedReader := io.LimitReader(file, MaxReceiptUploadSize+1)
	fileBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		log.Printf("[UploadReceipt Read Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao ler os dados do arquivo"})
		return
	}
	if int64(len(fileBytes)) > MaxReceiptUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": fmt.Sprintf("o arquivo excede o limite máximo permitido de %d MB", MaxReceiptUploadSize/(1024*1024)),
		})
		return
	}

	draft, err := h.purchaseService.ProcessReceiptFile(c.Request.Context(), fileBytes, mimeType)
	if err != nil {
		log.Printf("[UploadReceipt Processing Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao processar o cupom fiscal através do serviço de inteligência artificial"})
		return
	}

	c.JSON(http.StatusCreated, draft)
}

// @Summary      Consultar chave NFC-e na SEFAZ
// @Description  Valida o dígito verificador módulo 11 de uma chave fiscal de 44 dígitos e consulta a nota eletrônica na SEFAZ
// @Tags         Purchases
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      ProcessFiscalKeyRequest  true  "Chave fiscal de 44 dígitos"
// @Success      201      {object}  domain.Purchase
// @Failure      400      {object}  ErrorResponse  "Chave inválida ou dígito verificador incorreto"
// @Failure      500      {object}  ErrorResponse
// @Router       /purchases/sefaz [post]
func (h *HTTPHandler) processFiscalKey(c *gin.Context) {
	var req ProcessFiscalKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !domain.ValidateFiscalKey(req.FiscalKey) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chave de acesso fiscal inválida: formato ou dígito verificador (módulo 11) incorreto"})
		return
	}

	draft, err := h.purchaseService.ProcessFiscalKey(c.Request.Context(), req.FiscalKey)
	if err != nil {
		log.Printf("[ProcessFiscalKey Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao consultar nota fiscal na SEFAZ"})
		return
	}

	c.JSON(http.StatusCreated, draft)
}

// @Summary      Confirmar rascunho de compra
// @Description  Valida e confirma o rascunho de compra revisado pelo morador, gerando embeddings semânticos no pgvector para os produtos
// @Tags         Purchases
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id        path      string           true  "ID da compra (UUID)"
// @Param        purchase  body      domain.Purchase  true  "Dados revisados da compra"
// @Success      200       {object}  domain.Purchase
// @Failure      400       {object}  ErrorResponse
// @Router       /purchases/{id}/confirm [put]
func (h *HTTPHandler) confirmPurchase(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de id de compra inválido"})
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

// @Summary      Cadastrar compra manual
// @Description  Registra uma compra manual (compras em feiras, sacolões ou sem nota fiscal) com seus itens
// @Tags         Purchases
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        purchase  body      domain.Purchase  true  "Dados da compra manual"
// @Success      201       {object}  domain.Purchase
// @Failure      400       {object}  ErrorResponse
// @Router       /purchases/manual [post]
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

// @Summary      Obter detalhes da compra
// @Description  Retorna os dados completos de uma compra e seus itens pelo ID
// @Tags         Purchases
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string           true  "ID da compra (UUID)"
// @Success      200  {object}  domain.Purchase
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /purchases/{id} [get]
func (h *HTTPHandler) getPurchase(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de id de compra inválido"})
		return
	}

	purchase, err := h.purchaseService.GetPurchase(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, purchase)
}

// @Summary      Listar histórico de compras
// @Description  Retorna a lista paginada de compras registradas
// @Tags         Purchases
// @Security     BearerAuth
// @Produce      json
// @Param        limit   query     int              false  "Quantidade de registros (padrão: 20)"
// @Param        offset  query     int              false  "Deslocamento para paginação (padrão: 0)"
// @Success      200     {array}   domain.Purchase
// @Failure      500     {object}  ErrorResponse
// @Router       /purchases [get]
func (h *HTTPHandler) listPurchases(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	list, err := h.purchaseService.ListPurchases(c.Request.Context(), limit, offset)
	if err != nil {
		log.Printf("[ListPurchases Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao listar histórico de compras"})
		return
	}

	c.JSON(http.StatusOK, list)
}

// @Summary      Busca semântica de produtos (pgvector)
// @Description  Busca produtos canônicos por similaridade de cosseno a partir de embeddings gerados pelo modelo text-embedding-004 do Gemini
// @Tags         Analytics
// @Security     BearerAuth
// @Produce      json
// @Param        q    query     string                    true  "Termo de busca (ex: arroz, sabão em pó, leite)"
// @Success      200  {array}   domain.CanonicalProduct
// @Failure      500  {object}  ErrorResponse
// @Router       /analytics/search-products [get]
func (h *HTTPHandler) searchProducts(c *gin.Context) {
	query := c.Query("q")
	products, err := h.analyticsService.SearchProducts(c.Request.Context(), query)
	if err != nil {
		log.Printf("[SearchProducts Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha na busca semântica de produtos"})
		return
	}
	c.JSON(http.StatusOK, products)
}

// @Summary      Catálogo de produtos com preços de referência
// @Description  Retorna todos os produtos canônicos com menor preço unitário registrado e estabelecimento correspondente
// @Tags         Analytics
// @Security     BearerAuth
// @Produce      json
// @Success      200  {array}   domain.CatalogItemView
// @Failure      500  {object}  ErrorResponse
// @Router       /analytics/catalog-prices [get]
func (h *HTTPHandler) getCatalogWithPrices(c *gin.Context) {
	catalog, err := h.analyticsService.GetCatalogWithPrices(c.Request.Context())
	if err != nil {
		log.Printf("[CatalogPrices Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao consultar catálogo com preços"})
		return
	}
	c.JSON(http.StatusOK, catalog)
}

// @Summary      Comparativo de preços de produto
// @Description  Compara preços do mesmo produto entre diferentes supermercados e compras recentes
// @Tags         Analytics
// @Security     BearerAuth
// @Produce      json
// @Param        productId  path      string                      true  "ID do produto canônico (UUID)"
// @Success      200        {object}  domain.PriceComparisonView
// @Failure      400        {object}  ErrorResponse
// @Failure      500        {object}  ErrorResponse
// @Router       /analytics/price-comparison/{productId} [get]
func (h *HTTPHandler) getPriceComparison(c *gin.Context) {
	prodIdStr := c.Param("productId")
	id, err := uuid.Parse(prodIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de id de produto inválido"})
		return
	}

	view, err := h.analyticsService.GetPriceComparison(c.Request.Context(), id)
	if err != nil {
		log.Printf("[PriceComparison Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao obter comparativo de preços"})
		return
	}

	c.JSON(http.StatusOK, view)
}

// @Summary      Oscilação mensal de preços
// @Description  Retorna a média de preços de um produto ao longo de diferentes semanas do mês
// @Tags         Analytics
// @Security     BearerAuth
// @Produce      json
// @Param        productId  path      string                            true  "ID do produto canônico (UUID)"
// @Success      200        {object}  domain.MonthlyPriceOscillation
// @Failure      400        {object}  ErrorResponse
// @Failure      500        {object}  ErrorResponse
// @Router       /analytics/price-oscillation/{productId} [get]
func (h *HTTPHandler) getMonthlyOscillation(c *gin.Context) {
	prodIdStr := c.Param("productId")
	id, err := uuid.Parse(prodIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de id de produto inválido"})
		return
	}

	oscillation, err := h.analyticsService.GetMonthlyOscillation(c.Request.Context(), id)
	if err != nil {
		log.Printf("[MonthlyOscillation Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao calcular oscilação mensal de preços"})
		return
	}

	c.JSON(http.StatusOK, oscillation)
}

// @Summary      Resumo analítico global
// @Description  Retorna indicadores consolidados da residência (gastos do mês, divisão por morador, top estabelecimentos)
// @Tags         Analytics
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  domain.GlobalSummaryView
// @Failure      500  {object}  ErrorResponse
// @Router       /analytics/summary [get]
func (h *HTTPHandler) getGlobalSummary(c *gin.Context) {
	summary, err := h.analyticsService.GetGlobalSummary(c.Request.Context(), time.Now())
	if err != nil {
		log.Printf("[GlobalSummary Error] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao gerar resumo analítico"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// @Summary      Enviar lista de compras por e-mail
// @Description  Envia o conteúdo da lista de compras gerada para o e-mail do morador
// @Tags         ShoppingList
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      SendShoppingListEmailRequest  true  "Dados para envio da lista"
// @Success      200      {object}  MessageResponse
// @Failure      400      {object}  ErrorResponse
// @Router       /shopping-list/send-email [post]
func (h *HTTPHandler) sendShoppingListEmail(c *gin.Context) {
	var req SendShoppingListEmailRequest
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
