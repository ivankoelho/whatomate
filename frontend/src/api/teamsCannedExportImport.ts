import { api } from '@/services/api'

export interface ImportResult {
  message: string
  teams_imported: number
  canned_responses_imported: number
}

/**
 * Exporta equipes e respostas rápidas e dispara download no browser.
 */
export async function exportTeamsAndCanned(): Promise<void> {
  const response = await api.post('/api/teams-canned/export', {})
  const blob = new Blob([JSON.stringify(response.data.data ?? response.data, null, 2)], {
    type: 'application/json',
  })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `whatomate-teams-canned-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/**
 * Importa equipes e respostas rápidas a partir de um arquivo JSON exportado.
 * @param file - Arquivo .json escolhido pelo usuário.
 */
export async function importTeamsAndCanned(file: File): Promise<ImportResult> {
  const text = await file.text()
  const { data } = await api.post<{ data: ImportResult }>('/api/teams-canned/import', JSON.parse(text))
  return data.data
}
