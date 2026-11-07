package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPServer struct {
	server           *server.MCPServer
	purchaseService  ports.PurchaseService
	analyticsService ports.AnalyticsService
	userService      ports.UserService
	storeRepo        ports.StoreRepository
}

func NewMCPServer(
	purchaseService ports.PurchaseService,
	analyticsService ports.AnalyticsService,
	userService ports.UserService,
	storeRepo ports.StoreRepository,
) *MCPServer {
	s := server.NewMCPServer(
		"ComprasResidenciais-MCP",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
	)

	handler := &MCPServer{
		server:           s,
		purchaseService:  purchaseService,
		analyticsService: analyticsService,
		userService:      userService,
		storeRepo:        storeRepo,
	}

	handler.registerTools()
	return handler
}

func (s *MCPServer) ServeSTDIO() error {
	return server.ServeStdio(s.server)
}

func (s *MCPServer) registerTools() {
	// Ferramenta 1: Processar nota fiscal a partir de imagem em Base64
	s.server.AddTool(
		mcp.NewTool(
			"parse_receipt_image",
			mcp.WithDescription("Analisa imagem ou PDF de cupom fiscal em Base64, extrai itens, calcula normalização de preços e cria rascunho de compra."),
			mcp.WithString("file_base64", mcp.Required(), mcp.Description("Conteúdo do arquivo codificado em base64")),
			mcp.WithString("mime_type", mcp.Description("MIME type do arquivo (image/jpeg, image/png, application/pdf)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			b64, err := request.RequireString("file_base64")
			if err != nil || b64 == "" {
				return mcp.NewToolResultError("file_base64 is required"), nil
			}
			mimeType := request.GetString("mime_type", "image/jpeg")

			// Limite de segurança: 10MB decodificado equivale a ~14-15MB em base64
			const maxB64Len = 15 * 1024 * 1024
			if len(b64) > maxB64Len {
				return mcp.NewToolResultError("file_base64 excede o limite máximo permitido de 10MB"), nil
			}

			fileBytes, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid base64: %v", err)), nil
			}
			if len(fileBytes) > 10*1024*1024 {
				return mcp.NewToolResultError("arquivo decodificado excede o limite máximo permitido de 10MB"), nil
			}

			draft, err := s.purchaseService.ProcessReceiptFile(ctx, fileBytes, mimeType)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error processing receipt: %v", err)), nil
			}

			bytes, _ := json.MarshalIndent(draft, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// Ferramenta 2: Comparar preços e equivalência de embalagens para um produto
	s.server.AddTool(
		mcp.NewTool(
			"compare_item_prices",
			mcp.WithDescription("Pesquisa onde determinado produto canônico foi comprado mais barato (preço normalizado por kg/L/unidade) e histórico de oscilação."),
			mcp.WithString("product_name", mcp.Required(), mcp.Description("Nome ou termo de busca do produto (ex: 'Arroz', 'Leite Integral')")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, err := request.RequireString("product_name")
			if err != nil || query == "" {
				return mcp.NewToolResultError("product_name is required"), nil
			}

			products, err := s.analyticsService.SearchProducts(ctx, query)
			if err != nil || len(products) == 0 {
				return mcp.NewToolResultText(fmt.Sprintf("Nenhum produto cadastrado com o termo '%s'", query)), nil
			}

			canonical := products[0]
			comparison, err := s.analyticsService.GetPriceComparison(ctx, canonical.ID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("erro ao comparar preços: %v", err)), nil
			}

			bytes, _ := json.MarshalIndent(comparison, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// Ferramenta 3: Obter oscilação de preços por época do mês
	s.server.AddTool(
		mcp.NewTool(
			"get_monthly_price_trends",
			mcp.WithDescription("Analisa a oscilação do preço médio de um produto ao longo dos períodos do mês (dias 1-7, 8-14, 15-21, 22-31)."),
			mcp.WithString("product_name", mcp.Required(), mcp.Description("Nome do produto a ser analisado")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, err := request.RequireString("product_name")
			if err != nil || query == "" {
				return mcp.NewToolResultError("product_name is required"), nil
			}

			products, err := s.analyticsService.SearchProducts(ctx, query)
			if err != nil || len(products) == 0 {
				return mcp.NewToolResultText(fmt.Sprintf("Produto '%s' não localizado", query)), nil
			}

			oscillation, err := s.analyticsService.GetMonthlyOscillation(ctx, products[0].ID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("erro ao buscar tendências: %v", err)), nil
			}

			bytes, _ := json.MarshalIndent(oscillation, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// Ferramenta 4: Confirmar compra atribuindo morador/comprador
	s.server.AddTool(
		mcp.NewTool(
			"confirm_purchase",
			mcp.WithDescription("Confirma um rascunho de compra associando o comprador morador da residência."),
			mcp.WithString("purchase_id", mcp.Required(), mcp.Description("ID UUID da compra em rascunho")),
			mcp.WithString("user_id", mcp.Required(), mcp.Description("ID UUID do morador que comprou")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			purchaseIDStr, err := request.RequireString("purchase_id")
			if err != nil {
				return mcp.NewToolResultError("purchase_id is required"), nil
			}
			userIDStr, err := request.RequireString("user_id")
			if err != nil {
				return mcp.NewToolResultError("user_id is required"), nil
			}

			purchaseID, err := uuid.Parse(purchaseIDStr)
			if err != nil {
				return mcp.NewToolResultError("invalid purchase_id"), nil
			}
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return mcp.NewToolResultError("invalid user_id"), nil
			}

			purchase, err := s.purchaseService.GetPurchase(ctx, purchaseID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("purchase not found: %v", err)), nil
			}

			purchase.UserID = userID
			confirmed, err := s.purchaseService.ConfirmPurchase(ctx, purchase)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error confirming purchase: %v", err)), nil
			}

			bytes, _ := json.MarshalIndent(confirmed, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// Ferramenta 5: Listar moradores
	s.server.AddTool(
		mcp.NewTool(
			"list_household_users",
			mcp.WithDescription("Lista os moradores cadastrados na residência para seleção de comprador."),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			users, err := s.userService.ListUsers(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error: %v", err)), nil
			}
			bytes, _ := json.MarshalIndent(users, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// Ferramenta 6: Deletar ou Bloquear morador
	s.server.AddTool(
		mcp.NewTool(
			"delete_household_user",
			mcp.WithDescription("Exclui um morador da residência. Se houver compras associadas a ele, ele é bloqueado para manter o histórico; se não houver compras, é deletado permanentemente."),
			mcp.WithString("user_id", mcp.Required(), mcp.Description("ID UUID do morador a ser processado")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			userIDStr, err := request.RequireString("user_id")
			if err != nil {
				return mcp.NewToolResultError("user_id is required"), nil
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return mcp.NewToolResultError("invalid user_id format"), nil
			}

			result, err := s.userService.DeleteUser(ctx, userID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error: %v", err)), nil
			}

			bytes, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// Ferramenta 7: Resumo financeiro do mês
	s.server.AddTool(
		mcp.NewTool(
			"get_global_monthly_summary",
			mcp.WithDescription("Retorna total de compras do mês atual, divisão de gastos por morador e mercados mais visitados."),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			summary, err := s.analyticsService.GetGlobalSummary(ctx, time.Now())
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error: %v", err)), nil
			}
			bytes, _ := json.MarshalIndent(summary, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)

	// Ferramenta 8: Registro manual de compra
	s.server.AddTool(
		mcp.NewTool(
			"create_manual_purchase",
			mcp.WithDescription("Registra manualmente uma compra contendo múltiplos itens."),
			mcp.WithString("store_name", mcp.Required(), mcp.Description("Nome do estabelecimento")),
			mcp.WithString("user_id", mcp.Required(), mcp.Description("UUID do comprador")),
			mcp.WithString("items_json", mcp.Required(), mcp.Description("Array JSON de itens com raw_description, quantity, package_size, package_unit, total_price")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			storeName, err := request.RequireString("store_name")
			if err != nil {
				return mcp.NewToolResultError("store_name is required"), nil
			}
			userIDStr, err := request.RequireString("user_id")
			if err != nil {
				return mcp.NewToolResultError("user_id is required"), nil
			}
			itemsJSON, err := request.RequireString("items_json")
			if err != nil {
				return mcp.NewToolResultError("items_json is required"), nil
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return mcp.NewToolResultError("invalid user_id"), nil
			}

			store, err := s.storeRepo.FindOrCreateByName(ctx, storeName, "")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("store error: %v", err)), nil
			}

			var items []domain.PurchaseItem
			if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid items_json: %v", err)), nil
			}

			purchase := &domain.Purchase{
				UserID:      userID,
				StoreID:     store.ID,
				PurchasedAt: time.Now(),
				Items:       items,
			}

			created, err := s.purchaseService.CreateManualPurchase(ctx, purchase)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to save: %v", err)), nil
			}

			bytes, _ := json.MarshalIndent(created, "", "  ")
			return mcp.NewToolResultText(string(bytes)), nil
		},
	)
}
