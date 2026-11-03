package domain_test

import (
	"testing"

	"ComprasResidenciais/internal/core/domain"
)

func TestPurchaseItem_NormalizeUnits_Kilograms(t *testing.T) {
	// Exemplo: 1 pacote de 5kg por R$ 23,95 -> R$ 4,79 / kg
	item := domain.PurchaseItem{
		RawDescription: "ARROZ TIPO 1 TJ 5KG",
		Quantity:       1.0,
		PackageSize:    5.0,
		PackageUnit:    "kg",
		TotalPrice:     23.95,
	}

	err := item.NormalizeUnits()
	if err != nil {
		t.Fatalf("NormalizeUnits returned error: %v", err)
	}

	if item.TotalUnits != 5.0 {
		t.Errorf("expected TotalUnits 5.0, got %f", item.TotalUnits)
	}

	expectedNormalized := 23.95 / 5.0
	if item.NormalizedPrice != expectedNormalized {
		t.Errorf("expected NormalizedPrice %f, got %f", expectedNormalized, item.NormalizedPrice)
	}
}

func TestPurchaseItem_NormalizeUnits_GramsConversion(t *testing.T) {
	// Exemplo: 5 pacotes de 1000g (1kg) por R$ 29,00 -> R$ 5,80 / kg
	item := domain.PurchaseItem{
		RawDescription: "ARROZ PARBOILIZADO 1000G",
		Quantity:       5.0,
		PackageSize:    1000.0,
		PackageUnit:    "g",
		TotalPrice:     29.00,
	}

	err := item.NormalizeUnits()
	if err != nil {
		t.Fatalf("NormalizeUnits returned error: %v", err)
	}

	if item.PackageUnit != "kg" {
		t.Errorf("expected PackageUnit normalized to 'kg', got '%s'", item.PackageUnit)
	}

	if item.TotalUnits != 5.0 {
		t.Errorf("expected TotalUnits 5.0kg, got %f", item.TotalUnits)
	}

	if item.NormalizedPrice != 5.80 {
		t.Errorf("expected NormalizedPrice 5.80, got %f", item.NormalizedPrice)
	}
}

func TestPurchaseItem_NormalizeUnits_MillilitersConversion(t *testing.T) {
	// Exemplo: 12 caixas de Leite 1000ml por R$ 48,00 -> R$ 4,00 / L
	item := domain.PurchaseItem{
		RawDescription: "LEITE INTEGRAL 1000ML",
		Quantity:       12.0,
		PackageSize:    1000.0,
		PackageUnit:    "ml",
		TotalPrice:     48.00,
	}

	err := item.NormalizeUnits()
	if err != nil {
		t.Fatalf("NormalizeUnits returned error: %v", err)
	}

	if item.PackageUnit != "L" {
		t.Errorf("expected PackageUnit normalized to 'L', got '%s'", item.PackageUnit)
	}

	if item.TotalUnits != 12.0 {
		t.Errorf("expected TotalUnits 12.0L, got %f", item.TotalUnits)
	}

	if item.NormalizedPrice != 4.00 {
		t.Errorf("expected NormalizedPrice 4.00, got %f", item.NormalizedPrice)
	}
}
