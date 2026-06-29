import { api } from '@/services/api'

export interface ExportRequest {
  flow_ids?: string[]
  keyword_ids?: string[]
}

export interface ImportResult {
  message: string
  flows_imported: number
  keywords_imported: number
}

/**
 * Exports chatbot flows and keywords and triggers a browser file download.
 * Omitting payload (or empty arrays) exports everything in the organisation.
 *
 * O backend retorna JSON cru (sem envelope fastglue), por isso usamos
 * response.data diretamente — não response.data.data.
 */
export async function exportChatbotData(payload: ExportRequest = {}): Promise<void> {
  const response = await api.post('/api/chatbot/export', payload)

  // O backend usa SetBody(raw) — retorna JSON puro, sem envelope { data: ... }
  // O axios coloca esse JSON em response.data diretamente.
  const exportData = response.data?.data ?? response.data

  const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `whatomate-export-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/**
 * Imports chatbot flows and keywords from a JSON file.
 * @param file - The .json file chosen by the user.
 *
 * Envia o conteúdo como string raw com Content-Type: application/json
 * para garantir que o body chegue ao backend exatamente como está no arquivo,
 * sem re-serialização pelo axios que causaria 'Invalid import file format'.
 */
export async function importChatbotData(file: File): Promise<ImportResult> {
  const text = await file.text()
  const { data } = await api.post<{ data: ImportResult }>('/api/chatbot/import', text, {
    headers: { 'Content-Type': 'application/json' },
  })
  return data.data
}
