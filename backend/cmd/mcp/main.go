package main

import (
	geminiAdapter "ComprasResidenciais/internal/adapters/ai/gemini"
	mcpServer "ComprasResidenciais/internal/adapters/handlers/mcp"
	pgRepo "ComprasResidenciais/internal/adapters/repositories/postgres"
	sefazAdapter "ComprasResidenciais/internal/adapters/sefaz"
	"ComprasResidenciais/internal/config"
	"ComprasResidenciais/internal/core/ports"
	"ComprasResidenciais/internal/core/services"
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.LoadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Conexão com o banco de dados
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to parse DATABASE_URL: %v", err)
	}
	poolConfig.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	_ = pgRepo.RegisterPgvectorType(ctx, pool)

	// 2. Repositórios
	userRepository := pgRepo.NewUserRepository(pool)
	storeRepository := pgRepo.NewStoreRepository(pool)
	productRepository := pgRepo.NewCanonicalProductRepository(pool)
	purchaseRepository := pgRepo.NewPurchaseRepository(pool)

	// 3. Provedores de IA e Fiscais
	var gemini *geminiAdapter.GeminiAdapter
	if cfg.GeminiAPIKey != "" {
		g, err := geminiAdapter.NewGeminiAdapter(ctx, cfg.GeminiAPIKey)
		if err == nil {
			gemini = g
		}
	}

	var parser ports.ReceiptParser
	var embedder ports.EmbeddingProvider
	if gemini != nil {
		parser = gemini
		embedder = gemini
	}

	sefaz := sefazAdapter.NewSefazAdapter(cfg.SefazCertPath, cfg.SefazCertPassword)

	// 4. Casos de Uso / Domínio
	purchaseService := services.NewPurchaseService(
		purchaseRepository,
		productRepository,
		storeRepository,
		userRepository,
		parser,
		sefaz,
		embedder,
	)
	analyticsService := services.NewAnalyticsService(purchaseRepository, productRepository)
	userService := services.NewUserService(userRepository)

	// 5. Servidor MCP via STDIO
	server := mcpServer.NewMCPServer(
		purchaseService,
		analyticsService,
		userService,
		storeRepository,
	)

	if err := server.ServeSTDIO(); err != nil {
		log.Fatalf("MCP Server error: %v", err)
	}
}
