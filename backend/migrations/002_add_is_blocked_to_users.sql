-- Migração para adicionar coluna is_blocked na tabela de moradores
ALTER TABLE household_users ADD COLUMN IF NOT EXISTS is_blocked BOOLEAN NOT NULL DEFAULT FALSE;
