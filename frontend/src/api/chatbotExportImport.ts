import { api } from '@/services/api'

export interface ExportRequest {
  flow_ids?: string[]
  keyword_ids?: string[]
}

export interface ImportResult {
  message: string
  imported_flows: number
  imported_keywords: number
}

/**
 * Exports chatbot flows and keywords and triggers a browser file download.
 * Omitting payload (or empty arrays) exports everything in the organisation.
 */
export async function exportChatbotData(payload: ExportRequest = {}): Promise<void> {
  const response = await api.post('/api/chatbot/export', payload)
  const blob = new Blob([JSON.stringify(response.data.data, null, 2)], { type: 'application/json' })
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
 */
export async function importChatbotData(file: File): Promise<ImportResult> {
  const text = await file.text()
  const { data } = await api.post<{ data: ImportResult }>('/api/chatbot/import', JSON.parse(text))
  return data.data
}
