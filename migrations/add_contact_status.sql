-- Migration: add_contact_status
-- Aplica coluna contact_status, constraint, índices e backfill de dados.
-- Idempotente: pode rodar múltiplas vezes sem efeito colateral.
-- Run: psql -U postgres -d whatomate -f migrations/add_contact_status.sql

BEGIN;

-- 1. Coluna contact_status com default 'new'
ALTER TABLE contacts
  ADD COLUMN IF NOT EXISTS contact_status VARCHAR(20) NOT NULL DEFAULT 'new';

-- 2. CHECK constraint para garantir apenas valores válidos
ALTER TABLE contacts
  DROP CONSTRAINT IF EXISTS chk_contact_status;
ALTER TABLE contacts
  ADD CONSTRAINT chk_contact_status
  CHECK (contact_status IN ('new', 'in_progress', 'resolved'));

-- 3. Índice simples para filtros por status (listagem da sidebar)
CREATE INDEX IF NOT EXISTS idx_contacts_contact_status
  ON contacts(contact_status);

-- 4. Índice composto (organização + status) — usado em todas as queries do app
CREATE INDEX IF NOT EXISTS idx_contacts_org_status
  ON contacts(organization_id, contact_status);

-- 5. Índice composto (organização + status + last_message_at DESC) para ordenação
CREATE INDEX IF NOT EXISTS idx_contacts_org_status_last_msg
  ON contacts(organization_id, contact_status, last_message_at DESC NULLS LAST);

-- 6. Backfill: contatos com mensagens recentes (≤ 30 dias) → in_progress
UPDATE contacts
SET contact_status = 'in_progress'
WHERE last_message_at IS NOT NULL
  AND last_message_at >= NOW() - INTERVAL '30 days'
  AND contact_status = 'new';

-- 7. Coluna last_message_preview (prévia da última mensagem na sidebar)
ALTER TABLE contacts
  ADD COLUMN IF NOT EXISTS last_message_preview TEXT DEFAULT '';

-- 8. Backfill de last_message_preview com a última mensagem de cada contato
UPDATE contacts c
SET last_message_preview = COALESCE(
  CASE
    WHEN m.message_type = 'image'    THEN '📷 Imagem'
    WHEN m.message_type = 'video'    THEN '🎬 Vídeo'
    WHEN m.message_type = 'audio'    THEN '🎤 Áudio'
    WHEN m.message_type = 'document' THEN '📎 Documento'
    WHEN m.message_type = 'sticker'  THEN '😀 Sticker'
    WHEN m.message_type = 'location' THEN '📍 Localização'
    WHEN m.message_type = 'template' THEN '📋 Template'
    ELSE LEFT(COALESCE(m.content, ''), 60)
  END,
  ''
)
FROM (
  SELECT DISTINCT ON (contact_id)
    contact_id,
    message_type,
    content
  FROM messages
  ORDER BY contact_id, created_at DESC
) m
WHERE m.contact_id = c.id;

COMMIT;

-- Verificação pós-migration
SELECT contact_status, COUNT(*) AS total
FROM contacts
GROUP BY contact_status
ORDER BY contact_status;
