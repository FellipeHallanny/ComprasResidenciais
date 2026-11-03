package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// User representa um morador da residência responsável por compras
type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	IsBlocked bool      `json:"is_blocked"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
		return errors.New("total price cannot be negative")
	}

	unit := strings.ToLower(strings.TrimSpace(item.PackageUnit))
	qty := item.Quantity
	if qty <= 0 {
		qty = 1.0
		item.Quantity = 1.0
	}
	size := item.PackageSize
	if size <= 0 {
		size = 1.0
		item.PackageSize = 1.0
	}

	var baseTotal float64
	switch unit {
	case "kg", "quilo", "kilos":
		baseTotal = qty * size
		item.PackageUnit = "kg"
	case "g", "gramas", "gr":
		baseTotal = (qty * size) / 1000.0
		item.PackageUnit = "kg"
	case "l", "litro", "litros":
		baseTotal = qty * size
		item.PackageUnit = "L"
	case "ml", "mililitros":
		baseTotal = (qty * size) / 1000.0
		item.PackageUnit = "L"
	case "un", "unidade", "unidades", "pct", "cx":
		baseTotal = qty * size
		item.PackageUnit = "un"
	default:
		// Default para unidade genérica
		baseTotal = qty * size
		item.PackageUnit = "un"
	}

	item.TotalUnits = baseTotal
	if baseTotal > 0 {
		item.NormalizedPrice = item.TotalPrice / baseTotal
	} else {
		item.NormalizedPrice = item.TotalPrice
	}

	if qty > 0 {
		item.UnitPrice = item.TotalPrice / qty
	}

	return nil
}
