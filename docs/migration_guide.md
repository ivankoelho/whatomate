# 📦 Guia de Migração — Whatomate
## Backup e Restauração entre Instalações

> **Versão:** 1.0 · **Data:** 26/06/2026  
> **Abordagem:** Dump completo PostgreSQL — 100% fiel (contatos, mensagens, fluxos, configurações, usuários, mídia cacheada)

---

## 🎯 Visão Geral

Este guia cobre a migração **total** de uma instalação Whatomate (origem) para outra (destino), preservando:

| O que é migrado | Método |
|---|---|
| Contatos + histórico de mensagens | PostgreSQL dump |
| Fluxos de chatbot | PostgreSQL dump |
| Usuários e permissões | PostgreSQL dump |
| Tags, respostas prontas | PostgreSQL dump |
| Configurações da organização | PostgreSQL dump |
| Arquivos de mídia (local) | `rsync` ou `cp` |
| Arquivos de mídia (Meta CDN) | ⚠️ URLs expiram — ver seção específica |

---

## ⚠️ Sobre Mídias do WhatsApp (Meta CDN)

As mídias enviadas/recebidas via WhatsApp são hospedadas nos servidores da Meta. As URLs armazenadas no banco **expiram em aproximadamente 30 dias**.

- **Mídias recentes (< 30 dias):** A URL ainda funciona. O frontend carrega normalmente.
- **Mídias antigas (> 30 dias):** A URL expira. A mensagem aparece, mas a mídia não carrega — isso é uma **limitação da API da Meta**, não um bug da migração.
- **Não há forma de re-hospedar** mídias da Meta em outra conta sem reenvio pelo usuário.

---

## 🔧 Pré-requisitos

### Na máquina ORIGEM
```bash
# Verificar PostgreSQL
psql --version
pg_dump --version

# Verificar acesso ao banco
psql -U postgres -d whatomate -c "\l"
```

### Na máquina DESTINO
```bash
# Whatomate instalado e parado
systemctl stop whatomate   # ou docker compose stop

# PostgreSQL rodando
systemctl status postgresql

# Banco destino criado (se ainda não existir)
psql -U postgres -c "CREATE DATABASE whatomate;"
```

---

## 📋 PASSO A PASSO COMPLETO

---

### 1. Backup Completo na Origem

#### 1.1 Dump do banco de dados
```bash
# Na máquina ORIGEM — como root ou usuário postgres

# Dump completo em formato custom (comprimido, restaurável seletivamente)
pg_dump \
  -U postgres \
  -d whatomate \
  -F c \
  -Z 9 \
  -v \
  -f /opt/backup/whatomate_$(date +%Y%m%d_%H%M%S).dump

# Verificar o arquivo gerado
ls -lh /opt/backup/whatomate_*.dump
```

> 💡 **Formato `-F c`** (custom) permite restauração seletiva de tabelas e é comprimido com gzip nível 9. Preferível ao `.sql` puro para grandes volumes.

#### 1.2 Backup dos arquivos de mídia locais
```bash
# Se o Whatomate armazena mídia localmente (uploads, áudios de org)
# Localizar o diretório de dados
grep -r "upload" /opt/whatomate/config.toml 2>/dev/null || \
  grep -r "media" /opt/whatomate/config.toml 2>/dev/null

# Compactar diretório de mídia (ajustar caminho conforme sua config)
tar -czf /opt/backup/whatomate_media_$(date +%Y%m%d).tar.gz \
  /opt/whatomate/uploads/   # ajustar conforme seu ambiente

ls -lh /opt/backup/
```

#### 1.3 Backup do arquivo de configuração
```bash
cp /opt/whatomate/config.toml /opt/backup/config_origem.toml
```

---

### 2. Transferência dos Arquivos para o Destino

#### Opção A — scp (SSH direto)
```bash
# Na ORIGEM — enviar para o DESTINO
scp /opt/backup/whatomate_*.dump root@IP_DESTINO:/opt/backup/
scp /opt/backup/whatomate_media_*.tar.gz root@IP_DESTINO:/opt/backup/
```

#### Opção B — rsync (melhor para grandes volumes, retomável)
```bash
rsync -avz --progress \
  /opt/backup/ \
  root@IP_DESTINO:/opt/backup/
```

---

### 3. Restauração no Destino

#### 3.1 Parar a aplicação
```bash
# Na máquina DESTINO
systemctl stop whatomate
# ou
cd /opt && docker compose stop
```

