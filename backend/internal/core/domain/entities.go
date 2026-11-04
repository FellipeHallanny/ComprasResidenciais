package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

const (
	UserRoleAdmin  = "ADMIN"
	UserRoleMember = "MEMBER"
)

// User representa um morador da residência responsável por compras
type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	IsBlocked    bool      `json:"is_blocked"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LoginRequest representa a requisição de login por e-mail e senha
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse retorna o token JWT e os dados do usuário autenticado
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// SetupAdminRequest representa o payload para registrar o primeiro administrador do sistema
type SetupAdminRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Store representa o estabelecimento comercial onde a compra ocorreu
type Store struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CNPJ      string    `json:"cnpj,omitempty"`
	Address   string    `json:"address,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PurchaseStatus indica se a compra é rascunho ou confirmada
type PurchaseStatus string

const (
	PurchaseStatusDraft     PurchaseStatus = "DRAFT"
	PurchaseStatusConfirmed PurchaseStatus = "CONFIRMED"
)

// PurchaseOrigin indica como os dados da compra foram originados
type PurchaseOrigin string

const (
	PurchaseOriginManual PurchaseOrigin = "MANUAL"
	PurchaseOriginLLM    PurchaseOrigin = "LLM_UPLOAD"
	PurchaseOriginSefaz  PurchaseOrigin = "SEFAZ"
)

// Purchase representa a sessão ou nota fiscal de compra agrupada
type Purchase struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	StoreID     uuid.UUID      `json:"store_id"`
	TotalAmount float64        `json:"total_amount"`
	PurchasedAt time.Time      `json:"purchased_at"`
	Status      PurchaseStatus `json:"status"`
	Origin      PurchaseOrigin `json:"origin"`
	FiscalKey   string         `json:"fiscal_key,omitempty"`
	Items       []PurchaseItem `json:"items,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// CanonicalProduct representa um produto abstrato unificado semanticamente
type CanonicalProduct struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`      // ex: "Arroz Branco"
	Category  string          `json:"category"`  // ex: "Grãos"
	BaseUnit  string          `json:"base_unit"` // "kg", "L", "un"
	Embedding pgvector.Vector `json:"-"`         // vetor float32 para busca semântica
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// PurchaseItem representa cada item registrado em uma nota de compra
type PurchaseItem struct {
	ID                 uuid.UUID  `json:"id"`
	PurchaseID         uuid.UUID  `json:"purchase_id"`
	RawDescription     string     `json:"raw_description"`  // ex: "ARROZ TIPO 1 TJ 5KG"
	Quantity           float64    `json:"quantity"`         // ex: 1.0 (pacote)
	PackageSize        float64    `json:"package_size"`     // ex: 5.0 (tamanho contido no pacote)
	PackageUnit        string     `json:"package_unit"`     // ex: "kg", "g", "L", "ml", "un"
	TotalUnits         float64    `json:"total_units"`      // ex: 5.0 (em kg ou L)
	TotalPrice         float64    `json:"total_price"`      // ex: 23.95
	UnitPrice          float64    `json:"unit_price"`       // ex: 23.95 (preço unitário do pacote)
	NormalizedPrice    float64    `json:"normalized_price"` // ex: 4.79 (R$ por unidade base: R$/kg)
	CanonicalProductID *uuid.UUID `json:"canonical_product_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// NormalizeUnits normaliza a unidade e calcula o TotalUnits e NormalizedPrice
// Unidades aceitas: peso (kg, g), volume (l, ml), contagem (un)
func (item *PurchaseItem) NormalizeUnits() error {
	if item.TotalPrice < 0 {
		return errors.New("total_price cannot be negative")
	}
	if item.Quantity <= 0 {
		item.Quantity = 1.0
	}
	if item.PackageSize <= 0 {
		item.PackageSize = 1.0
	}

	unit := strings.ToLower(strings.TrimSpace(item.PackageUnit))
	var baseMultiplier float64

	switch unit {
	case "kg", "quilo", "kilo":
		baseMultiplier = 1.0
		item.PackageUnit = "kg"
	case "g", "grama", "gr":
		baseMultiplier = 0.001
		item.PackageUnit = "kg"
	case "l", "litro", "lt":
		baseMultiplier = 1.0
		item.PackageUnit = "L"
	case "ml", "mililitro":
		baseMultiplier = 0.001
		item.PackageUnit = "L"
	case "un", "unidade", "und", "pct", "pacote", "cx", "caixa", "dz", "duzia":
		baseMultiplier = 1.0
		item.PackageUnit = "un"
	default:
		baseMultiplier = 1.0
		item.PackageUnit = "un"
	}

	item.TotalUnits = item.Quantity * item.PackageSize * baseMultiplier

	if item.Quantity > 0 {
		item.UnitPrice = item.TotalPrice / item.Quantity
	} else {
		item.UnitPrice = item.TotalPrice
	}

	if item.TotalUnits > 0 {
		item.NormalizedPrice = item.TotalPrice / item.TotalUnits
	} else {
		item.NormalizedPrice = item.TotalPrice
	}

	return nil
}
