import api from '@/api'

export interface ExportRequest {
  flow_ids?: string[]
  keyword_ids?: string[]
}

export interface ImportResult {
  imported_flows: number
  imported_keywords: number
  skipped_flows: string[]
  skipped_keywords: string[]
}

/**
 * Exports chatbot flows and keywords and triggers a browser file download.
 * Omit payload (or send empty arrays) to export everything in the organisation.
 */
export async function exportChatbotData(payload: ExportRequest = {}): Promise<void> {
  const response = await api.post('/api/chatbot/export', payload, {
    responseType: 'blob',
  })
  const url = URL.createObjectURL(new Blob([response.data], { type: 'application/json' }))
  const a = document.createElement('a')
  a.href = url
  a.download = `whatomate-export-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`
  a.click()
  URL.revokeObjectURL(url)
}

/**
 * Imports chatbot flows and keywords from a JSON file.
 * @param file      - The .json file chosen by the user.
 * @param overwrite - When true, always creates new records (ignores name collisions).
 */
export async function importChatbotData(file: File, overwrite = false): Promise<ImportResult> {
  const text = await file.text()
  const params = overwrite ? '?overwrite=true' : ''
  const { data } = await api.post<{ data: ImportResult }>(`/api/chatbot/import${params}`, JSON.parse(text))
  return data.data
}
