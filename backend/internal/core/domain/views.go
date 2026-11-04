package domain

import "time"

// PriceComparisonView consolida os dados de menor preço e histórico de um produto
type PriceComparisonView struct {
	CanonicalProductID   string               `json:"canonical_product_id"`
	ProductName          string               `json:"product_name"`
	BaseUnit             string               `json:"base_unit"`
	LowestPricePerUnit   float64              `json:"lowest_price_per_unit"`
	HighestPricePerUnit  float64              `json:"highest_price_per_unit"`
	AveragePricePerUnit  float64              `json:"average_price_per_unit"`
	CheapestStoreName    string               `json:"cheapest_store_name"`
	CheapestPurchaseDate time.Time            `json:"cheapest_purchase_date"`
	MaxSavingsPercentage float64              `json:"max_savings_percentage"`
	BestMonthPeriod      string               `json:"best_month_period"` // ex: "Dias 15-21"
	RecentPurchases      []RecentItemPurchase `json:"recent_purchases"`
}

// RecentItemPurchase detalha uma compra recente do produto
type RecentItemPurchase struct {
	PurchaseID      string    `json:"purchase_id"`
	PurchasedAt     time.Time `json:"purchased_at"`
	StoreName       string    `json:"store_name"`
	BuyerName       string    `json:"buyer_name"`
	RawDescription  string    `json:"raw_description"`
	PackageInfo     string    `json:"package_info"` // ex: "1x 5kg"
	NormalizedPrice float64   `json:"normalized_price"`
	TotalPrice      float64   `json:"total_price"`
}

// MonthlyPriceOscillation representa a média de preço em cada período do mês
type MonthlyPriceOscillation struct {
	CanonicalProductID string         `json:"canonical_product_id"`
	ProductName        string         `json:"product_name"`
	BaseUnit           string         `json:"base_unit"`
	Periods            []PeriodMetric `json:"periods"`
}

// PeriodMetric consolida métricas para um período (ex: dias 01-07, 08-14, 15-21, 22-31)
type PeriodMetric struct {
	PeriodLabel  string  `json:"period_label"`  // "Dias 01-07", "Dias 08-14", etc.
	AveragePrice float64 `json:"average_price"` // R$ por unidade base
	TotalCount   int     `json:"total_count"`
}

// GlobalSummaryView consolida KPIs gerais da residência
type GlobalSummaryView struct {
	CurrentMonthTotal float64            `json:"current_month_total"`
	TotalPurchases    int                `json:"total_purchases"`
	SpendingByUser    map[string]float64 `json:"spending_by_user"`
	TopStores         []StoreShareMetric `json:"top_stores"`
}

// StoreShareMetric participação do estabelecimento
type StoreShareMetric struct {
	StoreName  string  `json:"store_name"`
	TotalSpent float64 `json:"total_spent"`
	Percentage float64 `json:"percentage"`
}

// CatalogItemView consolida dados de um produto canônico com o seu menor preço registrado e estabelecimento mais barato
type CatalogItemView struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Category             string     `json:"category"`
	BaseUnit             string     `json:"base_unit"`
	LowestPricePerUnit   float64    `json:"lowest_price_per_unit"`
	CheapestStoreName    string     `json:"cheapest_store_name"`
	CheapestStoreID      *string    `json:"cheapest_store_id,omitempty"`
	CheapestStoreAddress string     `json:"cheapest_store_address,omitempty"`
	CheapestPackageInfo  string     `json:"cheapest_package_info,omitempty"`
	CheapestPurchaseDate *time.Time `json:"cheapest_purchase_date,omitempty"`
	LastPurchasedAt      *time.Time `json:"last_purchased_at,omitempty"`
	CreatedAt            *time.Time `json:"created_at,omitempty"`
	TotalPurchases       int        `json:"total_purchases"`
}
