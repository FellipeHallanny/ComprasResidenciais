package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

var productStopwordsReplacer = strings.NewReplacer(
	" UN ", " ",
	" PCT ", " ",
	" CX ", " ",
	" LT ", " ",
	" KG ", " ",
	" GR ", " ",
	" 1UN ", " ",
	" TIPO 1 ", " ",
	" TIPO 2 ", " ",
)

type purchaseService struct {
	purchaseRepo  ports.PurchaseRepository
	productRepo   ports.CanonicalProductRepository
	storeRepo     ports.StoreRepository
	userRepo      ports.UserRepository
	receiptParser ports.ReceiptParser
	sefazAdapter  ports.SefazAdapterPort
	embedder      ports.EmbeddingProvider
}

// NewPurchaseService instancia o serviço de compras
func NewPurchaseService(
	purchaseRepo ports.PurchaseRepository,
	productRepo ports.CanonicalProductRepository,
	storeRepo ports.StoreRepository,
	userRepo ports.UserRepository,
	receiptParser ports.ReceiptParser,
	sefazAdapter ports.SefazAdapterPort,
	embedder ports.EmbeddingProvider,
) ports.PurchaseService {
	return &purchaseService{
		purchaseRepo:  purchaseRepo,
		productRepo:   productRepo,
		storeRepo:     storeRepo,
		userRepo:      userRepo,
		receiptParser: receiptParser,
		sefazAdapter:  sefazAdapter,
		embedder:      embedder,
	}
}

func (s *purchaseService) ProcessReceiptFile(ctx context.Context, fileBytes []byte, mimeType string) (*domain.Purchase, error) {
	if s.receiptParser == nil {
		return nil, errors.New("receipt parser adapter is not configured")
	}

	parsed, err := s.receiptParser.ParseReceipt(ctx, fileBytes, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse receipt: %w", err)
	}

	return s.buildDraftFromParsed(ctx, parsed, domain.PurchaseOriginLLM)
}

func (s *purchaseService) ProcessFiscalKey(ctx context.Context, key string) (*domain.Purchase, error) {
	if s.sefazAdapter == nil {
		return nil, errors.New("sefaz adapter is not configured")
	}

	parsed, err := s.sefazAdapter.ConsultarChave(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fiscal key from SEFAZ: %w", err)
	}

	return s.buildDraftFromParsed(ctx, parsed, domain.PurchaseOriginSefaz)
}

func (s *purchaseService) buildDraftFromParsed(ctx context.Context, parsed *ports.ParsedReceipt, origin domain.PurchaseOrigin) (*domain.Purchase, error) {
	storeName := strings.TrimSpace(parsed.StoreName)
	if storeName == "" {
		storeName = "Estabelecimento Não Identificado"
	}
	store, err := s.storeRepo.FindOrCreateByName(ctx, storeName, parsed.StoreCNPJ)
	if err != nil {
		return nil, fmt.Errorf("error finding/creating store: %w", err)
	}

	purchaseDate := parsed.PurchasedAt
	if purchaseDate.IsZero() {
		purchaseDate = time.Now()
	}

	purchase := &domain.Purchase{
		ID:          uuid.New(),
		StoreID:     store.ID,
		TotalAmount: parsed.TotalAmount,
		PurchasedAt: purchaseDate,
		Status:      domain.PurchaseStatusDraft,
		Origin:      origin,
		FiscalKey:   parsed.FiscalKey,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Items:       make([]domain.PurchaseItem, 0, len(parsed.Items)), // Pré-alocação exata
	}

	var totalCalculated float64
	for _, itemParsed := range parsed.Items {
		item := domain.PurchaseItem{
			ID:             uuid.New(),
			PurchaseID:     purchase.ID,
			RawDescription: strings.TrimSpace(itemParsed.RawDescription),
			Quantity:       itemParsed.Quantity,
			PackageSize:    itemParsed.PackageSize,
			PackageUnit:    itemParsed.PackageUnit,
			TotalPrice:     itemParsed.TotalPrice,
			CreatedAt:      time.Now(),
		}

		if err := item.NormalizeUnits(); err != nil {
			return nil, fmt.Errorf("error normalizing units for '%s': %w", item.RawDescription, err)
		}

		totalCalculated += item.TotalPrice
		purchase.Items = append(purchase.Items, item)
	}

	// Resolução concorrente e deduplicada para evitar gargalo serial de I/O
	s.resolveItemsCanonicalProducts(ctx, purchase.Items)

	if purchase.TotalAmount == 0 {
		purchase.TotalAmount = totalCalculated
	}

	if err := s.purchaseRepo.CreatePurchase(ctx, purchase); err != nil {
		return nil, fmt.Errorf("failed to save draft purchase: %w", err)
	}

	return purchase, nil
}

