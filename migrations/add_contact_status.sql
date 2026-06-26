-- Migration: add contact_status column to contacts
-- Run: psql -U postgres -d whatomate -f this_file.sql

BEGIN;

-- 1. Adicionar coluna se não existir
ALTER TABLE contacts
  ADD COLUMN IF NOT EXISTS contact_status VARCHAR(20) NOT NULL DEFAULT 'new';

-- 2. Criar índice para performance nas queries de filtro
CREATE INDEX IF NOT EXISTS idx_contacts_contact_status
  ON contacts(contact_status);

-- 3. Contatos com mensagens recentes (< 7 dias) → in_progress
UPDATE contacts
SET contact_status = 'in_progress'
WHERE last_message_at IS NOT NULL
  AND last_message_at >= NOW() - INTERVAL '7 days'
  AND contact_status = 'new';

COMMIT;

-- Verificação
SELECT contact_status, COUNT(*) FROM contacts GROUP BY contact_status;