#### 3.2 Restaurar o banco de dados
```bash
# Na máquina DESTINO

# Dropar e recriar o banco (limpo)
psql -U postgres -c "DROP DATABASE IF EXISTS whatomate;"
psql -U postgres -c "CREATE DATABASE whatomate;"

# Restaurar o dump
pg_restore \
  -U postgres \
  -d whatomate \
  -F c \
  -v \
  /opt/backup/whatomate_YYYYMMDD_HHMMSS.dump

# Verificar tabelas restauradas
psql -U postgres -d whatomate -c "\dt"
```

#### 3.3 Restaurar arquivos de mídia
```bash
# Na máquina DESTINO
cd /
tar -xzf /opt/backup/whatomate_media_YYYYMMDD.tar.gz

# Ajustar permissões se necessário
chown -R whatomate:whatomate /opt/whatomate/uploads/
```

#### 3.4 Atualizar configurações
```bash
# Editar config.toml do DESTINO com as credenciais da nova instalação
# (banco, secrets, Meta tokens — NÃO copiar o config da origem puro)
nano /opt/whatomate/config.toml
```

---

### 4. Ajustes Pós-Migração

#### 4.1 Verificar integridade do banco
```bash
psql -U postgres -d whatomate << 'EOF'
-- Contagem de registros por tabela principal
SELECT 'contacts'      AS tabela, COUNT(*) FROM contacts
UNION ALL
SELECT 'messages',               COUNT(*) FROM messages
UNION ALL
SELECT 'users',                  COUNT(*) FROM users
UNION ALL
SELECT 'organizations',          COUNT(*) FROM organizations
UNION ALL
SELECT 'chatbots',               COUNT(*) FROM chatbots
UNION ALL
SELECT 'tags',                   COUNT(*) FROM tags
UNION ALL
SELECT 'canned_responses',       COUNT(*) FROM canned_responses;
EOF
```

#### 4.2 Resetar senhas de usuários (recomendado)
```bash
# Após a migração, todos os usuários terão as mesmas senhas da origem.
# Recomenda-se forçar troca de senha no primeiro acesso ou resetar via painel.
psql -U postgres -d whatomate -c \
  "UPDATE users SET must_change_password = true WHERE is_active = true;"
```

#### 4.3 Reconectar WhatsApp Accounts
```bash
# As credenciais da Meta (tokens de acesso, webhook secrets) são
# específicas por instalação. Após restaurar, acesse:
# Configurações → WhatsApp Accounts → reconecte cada conta
#
# O número de telefone permanece o mesmo, mas o token precisa
# ser reconfigurado ou validado.
```

#### 4.4 Iniciar o serviço no destino
```bash
systemctl start whatomate
# ou
cd /opt && docker compose up -d

# Verificar logs
journalctl -u whatomate -f
# ou
docker compose logs -f
```

---

### 5. Validação Final

```bash
# Acessar o painel e verificar:
# [x] Login de usuários funciona
# [x] Contatos aparecem na inbox
# [x] Histórico de mensagens visível
# [x] Fluxos de chatbot carregados
# [x] Tags e respostas prontas presentes
# [x] WhatsApp Account reconectada (teste de envio)
# [x] Notificações WebSocket funcionando (badge de não lidos)
```

---

## 🔄 Estratégia de Migração com Zero Downtime (opcional)

Se você precisar migrar **sem interromper** a instalação de origem:

```
Dia 1: Criar dump da origem → restaurar no destino (dados históricos)
Dia 2: Reconectar WhatsApp Account no destino (tráfego novo vai para o destino)
Dia 3: Desligar a origem após confirmar que tudo opera no destino
```

> **Atenção:** O número WhatsApp só pode estar conectado a **uma** instalação por vez. A reconexão no destino desconecta automaticamente a origem.

---

## 🗓️ Rotina de Backup Automático (recomendado)

Adicione ao crontab da origem para backups diários:

```bash
# crontab -e
0 2 * * * pg_dump -U postgres -d whatomate -F c -Z 9 -f /opt/backup/whatomate_$(date +\%Y\%m\%d).dump && find /opt/backup -name "*.dump" -mtime +7 -delete
```

> Mantém os últimos 7 dias de backup e roda às 02:00 todo dia.

---

## 📞 Contato de Suporte

Em caso de erros durante a restauração, verifique:

1. Versão do PostgreSQL — origem e destino devem ser **compatíveis** (mesma major version, ex: ambos PG15)
2. Espaço em disco no destino — reserve **3x o tamanho do dump**
3. Permissões do usuário PostgreSQL

```bash
# Verificar versão
psql --version
pg_dump --version
```

---

*Documento gerado automaticamente pelo Professional Dev AI — Whatomate Migration Guide v1.0*
