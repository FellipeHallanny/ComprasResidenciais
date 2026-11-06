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

// PurchaseOrigin indica como a nota foi inserida
type PurchaseOrigin string

const (
	PurchaseOriginLLM    PurchaseOrigin = "LLM"
	PurchaseOriginSefaz  PurchaseOrigin = "SEFAZ"
	PurchaseOriginManual PurchaseOrigin = "MANUAL"
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
	TotalPrice         float64    `json:"total_price"`      // ex: 23.90
	UnitPrice          float64    `json:"unit_price"`       // TotalPrice / Quantity (ex: 23.90/un)
	NormalizedPrice    float64    `json:"normalized_price"` // TotalPrice / TotalUnits (ex: 4.78/kg)
	CanonicalProductID *uuid.UUID `json:"canonical_product_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

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

// ValidateFiscalKey valida se a chave de acesso da NF-e / NFC-e brasileira é válida:
// 1. Deve conter 44 dígitos numéricos
// 2. Não pode ser sequência trivial repetida (ex: todos zeros)
// 3. Código da UF inicial deve corresponder a uma unidade federativa válida do IBGE
// 4. Mês de emissão deve estar entre 01 e 12
// 5. O dígito verificador final (posição 44) deve corresponder ao cálculo módulo 11
func ValidateFiscalKey(key string) bool {
	var clean strings.Builder
	for _, r := range key {
		if r >= '0' && r <= '9' {
			clean.WriteRune(r)
		}
	}
	s := clean.String()
	if len(s) != 44 {
		return false
	}

	// Rejeita sequências repetidas triviais (ex: 44 zeros ou 44 uns)
	allSame := true
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return false
	}

	// Validação de código de UF (IBGE)
	uf := s[:2]
	switch uf {
	case "11", "12", "13", "14", "15", "16", "17",
		"21", "22", "23", "24", "25", "26", "27", "28", "29",
		"31", "32", "33", "35",
		"41", "42", "43",
		"50", "51", "52", "53":
		// UF válida
	default:
		return false
	}

	// Validação de mês de emissão (01 a 12)
	month := int(s[4]-'0')*10 + int(s[5]-'0')
	if month < 1 || month > 12 {
		return false
	}

	base := s[:43]
	expectedDV := int(s[43] - '0')

	sum := 0
	weight := 2
	for i := len(base) - 1; i >= 0; i-- {
		digit := int(base[i] - '0')
		sum += digit * weight
		weight++
		if weight > 9 {
			weight = 2
		}
	}

	mod := sum % 11
	calculatedDV := 0
	if mod > 1 {
		calculatedDV = 11 - mod
	}

	return calculatedDV == expectedDV
}
