<template>
  <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
    <div class="bg-white rounded-xl shadow-2xl w-full max-w-lg p-6">

      <!-- Header -->
      <div class="flex items-center justify-between mb-6">
        <h2 class="text-lg font-semibold text-slate-800">
          {{ mode === 'export' ? 'Exportar' : 'Importar' }} Fluxos e Palavras-chave
        </h2>
        <button @click="close" class="text-slate-400 hover:text-slate-600 transition">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <!-- EXPORT MODE -->
      <div v-if="mode === 'export'">
        <p class="text-sm text-slate-500 mb-6">
          Clique em <strong>Baixar JSON</strong> para exportar todos os fluxos e
          palavras-chave desta organização.
        </p>
        <button
          @click="handleExport"
          :disabled="store.isExporting"
          class="w-full py-2.5 px-4 bg-blue-600 hover:bg-blue-700 disabled:opacity-50
                 text-white text-sm font-medium rounded-lg transition"
        >
          <span v-if="store.isExporting">Exportando…</span>
          <span v-else>⬇ Baixar JSON</span>
        </button>
      </div>

      <!-- IMPORT MODE -->
      <div v-else>
        <p class="text-sm text-slate-500 mb-4">
          Selecione o arquivo JSON gerado pelo Whatomate.
        </p>

        <!-- Drop zone -->
        <label
          class="flex flex-col items-center justify-center w-full h-36 border-2 border-dashed
                 border-slate-300 hover:border-blue-400 rounded-lg cursor-pointer bg-slate-50
                 transition mb-4"
          @dragover.prevent
          @drop.prevent="onDrop"
        >
          <svg class="w-8 h-8 text-slate-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/>
          </svg>
          <span class="text-sm text-slate-500">
            Arraste o arquivo ou <span class="text-blue-600 font-medium">clique aqui</span>
          </span>
          <input type="file" accept=".json" class="hidden" @change="onFileChange" />
        </label>

        <!-- Preview -->
        <div v-if="preview" class="bg-slate-50 border border-slate-200 rounded-lg p-4 mb-4 text-sm">
          <p class="font-medium text-slate-700 mb-1">📁 {{ selectedFile?.name }}</p>
          <p class="text-slate-500">
            🔄 {{ preview.flows?.length ?? 0 }} fluxo(s)&nbsp;·&nbsp;
            🔑 {{ preview.keywords?.length ?? 0 }} palavra(s)-chave
          </p>
          <p v-if="preview.exported_at" class="text-xs text-slate-400 mt-1">
            Exportado em: {{ formatDate(preview.exported_at) }}
          </p>
        </div>

        <!-- Import result -->
        <div v-if="importResult" class="mb-4 p-3 rounded-lg bg-green-50 border border-green-200">
          <p class="text-sm font-medium text-green-700">✅ {{ importResult.message }}</p>
          <p class="text-xs text-slate-500 mt-1">
            {{ importResult.imported_flows }} fluxo(s) · {{ importResult.imported_keywords }} palavra(s)-chave importado(s)
          </p>
          <ul v-if="importResult.errors?.length" class="mt-2 text-xs text-red-600 list-disc list-inside">
            <li v-for="e in importResult.errors" :key="e">{{ e }}</li>
          </ul>
        </div>

        <button
          @click="handleImport"
          :disabled="!selectedFile || store.isImporting"
          class="w-full py-2.5 px-4 bg-green-600 hover:bg-green-700 disabled:opacity-50
                 text-white text-sm font-medium rounded-lg transition"
        >
          <span v-if="store.isImporting">Importando…</span>
          <span v-else>⬆ Importar</span>
        </button>
      </div>

      <!-- Error -->
      <p v-if="store.error || localError" class="mt-3 text-xs text-red-600">
        ⚠ {{ store.error || localError }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useChatbotExportImportStore } from '@/stores/chatbot_export_import'

const props = defineProps<{
  visible: boolean
  mode: 'export' | 'import'
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'imported'): void
}>()

// ✅ Store correto: chatbot_export_import (não useChatbotStore)
const store = useChatbotExportImportStore()

const localError   = ref('')
const selectedFile = ref<File | null>(null)
const preview      = ref<any>(null)
const importResult = ref<any>(null)

function close() {
  localError.value   = ''
  preview.value      = null
  importResult.value = null
  selectedFile.value = null
  store.reset()
  emit('close')
}

function onFileChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (file) loadFile(file)
}

function onDrop(e: DragEvent) {
  const file = e.dataTransfer?.files?.[0]
  if (file) loadFile(file)
}

function loadFile(file: File) {
  selectedFile.value = file
  const reader = new FileReader()
  reader.onload = (ev) => {
    try {
      preview.value    = JSON.parse(ev.target?.result as string)
      localError.value = ''
    } catch {
      localError.value = 'Arquivo JSON inválido.'
      preview.value    = null
    }
  }
  reader.readAsText(file)
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString('pt-BR')
}

// ✅ store.exportData() não retorna blob — o download é feito dentro de exportChatbotData()
async function handleExport() {
  localError.value = ''
  try {
    await store.exportData({})
    close()
  } catch {
    // store.error preenchido automaticamente
  }
}

// ✅ store.importData() recebe File (não string)
async function handleImport() {
  if (!selectedFile.value) return
  localError.value   = ''
  importResult.value = null
  try {
    importResult.value = await store.importData(selectedFile.value)
    emit('imported')
  } catch {
    // store.error preenchido automaticamente
  }
}
</script>
