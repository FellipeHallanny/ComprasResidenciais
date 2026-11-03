-- Habilitar extensão pgvector
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tabela de Moradores
CREATE TABLE IF NOT EXISTS household_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de Estabelecimentos Comerciais
CREATE TABLE IF NOT EXISTS stores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    cnpj VARCHAR(30),
    address TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de Produtos Canônicos (com vetor de 768 dimensões para text-embedding-004 do Gemini)
CREATE TABLE IF NOT EXISTS canonical_products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) DEFAULT 'Geral',
    base_unit VARCHAR(20) NOT NULL, -- 'kg', 'L', 'un'
    embedding vector(768),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Índice HNSW no pgvector para busca aproximada ultra-rápida (cosseno)
CREATE INDEX IF NOT EXISTS canonical_products_embedding_hnsw_idx 
ON canonical_products USING hnsw (embedding vector_cosine_ops);

-- Tabela de Compras (Sessão / Nota Fiscal)
CREATE TABLE IF NOT EXISTS purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES household_users(id) ON DELETE RESTRICT,
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    total_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    purchased_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT', -- 'DRAFT', 'CONFIRMED'
    origin VARCHAR(50) NOT NULL DEFAULT 'MANUAL', -- 'MANUAL', 'LLM_UPLOAD', 'SEFAZ'
    fiscal_key VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS purchases_purchased_at_idx ON purchases (purchased_at);
CREATE INDEX IF NOT EXISTS purchases_status_idx ON purchases (status);

-- Tabela de Itens de Compra
CREATE TABLE IF NOT EXISTS purchase_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_id UUID NOT NULL REFERENCES purchases(id) ON DELETE CASCADE,
    raw_description VARCHAR(255) NOT NULL,
    quantity NUMERIC(10, 3) NOT NULL DEFAULT 1.0,
    package_size NUMERIC(10, 3) NOT NULL DEFAULT 1.0,
    package_unit VARCHAR(20) NOT NULL,
    total_units NUMERIC(10, 3) NOT NULL,
    total_price NUMERIC(12, 2) NOT NULL,
    unit_price NUMERIC(12, 2) NOT NULL,
    normalized_price NUMERIC(12, 4) NOT NULL, -- R$ por kg, L ou un
    canonical_product_id UUID REFERENCES canonical_products(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS purchase_items_canonical_idx ON purchase_items (canonical_product_id);
CREATE INDEX IF NOT EXISTS purchase_items_purchase_id_idx ON purchase_items (purchase_id);