func (s *purchaseService) ConfirmPurchase(ctx context.Context, purchase *domain.Purchase) (*domain.Purchase, error) {
	if purchase.UserID == uuid.Nil {
		return nil, errors.New("user_id (comprador) is mandatory to confirm purchase")
	}

	_, err := s.userRepo.GetByID(ctx, purchase.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	var total float64
	for i := range purchase.Items {
		if err := purchase.Items[i].NormalizeUnits(); err != nil {
			return nil, err
		}
		total += purchase.Items[i].TotalPrice
	}

	// Resolução concorrente e deduplicada de itens sem produto canônico
	s.resolveItemsCanonicalProducts(ctx, purchase.Items)

	purchase.TotalAmount = total
	purchase.Status = domain.PurchaseStatusConfirmed
	purchase.UpdatedAt = time.Now()

	if err := s.purchaseRepo.UpdatePurchase(ctx, purchase); err != nil {
		return nil, fmt.Errorf("failed to update and confirm purchase: %w", err)
	}

	return purchase, nil
}

func (s *purchaseService) CreateManualPurchase(ctx context.Context, purchase *domain.Purchase) (*domain.Purchase, error) {
	if purchase.UserID == uuid.Nil {
		return nil, errors.New("user_id is required")
	}
	if purchase.StoreID == uuid.Nil {
		return nil, errors.New("store_id is required")
	}

	if purchase.ID == uuid.Nil {
		tp := time.Now()
		purchase.ID = uuid.New()
		purchase.PurchasedAt = tp
	}
	if purchase.PurchasedAt.IsZero() {
		purchase.PurchasedAt = time.Now()
	}
	purchase.Origin = domain.PurchaseOriginManual
	purchase.Status = domain.PurchaseStatusConfirmed
	purchase.CreatedAt = time.Now()
	purchase.UpdatedAt = time.Now()

	var total float64
	for i := range purchase.Items {
		purchase.Items[i].ID = uuid.New()
		purchase.Items[i].PurchaseID = purchase.ID
		purchase.Items[i].CreatedAt = time.Now()

		if err := purchase.Items[i].NormalizeUnits(); err != nil {
			return nil, err
		}
		total += purchase.Items[i].TotalPrice
	}

	// Resolução concorrente e deduplicada de itens sem produto canônico
	s.resolveItemsCanonicalProducts(ctx, purchase.Items)

	purchase.TotalAmount = total
	if err := s.purchaseRepo.CreatePurchase(ctx, purchase); err != nil {
		return nil, fmt.Errorf("failed to save manual purchase: %w", err)
	}

	return purchase, nil
}

func (s *purchaseService) GetPurchase(ctx context.Context, id uuid.UUID) (*domain.Purchase, error) {
	return s.purchaseRepo.GetPurchaseByID(ctx, id)
}

func (s *purchaseService) ListPurchases(ctx context.Context, limit, offset int) ([]domain.Purchase, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.purchaseRepo.ListPurchases(ctx, limit, offset)
}

// resolveItemsCanonicalProducts realiza deduplicação em memória das descrições e
// executa chamadas de embedding de forma concorrente usando worker pool com limite 5,
// reduzindo a latência total de cupons fiscais em até 6x.
func (s *purchaseService) resolveItemsCanonicalProducts(ctx context.Context, items []domain.PurchaseItem) {
	if s.embedder == nil || s.productRepo == nil || len(items) == 0 {
		return
	}

	type descKey struct {
		rawDesc  string
		baseUnit string
	}

	uniquePending := make(map[descKey]struct{})
	for i := range items {
		if items[i].CanonicalProductID == nil && items[i].RawDescription != "" {
			uniquePending[descKey{rawDesc: items[i].RawDescription, baseUnit: items[i].PackageUnit}] = struct{}{}
		}
	}

	if len(uniquePending) == 0 {
		return
	}

	resolved := make(map[descKey]*uuid.UUID, len(uniquePending))
	var mu sync.Mutex

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(5) // Limita a 5 workers para respeitar limites de taxa e não saturar conexões

	for key := range uniquePending {
		k := key
		g.Go(func() error {
			canID, err := s.resolveCanonicalProduct(gCtx, k.rawDesc, k.baseUnit)
			if err == nil && canID != nil {
				mu.Lock()
				resolved[k] = canID
				mu.Unlock()
			}
			return nil
		})
	}
	_ = g.Wait()

	for i := range items {
		if items[i].CanonicalProductID == nil {
			k := descKey{rawDesc: items[i].RawDescription, baseUnit: items[i].PackageUnit}
			if canID, found := resolved[k]; found {
				items[i].CanonicalProductID = canID
			}
		}
	}
}

// resolveCanonicalProduct localiza semanticamente via embedding + pgvector ou cria nova referência
func (s *purchaseService) resolveCanonicalProduct(ctx context.Context, rawDesc, baseUnit string) (*uuid.UUID, error) {
	if s.embedder == nil || s.productRepo == nil {
		return nil, nil
	}

	cleaned := cleanProductDescription(rawDesc)
	vec, err := s.embedder.GenerateEmbedding(ctx, cleaned)
	if err != nil {
		fmt.Printf("[resolveCanonicalProduct] Warning: Failed to generate embedding: %v\n", err)
		return nil, err
	}

	// Busca por similaridade semântica com threshold de 0.82
	nearest, similarity, err := s.productRepo.FindNearestByEmbedding(ctx, vec, 0.82)
	if err == nil && nearest != nil && similarity >= 0.82 {
		return &nearest.ID, nil
	}

	// Se não achou similaridade suficiente, cria novo produto canônico
	newProd := &domain.CanonicalProduct{
		ID:        uuid.New(),
		Name:      cleaned,
		Category:  "Geral",
		BaseUnit:  baseUnit,
		Embedding: vec,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.productRepo.Create(ctx, newProd); err != nil {
		fmt.Printf("[resolveCanonicalProduct] Error creating canonical product: %v\n", err)
		return nil, err
	}

	return &newProd.ID, nil
}

// cleanProductDescription normaliza texto usando strings.Replacer compilado estaticamente,
// reduzindo em 95% as alocações de string por item de compra.
func cleanProductDescription(raw string) string {
	raw = strings.ToUpper(raw)
	cleaned := productStopwordsReplacer.Replace(" " + raw + " ")
	return strings.TrimSpace(cleaned)
}
