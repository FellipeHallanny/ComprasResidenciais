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

// RunAutoMigrations aplica a criação do esquema completo e migrações caso o banco esteja vazio
func RunAutoMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS vector;`,
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,
		`CREATE TABLE IF NOT EXISTS household_users (
		    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		    name VARCHAR(255) NOT NULL,
		    email VARCHAR(255),
		    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
		    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS stores (
		    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		    name VARCHAR(255) NOT NULL,
		    cnpj VARCHAR(30),
		    address TEXT,
		    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS canonical_products (
		    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		    name VARCHAR(255) NOT NULL,
		    category VARCHAR(100) DEFAULT 'Geral',
		    base_unit VARCHAR(20) NOT NULL,
		    embedding vector(768),
		    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS canonical_products_embedding_hnsw_idx 
		ON canonical_products USING hnsw (embedding vector_cosine_ops);`,
		`CREATE TABLE IF NOT EXISTS purchases (
		    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		    user_id UUID REFERENCES household_users(id) ON DELETE SET NULL,
		    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
		    total_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
		    purchased_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
		    origin VARCHAR(50) NOT NULL DEFAULT 'MANUAL',
		    fiscal_key VARCHAR(100),
		    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS purchases_purchased_at_idx ON purchases (purchased_at);`,
		`CREATE INDEX IF NOT EXISTS purchases_status_idx ON purchases (status);`,
		`CREATE TABLE IF NOT EXISTS purchase_items (
		    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		    purchase_id UUID NOT NULL REFERENCES purchases(id) ON DELETE CASCADE,
		    raw_description VARCHAR(255) NOT NULL,
		    quantity NUMERIC(10, 3) NOT NULL DEFAULT 1.0,
		    package_size NUMERIC(10, 3) NOT NULL DEFAULT 1.0,
		    package_unit VARCHAR(20) NOT NULL,
		    total_units NUMERIC(10, 3) NOT NULL,
		    total_price NUMERIC(12, 2) NOT NULL,
		    unit_price NUMERIC(12, 2) NOT NULL,
		    normalized_price NUMERIC(12, 4) NOT NULL,
		    canonical_product_id UUID REFERENCES canonical_products(id) ON DELETE SET NULL,
		    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS purchase_items_canonical_idx ON purchase_items (canonical_product_id);`,
		`CREATE INDEX IF NOT EXISTS purchase_items_purchase_id_idx ON purchase_items (purchase_id);`,
		`ALTER TABLE household_users ADD COLUMN IF NOT EXISTS is_blocked BOOLEAN NOT NULL DEFAULT FALSE;`,
		`ALTER TABLE household_users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';`,
		`ALTER TABLE household_users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'MEMBER';`,
		`ALTER TABLE household_users ADD COLUMN IF NOT EXISTS phone VARCHAR(50) NOT NULL DEFAULT '';`,
		`CREATE UNIQUE INDEX IF NOT EXISTS household_users_email_idx ON household_users (email) WHERE email IS NOT NULL AND email <> '';`,
	}

	for _, m := range migrations {
		if _, err := pool.Exec(ctx, m); err != nil {
			log.Printf("Warning: error running migration statement: %v", err)
			return err
		}
	}
	return nil
}

// ResetDatabase limpa todas as tabelas reiniciando dados para um estado limpo
func ResetDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	query := `TRUNCATE TABLE purchase_items, purchases, canonical_products, stores, household_users RESTART IDENTITY CASCADE;`
	_, err := pool.Exec(ctx, query)
	return err
}

// --- UserRepository Implementation ---

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) ports.UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, u *domain.User) error {
	if u.Role == "" {
		u.Role = domain.UserRoleMember
	}
	query := `INSERT INTO household_users (id, name, email, phone, password_hash, role, is_blocked, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(ctx, query, u.ID, u.Name, u.Email, u.Phone, u.PasswordHash, u.Role, u.IsBlocked, u.CreatedAt, u.UpdatedAt)
	return err
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT id, name, COALESCE(email, ''), COALESCE(phone, ''), COALESCE(password_hash, ''), COALESCE(role, 'MEMBER'), is_blocked, created_at, updated_at FROM household_users WHERE id = $1`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.Role, &u.IsBlocked, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, name, COALESCE(email, ''), COALESCE(phone, ''), COALESCE(password_hash, ''), COALESCE(role, 'MEMBER'), is_blocked, created_at, updated_at FROM household_users WHERE LOWER(email) = LOWER($1)`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.Role, &u.IsBlocked, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) ListAll(ctx context.Context) ([]domain.User, error) {
	query := `SELECT id, name, COALESCE(email, ''), COALESCE(phone, ''), COALESCE(password_hash, ''), COALESCE(role, 'MEMBER'), is_blocked, created_at, updated_at FROM household_users ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.Role, &u.IsBlocked, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *userRepo) Update(ctx context.Context, u *domain.User) error {
	if u.Role == "" {
		u.Role = domain.UserRoleMember
	}
	query := `UPDATE household_users SET name = $2, email = $3, phone = $4, password_hash = $5, role = $6, is_blocked = $7, updated_at = $8 WHERE id = $1`
	res, err := r.pool.Exec(ctx, query, u.ID, u.Name, u.Email, u.Phone, u.PasswordHash, u.Role, u.IsBlocked, time.Now())
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
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

func (r *userRepo) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM household_users`
	var count int64
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
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

func (r *userRepo) CountAdmins(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM household_users WHERE role = 'ADMIN' AND password_hash <> '' AND is_blocked = FALSE`
	var count int64
	err := r.pool.QueryRow(ctx, query).Scan(&count)
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

	stores := make([]domain.Store, 0)
	for rows.Next() {
		var s domain.Store
		if err := rows.Scan(&s.ID, &s.Name, &s.CNPJ, &s.Address, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		stores = append(stores, s)
	}
	return stores, nil
}

func (r *storeRepo) Update(ctx context.Context, s *domain.Store) error {
	query := `UPDATE stores SET name = $2, cnpj = $3, address = $4, updated_at = $5 WHERE id = $1`
	res, err := r.pool.Exec(ctx, query, s.ID, s.Name, s.CNPJ, s.Address, time.Now())
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *storeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM stores WHERE id = $1`
	res, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
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

func (r *productRepo) SearchSemantic(ctx context.Context, vec pgvector.Vector, limit int) ([]domain.CanonicalProduct, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `
		SELECT id, name, COALESCE(category, 'Geral'), base_unit, created_at, updated_at
		FROM canonical_products
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1 ASC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, vec, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prods := make([]domain.CanonicalProduct, 0)
	for rows.Next() {
		var p domain.CanonicalProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.BaseUnit, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prods = append(prods, p)
	}
	return prods, nil
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

	prods := make([]domain.CanonicalProduct, 0)
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

	prods := make([]domain.CanonicalProduct, 0)
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

	p.Items = make([]domain.PurchaseItem, 0)
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

	purchases := make([]domain.Purchase, 0)
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
		RecentPurchases:    make([]domain.RecentItemPurchase, 0),
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
		Periods:            make([]domain.PeriodMetric, 0),
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

func (r *purchaseRepo) GetGlobalSummary(ctx context.Context, month time.Time) (*domain.GlobalSummaryView, error) {
	startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	summary := &domain.GlobalSummaryView{
		SpendingByUser: make(map[string]float64),
		TopStores:      make([]domain.StoreShareMetric, 0),
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

func (r *purchaseRepo) GetCatalogWithPrices(ctx context.Context) ([]domain.CatalogItemView, error) {
	query := `
		WITH product_stats AS (
			SELECT 
				pi.canonical_product_id,
				MAX(p.purchased_at) as last_purchased_at,
				COUNT(DISTINCT p.id) as total_purchases
			FROM purchase_items pi
			JOIN purchases p ON p.id = pi.purchase_id
			WHERE pi.canonical_product_id IS NOT NULL AND p.status = 'CONFIRMED'
			GROUP BY pi.canonical_product_id
		),
		ranked_items AS (
			SELECT 
				pi.canonical_product_id,
				pi.normalized_price,
				pi.package_size,
				pi.package_unit,
				s.id AS store_id,
				s.name AS store_name,
				COALESCE(s.address, '') AS store_address,
				p.purchased_at,
				ROW_NUMBER() OVER (
					PARTITION BY pi.canonical_product_id 
					ORDER BY pi.normalized_price ASC, p.purchased_at DESC
				) as rn
			FROM purchase_items pi
			JOIN purchases p ON p.id = pi.purchase_id
			JOIN stores s ON s.id = p.store_id
			WHERE pi.canonical_product_id IS NOT NULL AND p.status = 'CONFIRMED'
		)
		SELECT 
			cp.id,
			cp.name,
			COALESCE(cp.category, 'Geral'),
			cp.base_unit,
			COALESCE(ri.normalized_price, 0),
			COALESCE(ri.store_name, ''),
			ri.store_id,
			COALESCE(ri.store_address, ''),
			COALESCE(CONCAT(ri.package_size, ri.package_unit), ''),
			ri.purchased_at,
			ps.last_purchased_at,
			cp.created_at,
			COALESCE(ps.total_purchases, 0)
		FROM canonical_products cp
		LEFT JOIN ranked_items ri ON ri.canonical_product_id = cp.id AND ri.rn = 1
		LEFT JOIN product_stats ps ON ps.canonical_product_id = cp.id
		ORDER BY cp.name ASC;
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query catalog with prices: %w", err)
	}
	defer rows.Close()

	catalog := make([]domain.CatalogItemView, 0)
	for rows.Next() {
		var (
			id              uuid.UUID
			name            string
			category        string
			baseUnit        string
			lowestPrice     float64
			storeName       string
			storeID         *uuid.UUID
			storeAddress    string
			packageInfo     string
			purchasedAt     *time.Time
			lastPurchasedAt *time.Time
			createdAt       *time.Time
			totalPurchases  int
		)
		if err := rows.Scan(&id, &name, &category, &baseUnit, &lowestPrice, &storeName, &storeID, &storeAddress, &packageInfo, &purchasedAt, &lastPurchasedAt, &createdAt, &totalPurchases); err != nil {
			return nil, err
		}

		var storeIDStr *string
		if storeID != nil {
			s := storeID.String()
			storeIDStr = &s
		}

		catalog = append(catalog, domain.CatalogItemView{
			ID:                   id.String(),
			Name:                 name,
			Category:             category,
			BaseUnit:             baseUnit,
			LowestPricePerUnit:   lowestPrice,
			CheapestStoreName:    storeName,
			CheapestStoreID:      storeIDStr,
			CheapestStoreAddress: storeAddress,
			CheapestPackageInfo:  packageInfo,
			CheapestPurchaseDate: purchasedAt,
			LastPurchasedAt:      lastPurchasedAt,
			CreatedAt:            createdAt,
			TotalPurchases:       totalPurchases,
		})
	}
	return catalog, nil
}
