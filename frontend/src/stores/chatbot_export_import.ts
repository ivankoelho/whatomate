// ─────────────────────────────────────────────────────────────────────────────
// INSTRUÇÃO: Adicione as duas funções abaixo dentro do useChatbotStore()
// em frontend/src/stores/chatbot.ts, junto com as outras actions existentes.
// ─────────────────────────────────────────────────────────────────────────────

// ---------- EXPORT ----------
async function exportData(): Promise<Blob> {
  const res = await api.post('/api/chatbot/export', {}, { responseType: 'blob' })
  if (!res.ok) throw new Error('Falha ao exportar dados')
  return res.blob()
}

// ---------- IMPORT ----------
async function importData(jsonText: string): Promise<ImportResult> {
  const res = await api.post('/api/chatbot/import', jsonText, {
    headers: { 'Content-Type': 'application/json' },
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.message ?? 'Falha ao importar dados')
  }
  return res.json()
}

interface ImportResult {
  imported_flows:    number
  imported_keywords: number
  errors:            string[]
  message:           string
}

// Adicione exportData e importData no return {} do store:
// return { ...outrasActions, exportData, importData }
