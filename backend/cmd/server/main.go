package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "ComprasResidenciais/docs"
	geminiAdapter "ComprasResidenciais/internal/adapters/ai/gemini"
	httpHandler "ComprasResidenciais/internal/adapters/handlers/http"
	pgRepo "ComprasResidenciais/internal/adapters/repositories/postgres"
	sefazAdapter "ComprasResidenciais/internal/adapters/sefaz"
	"ComprasResidenciais/internal/config"
	"ComprasResidenciais/internal/core/ports"
	"ComprasResidenciais/internal/core/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Compras Residenciais API
// @version         1.0
// @description     API RESTful para controle de compras residenciais inteligentes com IA Gemini Multimodal, busca semântica pgvector e integração com NFC-e SEFAZ.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Suporte Compras Residenciais
// @contact.email  contato@comprasresidenciais.local

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Digite "Bearer " seguido do token JWT recebido no login (ex: Bearer eyJhbGci...).
func main() {
	cfg := config.LoadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Conexão com o banco de dados PostgreSQL
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to parse DATABASE_URL: %v", err)
	}
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 1 * time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Unable to connect to database pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Printf("Warning: Database ping failed: %v (continuing startup)", err)
	} else {
		log.Println("Database connection established successfully.")
		// Registra tipos do pgvector no pgxpool
		if err := pgRepo.RegisterPgvectorType(ctx, pool); err != nil {
			log.Printf("Warning: Failed to register pgvector type: %v", err)
		}
		// Executa migrações automáticas incrementais / esquema inicial
		if err := pgRepo.RunAutoMigrations(ctx, pool); err != nil {
			log.Printf("Warning: Auto migrations failed: %v", err)
		}
	}

	// 2. Instanciação dos Repositórios (Driven Adapters)
	userRepository := pgRepo.NewUserRepository(pool)
	storeRepository := pgRepo.NewStoreRepository(pool)
	productRepository := pgRepo.NewCanonicalProductRepository(pool)
	purchaseRepository := pgRepo.NewPurchaseRepository(pool)

	// 3. Provedores de IA e Fiscais
	var gemini *geminiAdapter.GeminiAdapter
	if cfg.GeminiAPIKey != "" {
		g, err := geminiAdapter.NewGeminiAdapter(ctx, cfg.GeminiAPIKey)
		if err != nil {
			log.Printf("Warning: failed to initialize Gemini adapter: %v", err)
		} else {
			gemini = g
			log.Println("Gemini Multimodal & Embeddings Adapter initialized.")
		}
	} else {
		log.Println("Notice: GEMINI_API_KEY not set. Multimodal vision and semantic embeddings will be mock/disabled.")
	}

	var parser ports.ReceiptParser
	var embedder ports.EmbeddingProvider
	if gemini != nil {
		parser = gemini
		embedder = gemini
	}

	sefaz := sefazAdapter.NewSefazAdapter(cfg.SefazCertPath, cfg.SefazCertPassword)

	// 4. Casos de Uso / Domínio (Core Services)
	purchaseService := services.NewPurchaseService(
		purchaseRepository,
		productRepository,
		storeRepository,
		userRepository,
		parser,
		sefaz,
		embedder,
	)
	analyticsService := services.NewAnalyticsService(purchaseRepository, productRepository, embedder)
	userService := services.NewUserService(userRepository)
	authService := services.NewAuthService(userRepository, cfg.JWTSecret)

	// 5. Adaptador HTTP Gin (Driving Adapter)
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.MaxMultipartMemory = 12 << 20 // 12 MB limite de buffer para uploads multipart

	// Documentação Interativa Swagger / OpenAPI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	handler := httpHandler.NewHTTPHandler(purchaseService, analyticsService, userService, authService, storeRepository, pool)
	handler.RegisterRoutes(router)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// Graceful shutdown em background
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		sig := <-quit
		log.Printf("Encerrando servidor com segurança (sinal: %v)...\n", sig)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Erro no encerramento: %v\n", err)
		}
	}()

	log.Printf("Servidor HTTP REST ouvindo na porta %s (http://localhost:%s)", cfg.Port, cfg.Port)
	log.Printf("Documentação Swagger UI disponível em http://localhost:%s/swagger/index.html", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}
	log.Println("Servidor finalizado com sucesso.")
}
