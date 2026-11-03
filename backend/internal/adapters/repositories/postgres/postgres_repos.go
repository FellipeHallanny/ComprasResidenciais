package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"ComprasResidenciais/internal/core/domain"
	"ComprasResidenciais/internal/core/ports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

type PostgresRepositories struct {
	pool *pgxpool.Pool
}

func NewPostgresRepositories(pool *pgxpool.Pool) *PostgresRepositories {
	return &PostgresRepositories{pool: pool}
}

// RegisterPgvectorType registra os codecs necessários para manipular pgvector no pgxpool
func RegisterPgvectorType(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	return pgxvec.RegisterTypes(ctx, conn.Conn())
}

// RunAutoMigrations aplica migrações essenciais caso as colunas ou tabelas não existam
func RunAutoMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrations := []string{
		`ALTER TABLE household_users ADD COLUMN IF NOT EXISTS is_blocked BOOLEAN NOT NULL DEFAULT FALSE;`,
	}

	for _, m := range migrations {
		if _, err := pool.Exec(ctx, m); err != nil {
			log.Printf("Warning: error running auto migration: %v", err)
			return err
		}
	}
	return nil
}

// --- UserRepository Implementation ---

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) ports.UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, u *domain.User) error {
	query := `INSERT INTO household_users (id, name, email, is_blocked, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(ctx, query, u.ID, u.Name, u.Email, u.IsBlocked, u.CreatedAt, u.UpdatedAt)
	return err
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT id, name, COALESCE(email, ''), is_blocked, created_at, updated_at FROM household_users WHERE id = $1`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(&u.ID, &u.Name, &u.Email, &u.IsBlocked, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) ListAll(ctx context.Context) ([]domain.User, error) {
	query := `SELECT id, name, COALESCE(email, ''), is_blocked, created_at, updated_at FROM household_users ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.IsBlocked, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *userRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM household_users WHERE id = $1`
	res, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *userRepo) Block(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE household_users SET is_blocked = TRUE, updated_at = $2 WHERE id = $1`
	res, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *userRepo) CountPurchasesByUserID(ctx context.Context, id uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM purchases WHERE user_id = $1`
	var count int64
	err := r.pool.QueryRow(ctx, query, id).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// --- StoreRepository Implementation ---

type storeRepo struct {
	pool *pgxpool.Pool
}

func NewStoreRepository(pool *pgxpool.Pool) ports.StoreRepository {
	return &storeRepo{pool: pool}
}

func (r *storeRepo) FindOrCreateByName(ctx context.Context, name string, cnpj string) (*domain.Store, error) {
	query := `SELECT id, name, COALESCE(cnpj, ''), COALESCE(address, ''), created_at, updated_at FROM stores WHERE LOWER(name) = LOWER($1) LIMIT 1`
	var s domain.Store
	err := r.pool.QueryRow(ctx, query, name).Scan(&s.ID, &s.Name, &s.CNPJ, &s.Address, &s.CreatedAt, &s.UpdatedAt)
	if err == nil {
		return &s, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Criar novo
	s = domain.Store{
		ID:        uuid.New(),
		Name:      name,
		CNPJ:      cnpj,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertQuery := `INSERT INTO stores (id, name, cnpj, address, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = r.pool.Exec(ctx, insertQuery, s.ID, s.Name, s.CNPJ, s.Address, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *storeRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Store, error) {
	query := `SELECT id, name, COALESCE(cnpj, ''), COALESCE(address, ''), created_at, updated_at FROM stores WHERE id = $1`
	var s domain.Store
	err := r.pool.QueryRow(ctx, query, id).Scan(&s.ID, &s.Name, &s.CNPJ, &s.Address, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *storeRepo) ListAll(ctx context.Context) ([]domain.Store, error) {
	query := `SELECT id, name, COALESCE(cnpj, ''), COALESCE(address, ''), created_at, updated_at FROM stores ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stores []domain.Store
	for rows.Next() {
		var s domain.Store
		if err := rows.Scan(&s.ID, &s.Name, &s.CNPJ, &s.Address, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		stores = append(stores, s)
	}
	return stores, nil
}

// --- CanonicalProductRepository Implementation ---

type productRepo struct {
	pool *pgxpool.Pool
}

func NewCanonicalProductRepository(pool *pgxpool.Pool) ports.CanonicalProductRepository {
	return &productRepo{pool: pool}
}

func (r *productRepo) Create(ctx context.Context, p *domain.CanonicalProduct) error {
	query := `INSERT INTO canonical_products (id, name, category, base_unit, embedding, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, query, p.ID, p.Name, p.Category, p.BaseUnit, p.Embedding, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *productRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.CanonicalProduct, error) {
	query := `SELECT id, name, COALESCE(category, 'Geral'), base_unit, embedding, created_at, updated_at FROM canonical_products WHERE id = $1`
	var p domain.CanonicalProduct
	err := r.pool.QueryRow(ctx, query, id).Scan(&p.ID, &p.Name, &p.Category, &p.BaseUnit, &p.Embedding, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *productRepo) FindNearestByEmbedding(ctx context.Context, vec pgvector.Vector, minSimilarity float32) (*domain.CanonicalProduct, float32, error) {
	// Usando operador de distância de cosseno <=> do pgvector: 1 - cosine_distance = similarity
	query := `
		SELECT id, name, COALESCE(category, 'Geral'), base_unit, embedding, created_at, updated_at,
		       1 - (embedding <=> $1) AS similarity
		FROM canonical_products
		WHERE 1 - (embedding <=> $1) >= $2
		ORDER BY embedding <=> $1 ASC
		LIMIT 1
	`
	var p domain.CanonicalProduct
	var similarity float32
	err := r.pool.QueryRow(ctx, query, vec, minSimilarity).Scan(
		&p.ID, &p.Name, &p.Category, &p.BaseUnit, &p.Embedding, &p.CreatedAt, &p.UpdatedAt, &similarity,
	)
	if err != nil {
		return nil, 0, err
	}
	return &p, similarity, nil
}

func (r *productRepo) SearchByName(ctx context.Context, queryStr string) ([]domain.CanonicalProduct, error) {
	query := `
		SELECT id, name, COALESCE(category, 'Geral'), base_unit, created_at, updated_at 
		FROM canonical_products 
		WHERE name ILIKE '%' || $1 || '%' 
		ORDER BY name ASC 
		LIMIT 20
	`
	rows, err := r.pool.Query(ctx, query, queryStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prods []domain.CanonicalProduct
	for rows.Next() {
		var p domain.CanonicalProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.BaseUnit, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prods = append(prods, p)
	}
	return prods, nil
}

func (r *productRepo) ListAll(ctx context.Context) ([]domain.CanonicalProduct, error) {
	query := `SELECT id, name, COALESCE(category, 'Geral'), base_unit, created_at, updated_at FROM canonical_products ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prods []domain.CanonicalProduct
	for rows.Next() {
		var p domain.CanonicalProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.BaseUnit, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prods = append(prods, p)
	}
	return prods, nil
}

// --- PurchaseRepository Implementation ---

type purchaseRepo struct {
	pool *pgxpool.Pool
}

func NewPurchaseRepository(pool *pgxpool.Pool) ports.PurchaseRepository {
	return &purchaseRepo{pool: pool}
}

func (r *purchaseRepo) CreatePurchase(ctx context.Context, p *domain.Purchase) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	purchaseQuery := `
		INSERT INTO purchases (id, user_id, store_id, total_amount, purchased_at, status, origin, fiscal_key, created_at, updated_at)
		VALUES ($1, NULLIF($2, '00000000-0000-0000-0000-000000000000'::uuid), $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = tx.Exec(ctx, purchaseQuery,
		p.ID, p.UserID, p.StoreID, p.TotalAmount, p.PurchasedAt, p.Status, p.Origin, p.FiscalKey, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert purchase: %w", err)
	}

	itemQuery := `
		INSERT INTO purchase_items (
			id, purchase_id, raw_description, quantity, package_size, package_unit,
			total_units, total_price, unit_price, normalized_price, canonical_product_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	for _, item := range p.Items {
		_, err = tx.Exec(ctx, itemQuery,
			item.ID, p.ID, item.RawDescription, item.Quantity, item.PackageSize, item.PackageUnit,
			item.TotalUnits, item.TotalPrice, item.UnitPrice, item.NormalizedPrice, item.CanonicalProductID, item.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert purchase item '%s': %w", item.RawDescription, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *purchaseRepo) UpdatePurchase(ctx context.Context, p *domain.Purchase) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	updateQuery := `
		UPDATE purchases 
		SET user_id = $2, store_id = $3, total_amount = $4, purchased_at = $5, status = $6, updated_at = $7
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, updateQuery, p.ID, p.UserID, p.StoreID, p.TotalAmount, p.PurchasedAt, p.Status, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update purchase: %w", err)
	}

	// Deleta itens existentes e re-insere atualizados
	_, err = tx.Exec(ctx, `DELETE FROM purchase_items WHERE purchase_id = $1`, p.ID)
	if err != nil {
		return fmt.Errorf("failed to clean existing items: %w", err)
	}

	itemQuery := `
		INSERT INTO purchase_items (
			id, purchase_id, raw_description, quantity, package_size, package_unit,
			total_units, total_price, unit_price, normalized_price, canonical_product_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	for _, item := range p.Items {
		itemUUID := item.ID
		if itemUUID == uuid.Nil {
			itemUUID = uuid.New()
		}
		itemCreatedAt := item.CreatedAt
		if itemCreatedAt.IsZero() {
			itemCreatedAt = time.Now()
		}
		_, err = tx.Exec(ctx, itemQuery,
			itemUUID, p.ID, item.RawDescription, item.Quantity, item.PackageSize, item.PackageUnit,
			item.TotalUnits, item.TotalPrice, item.UnitPrice, item.NormalizedPrice, item.CanonicalProductID, itemCreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to re-insert purchase item '%s': %w", item.RawDescription, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *purchaseRepo) GetPurchaseByID(ctx context.Context, id uuid.UUID) (*domain.Purchase, error) {
	query := `
		SELECT id, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid), store_id, total_amount, purchased_at, status, origin, COALESCE(fiscal_key, ''), created_at, updated_at
		FROM purchases WHERE id = $1
	`
	var p domain.Purchase
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.UserID, &p.StoreID, &p.TotalAmount, &p.PurchasedAt, &p.Status, &p.Origin, &p.FiscalKey, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	itemQuery := `
		SELECT id, purchase_id, raw_description, quantity, package_size, package_unit,
		       total_units, total_price, unit_price, normalized_price, canonical_product_id, created_at
		FROM purchase_items WHERE purchase_id = $1 ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, itemQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.PurchaseItem
		if err := rows.Scan(
			&item.ID, &item.PurchaseID, &item.RawDescription, &item.Quantity, &item.PackageSize, &item.PackageUnit,
			&item.TotalUnits, &item.TotalPrice, &item.UnitPrice, &item.NormalizedPrice, &item.CanonicalProductID, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		p.Items = append(p.Items, item)
	}

	return &p, nil
}

func (r *purchaseRepo) ListPurchases(ctx context.Context, limit int, offset int) ([]domain.Purchase, error) {
	query := `
		SELECT id, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid), store_id, total_amount, purchased_at, status, origin, COALESCE(fiscal_key, ''), created_at, updated_at
		FROM purchases ORDER BY purchased_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var purchases []domain.Purchase
	for rows.Next() {
		var p domain.Purchase
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.StoreID, &p.TotalAmount, &p.PurchasedAt, &p.Status, &p.Origin, &p.FiscalKey, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		purchases = append(purchases, p)
	}
	return purchases, nil
}

// GetPriceComparison calcula o menor preço normalizado, oscilação e histórico do produto canônico
func (r *purchaseRepo) GetPriceComparison(ctx context.Context, canonicalProductID uuid.UUID) (*domain.PriceComparisonView, error) {
	var prodName, baseUnit string
	err := r.pool.QueryRow(ctx, `SELECT name, base_unit FROM canonical_products WHERE id = $1`, canonicalProductID).Scan(&prodName, &baseUnit)
	if err != nil {
		return nil, fmt.Errorf("canonical product not found: %w", err)
	}

	view := &domain.PriceComparisonView{
		CanonicalProductID: canonicalProductID.String(),
		ProductName:        prodName,
		BaseUnit:           baseUnit,
		RecentPurchases:    []domain.RecentItemPurchase{},
	}

	summaryQuery := `
		SELECT MIN(pi.normalized_price), MAX(pi.normalized_price), AVG(pi.normalized_price)
		FROM purchase_items pi
		JOIN purchases p ON p.id = pi.purchase_id
		WHERE pi.canonical_product_id = $1 AND p.status = 'CONFIRMED'
	`
	var minP, maxP, avgP *float64
	err = r.pool.QueryRow(ctx, summaryQuery, canonicalProductID).Scan(&minP, &maxP, &avgP)
	if err == nil && minP != nil && maxP != nil && avgP != nil {
		view.LowestPricePerUnit = *minP
		view.HighestPricePerUnit = *maxP
		view.AveragePricePerUnit = *avgP
		if *maxP > 0 {
			view.MaxSavingsPercentage = ((*maxP - *minP) / *maxP) * 100
		}
	}

	cheapestQuery := `
		SELECT s.name, p.purchased_at
		FROM purchase_items pi
		JOIN purchases p ON p.id = pi.purchase_id
		JOIN stores s ON s.id = p.store_id
		WHERE pi.canonical_product_id = $1 AND p.status = 'CONFIRMED'
		ORDER BY pi.normalized_price ASC, p.purchased_at DESC
		LIMIT 1
	`
	_ = r.pool.QueryRow(ctx, cheapestQuery, canonicalProductID).Scan(&view.CheapestStoreName, &view.CheapestPurchaseDate)

	historyQuery := `
		SELECT 
			p.id, p.purchased_at, s.name, COALESCE(u.name, 'Não Definido'),
			pi.raw_description, CONCAT(pi.quantity, 'x ', pi.package_size, pi.package_unit),
			pi.normalized_price, pi.total_price
		FROM purchase_items pi
		JOIN purchases p ON p.id = pi.purchase_id
		JOIN stores s ON s.id = p.store_id
		LEFT JOIN household_users u ON u.id = p.user_id
		WHERE pi.canonical_product_id = $1 AND p.status = 'CONFIRMED'
		ORDER BY p.purchased_at DESC
		LIMIT 10
	`
	rows, err := r.pool.Query(ctx, historyQuery, canonicalProductID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r domain.RecentItemPurchase
			_ = rows.Scan(&r.PurchaseID, &r.PurchasedAt, &r.StoreName, &r.BuyerName, &r.RawDescription, &r.PackageInfo, &r.NormalizedPrice, &r.TotalPrice)
			view.RecentPurchases = append(view.RecentPurchases, r)
		}
	}

	return view, nil
}

// GetMonthlyOscillation agrupa os preços do produto por quinzenas/períodos do mês
func (r *purchaseRepo) GetMonthlyOscillation(ctx context.Context, canonicalProductID uuid.UUID) (*domain.MonthlyPriceOscillation, error) {
	var prodName, baseUnit string
	err := r.pool.QueryRow(ctx, `SELECT name, base_unit FROM canonical_products WHERE id = $1`, canonicalProductID).Scan(&prodName, &baseUnit)
	if err != nil {
		return nil, fmt.Errorf("canonical product not found: %w", err)
	}

	oscillation := &domain.MonthlyPriceOscillation{
		CanonicalProductID: canonicalProductID.String(),
		ProductName:        prodName,
		BaseUnit:           baseUnit,
	}

	periodsQuery := `
		SELECT 
			CASE 
				WHEN EXTRACT(DAY FROM p.purchased_at) BETWEEN 1 AND 7 THEN 'Dias 01-07'
				WHEN EXTRACT(DAY FROM p.purchased_at) BETWEEN 8 AND 14 THEN 'Dias 08-14'
				WHEN EXTRACT(DAY FROM p.purchased_at) BETWEEN 15 AND 21 THEN 'Dias 15-21'
				ELSE 'Dias 22-31'
			END as period_label,
			AVG(pi.normalized_price) as avg_price,
			COUNT(pi.id) as item_count
		FROM purchase_items pi
		JOIN purchases p ON p.id = pi.purchase_id
		WHERE pi.canonical_product_id = $1 AND p.status = 'CONFIRMED'
		GROUP BY period_label
		ORDER BY period_label ASC
	`
	rows, err := r.pool.Query(ctx, periodsQuery, canonicalProductID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m domain.PeriodMetric
		if err := rows.Scan(&m.PeriodLabel, &m.AveragePrice, &m.TotalCount); err != nil {
			return nil, err
		}
		oscillation.Periods = append(oscillation.Periods, m)
	}

	return oscillation, nil
}

// GetGlobalSummary calcula totais por morador e mercados
func (r *purchaseRepo) GetGlobalSummary(ctx context.Context, month time.Time) (*domain.GlobalSummaryView, error) {
	startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	summary := &domain.GlobalSummaryView{
		SpendingByUser: make(map[string]float64),
	}

	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(id)
		FROM purchases
		WHERE purchased_at BETWEEN $1 AND $2 AND status = 'CONFIRMED'
	`, startOfMonth, endOfMonth).Scan(&summary.CurrentMonthTotal, &summary.TotalPurchases)

	userRows, err := r.pool.Query(ctx, `
		SELECT COALESCE(u.name, 'Não Definido'), SUM(p.total_amount)
		FROM purchases p
		LEFT JOIN household_users u ON u.id = p.user_id
		WHERE p.purchased_at BETWEEN $1 AND $2 AND p.status = 'CONFIRMED'
		GROUP BY u.name
	`, startOfMonth, endOfMonth)
	if err == nil {
		defer userRows.Close()
		for userRows.Next() {
			var name string
			var total float64
			if err := userRows.Scan(&name, &total); err == nil {
				summary.SpendingByUser[name] = total
			}
		}
	}

	storeRows, err := r.pool.Query(ctx, `
		SELECT s.name, SUM(p.total_amount)
		FROM purchases p
		JOIN stores s ON s.id = p.store_id
		WHERE p.purchased_at BETWEEN $1 AND $2 AND p.status = 'CONFIRMED'
		GROUP BY s.name
		ORDER BY SUM(p.total_amount) DESC
		LIMIT 5
	`, startOfMonth, endOfMonth)
	if err == nil {
		defer storeRows.Close()
		for storeRows.Next() {
			var s domain.StoreShareMetric
			if err := storeRows.Scan(&s.StoreName, &s.TotalSpent); err == nil {
				if summary.CurrentMonthTotal > 0 {
					s.Percentage = (s.TotalSpent / summary.CurrentMonthTotal) * 100
				}
				summary.TopStores = append(summary.TopStores, s)
			}
		}
	}

	return summary, nil
}
