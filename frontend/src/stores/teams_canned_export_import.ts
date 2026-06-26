import { defineStore } from 'pinia'
import { ref } from 'vue'
import { exportTeamsAndCanned, importTeamsAndCanned } from '@/api/teamsCannedExportImport'
import type { ImportResult } from '@/api/teamsCannedExportImport'

export const useTeamsCannedExportImportStore = defineStore('teamsCannedExportImport', () => {
  const isExporting = ref(false)
  const isImporting = ref(false)
  const lastImportResult = ref<ImportResult | null>(null)
  const error = ref<string | null>(null)

  async function exportData(): Promise<void> {
    isExporting.value = true
    error.value = null
    try {
      await exportTeamsAndCanned()
    } catch (err: any) {
      error.value = err?.response?.data?.message ?? err?.message ?? 'Falha ao exportar'
      throw err
    } finally {
      isExporting.value = false
    }
  }

  async function importData(file: File): Promise<ImportResult> {
    isImporting.value = true
    error.value = null
    lastImportResult.value = null
    try {
      const result = await importTeamsAndCanned(file)
      lastImportResult.value = result
      return result
    } catch (err: any) {
      error.value = err?.response?.data?.message ?? err?.message ?? 'Falha ao importar'
      throw err
    } finally {
      isImporting.value = false
    }
  }

  function reset() {
    isExporting.value = false
    isImporting.value = false
    lastImportResult.value = null
    error.value = null
  }

  return { isExporting, isImporting, lastImportResult, error, exportData, importData, reset }
})
