package ports

import (
	"context"
	"time"

	"ComprasResidenciais/internal/core/domain"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// DeleteUserResult descreve a ação tomada ao solicitar a remoção de um usuário
type DeleteUserAction string

const (
	UserActionDeleted DeleteUserAction = "DELETED"
	UserActionBlocked DeleteUserAction = "BLOCKED"
)

type DeleteUserResult struct {
	Action  DeleteUserAction `json:"action"`
	Message string           `json:"message"`
	UserID  uuid.UUID        `json:"user_id"`
}

// UserRepository define o contrato para armazenamento de moradores/usuários
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	ListAll(ctx context.Context) ([]domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Block(ctx context.Context, id uuid.UUID) error
	CountPurchasesByUserID(ctx context.Context, id uuid.UUID) (int64, error)
}

// StoreRepository define o contrato para estabelecimentos comerciais
type StoreRepository interface {
	FindOrCreateByName(ctx context.Context, name string, cnpj string) (*domain.Store, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Store, error)
	ListAll(ctx context.Context) ([]domain.Store, error)
}

// CanonicalProductRepository define o contrato para produtos com suporte a busca vetorial (pgvector)
type CanonicalProductRepository interface {
	Create(ctx context.Context, product *domain.CanonicalProduct) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CanonicalProduct, error)
	FindNearestByEmbedding(ctx context.Context, embedding pgvector.Vector, minSimilarity float32) (*domain.CanonicalProduct, float32, error)
	SearchByName(ctx context.Context, query string) ([]domain.CanonicalProduct, error)
	ListAll(ctx context.Context) ([]domain.CanonicalProduct, error)
}

// PurchaseRepository define o contrato de persistência de compras e itens
type PurchaseRepository interface {
	CreatePurchase(ctx context.Context, purchase *domain.Purchase) error
	UpdatePurchase(ctx context.Context, purchase *domain.Purchase) error
	GetPurchaseByID(ctx context.Context, id uuid.UUID) (*domain.Purchase, error)
	ListPurchases(ctx context.Context, limit int, offset int) ([]domain.Purchase, error)

	// Consultas analíticas
	GetPriceComparison(ctx context.Context, canonicalProductID uuid.UUID) (*domain.PriceComparisonView, error)
	GetMonthlyOscillation(ctx context.Context, canonicalProductID uuid.UUID) (*domain.MonthlyPriceOscillation, error)
	GetGlobalSummary(ctx context.Context, month time.Time) (*domain.GlobalSummaryView, error)
}

// ParsedReceiptItem representa os dados de item brutos extraídos pela LLM ou SEFAZ
type ParsedReceiptItem struct {
	RawDescription string  `json:"raw_description"`
	Quantity       float64 `json:"quantity"`
	PackageSize    float64 `json:"package_size"`
	PackageUnit    string  `json:"package_unit"`
	TotalPrice     float64 `json:"total_price"`
}

// ParsedReceipt representa o resultado da interpretação de uma nota fiscal
type ParsedReceipt struct {
	StoreName   string              `json:"store_name"`
	StoreCNPJ   string              `json:"store_cnpj"`
	PurchasedAt time.Time           `json:"purchased_at"`
	TotalAmount float64             `json:"total_amount"`
	FiscalKey   string              `json:"fiscal_key"`
	Items       []ParsedReceiptItem `json:"items"`
}

// ReceiptParser define a porta para interpretadores de notas fiscais (Gemini Multimodal, SEFAZ, etc)
type ReceiptParser interface {
	ParseReceipt(ctx context.Context, fileBytes []byte, mimeType string) (*ParsedReceipt, error)
}

// SefazAdapterPort define a porta para consultas de NFe/NFCe via SEFAZ
type SefazAdapterPort interface {
	ConsultarChave(ctx context.Context, chaveNFe string) (*ParsedReceipt, error)
}

// EmbeddingProvider define a porta para geração de vetores de embedding
type EmbeddingProvider interface {
	GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error)
}

// --- Casos de Uso (Primary / Inbound Ports) ---

// PurchaseService define os casos de uso para compras
type PurchaseService interface {
	ProcessReceiptFile(ctx context.Context, fileBytes []byte, mimeType string) (*domain.Purchase, error)
	ProcessFiscalKey(ctx context.Context, key string) (*domain.Purchase, error)
	ConfirmPurchase(ctx context.Context, purchase *domain.Purchase) (*domain.Purchase, error)
	CreateManualPurchase(ctx context.Context, purchase *domain.Purchase) (*domain.Purchase, error)
	GetPurchase(ctx context.Context, id uuid.UUID) (*domain.Purchase, error)
	ListPurchases(ctx context.Context, limit, offset int) ([]domain.Purchase, error)
}

// AnalyticsService define os casos de uso analíticos e inteligência de preços
type AnalyticsService interface {
	GetPriceComparison(ctx context.Context, canonicalProductID uuid.UUID) (*domain.PriceComparisonView, error)
	GetMonthlyOscillation(ctx context.Context, canonicalProductID uuid.UUID) (*domain.MonthlyPriceOscillation, error)
	GetGlobalSummary(ctx context.Context, month time.Time) (*domain.GlobalSummaryView, error)
	SearchProducts(ctx context.Context, query string) ([]domain.CanonicalProduct, error)
}

// UserService define os casos de uso para gerenciamento de moradores
type UserService interface {
	CreateUser(ctx context.Context, name, email string) (*domain.User, error)
	ListUsers(ctx context.Context) ([]domain.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) (*DeleteUserResult, error)
}
