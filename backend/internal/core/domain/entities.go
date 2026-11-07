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

// User representa um morador da residência responsável por compras.
// Campos alinhados para minimizar bytes de padding e maximizar localidade de cache.
type User struct {
	CreatedAt    time.Time `json:"created_at"` // 24 bytes
	UpdatedAt    time.Time `json:"updated_at"` // 24 bytes
	ID           uuid.UUID `json:"id"`         // 16 bytes
	Name         string    `json:"name"`       // 16 bytes
	Email        string    `json:"email,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	IsBlocked    bool      `json:"is_blocked"` // 1 byte + 7 bytes trailing padding
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
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CNPJ      string    `json:"cnpj,omitempty"`
	Address   string    `json:"address,omitempty"`
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
	PurchasedAt time.Time      `json:"purchased_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	StoreID     uuid.UUID      `json:"store_id"`
	TotalAmount float64        `json:"total_amount"`
	Status      PurchaseStatus `json:"status"`
	Origin      PurchaseOrigin `json:"origin"`
	FiscalKey   string         `json:"fiscal_key,omitempty"`
	Items       []PurchaseItem `json:"items,omitempty"`
}

// CanonicalProduct representa um produto abstrato unificado semanticamente
type CanonicalProduct struct {
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`      // ex: "Arroz Branco"
	Category  string          `json:"category"`  // ex: "Grãos"
	BaseUnit  string          `json:"base_unit"` // "kg", "L", "un"
	Embedding pgvector.Vector `json:"-"`         // vetor float32 para busca semântica
}

// PurchaseItem representa cada item registrado em uma nota de compra
type PurchaseItem struct {
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	ID                 uuid.UUID  `json:"id"`
	PurchaseID         uuid.UUID  `json:"purchase_id"`
	CanonicalProductID *uuid.UUID `json:"canonical_product_id,omitempty"`
	RawDescription     string     `json:"raw_description"`  // ex: "ARROZ TIPO 1 TJ 5KG"
	PackageUnit        string     `json:"package_unit"`     // ex: "kg", "g", "L", "ml", "un"
	Quantity           float64    `json:"quantity"`         // ex: 1.0 (pacote)
	PackageSize        float64    `json:"package_size"`     // ex: 5.0 (tamanho contido no pacote)
	TotalUnits         float64    `json:"total_units"`      // ex: 5.0 (em kg ou L)
	TotalPrice         float64    `json:"total_price"`      // ex: 23.90
	UnitPrice          float64    `json:"unit_price"`       // TotalPrice / Quantity (ex: 23.90/un)
	NormalizedPrice    float64    `json:"normalized_price"` // TotalPrice / TotalUnits (ex: 4.78/kg)
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
// 1. Deve conter exatamente 44 dígitos numéricos
// 2. Não pode ser sequência trivial repetida (ex: todos zeros)
// 3. Código da UF inicial deve corresponder a uma unidade federativa válida do IBGE
// 4. Mês de emissão deve estar entre 01 e 12
// 5. O dígito verificador final (posição 44) deve corresponder ao cálculo módulo 11
//
// OTIMIZAÇÃO: Utiliza buffer na stack [44]byte para validação direta, resultando em
// 0 allocs/op (zero heap allocation) mesmo sob alto volume de requisições.
func ValidateFiscalKey(key string) bool {
	var buf [44]byte
	count := 0

	for i := 0; i < len(key); i++ {
		c := key[i]
		if c >= '0' && c <= '9' {
			if count == 44 {
				return false // Ultrapassou 44 dígitos
			}
			buf[count] = c
			count++
		}
	}

	if count != 44 {
		return false
	}

	// Rejeita sequências repetidas triviais (ex: 44 zeros ou 44 uns)
	allSame := true
	for i := 1; i < 44; i++ {
		if buf[i] != buf[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return false
	}

	// Validação de código de UF (IBGE)
	uf := string(buf[:2])
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
	month := int(buf[4]-'0')*10 + int(buf[5]-'0')
	if month < 1 || month > 12 {
		return false
	}

	expectedDV := int(buf[43] - '0')

	sum := 0
	weight := 2
	for i := 42; i >= 0; i-- {
		digit := int(buf[i] - '0')
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
