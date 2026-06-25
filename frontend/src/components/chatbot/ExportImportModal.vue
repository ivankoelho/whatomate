<script setup lang="ts">
import { ref, computed } from 'vue'
import { exportChatbotData, importChatbotData, type ImportResult } from '@/api/chatbotExportImport'
import { useToast } from '@/composables/useToast'

const { toast } = useToast()

// ── state ──────────────────────────────────────────────────────────────────
const open = ref(false)
const tab  = ref<'export' | 'import'>('export')

const loading      = ref(false)

const file         = ref<File | null>(null)
const filePreview  = ref<{ flows: number; keywords: number } | null>(null)
const fileError    = ref('')
const overwrite    = ref(false)
const importing    = ref(false)
const importResult = ref<ImportResult | null>(null)

// ── computed ───────────────────────────────────────────────────────────────
const canImport = computed(() => file.value !== null && filePreview.value !== null && !fileError.value)

// ── helpers ────────────────────────────────────────────────────────────────
function resetImport() {
  file.value         = null
  filePreview.value  = null
  fileError.value    = ''
  importResult.value = null
  overwrite.value    = false
}

function onFileChange(e: Event) {
  resetImport()
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  if (!f.name.endsWith('.json')) {
    fileError.value = 'Apenas arquivos .json são aceitos.'
    return
  }
  file.value = f
  const reader = new FileReader()
  reader.onload = (ev) => {
    try {
      const json = JSON.parse(ev.target?.result as string)
      if (json.version !== '1.0') throw new Error('versão inválida')
      filePreview.value = {
        flows:    (json.flows    ?? []).length,
        keywords: (json.keywords ?? []).length,
      }
    } catch {
      fileError.value = 'Arquivo inválido ou versão incompatível.'
    }
  }
  reader.readAsText(f)
}

// ── actions ────────────────────────────────────────────────────────────────
async function handleExport() {
  loading.value = true
  try {
    await exportChatbotData()
    toast({ title: 'Exportação concluída', description: 'Arquivo baixado com sucesso.', variant: 'success' })
    open.value = false
  } catch {
    toast({ title: 'Erro ao exportar', description: 'Tente novamente.', variant: 'destructive' })
  } finally {
    loading.value = false
  }
}

async function handleImport() {
  if (!file.value) return
  importing.value = true
  try {
    importResult.value = await importChatbotData(file.value, overwrite.value)
    toast({
      title: 'Importação concluída',
      description: `${importResult.value.imported_flows} fluxo(s) e ${importResult.value.imported_keywords} keyword(s) importado(s).`,
      variant: 'success',
    })
  } catch (err: any) {
    toast({
      title: 'Erro ao importar',
      description: err?.response?.data?.message ?? 'Tente novamente.',
      variant: 'destructive',
    })
  } finally {
    importing.value = false
  }
}
</script>

<template>
  <slot name="trigger" :open="() => (open = true)" />

  <Dialog v-model:open="open" @update:open="(v) => { if (!v) resetImport() }">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Exportar / Importar</DialogTitle>
        <DialogDescription>
          Transfira fluxos e palavras-chave entre ambientes.
        </DialogDescription>
      </DialogHeader>

      <Tabs v-model="tab" class="mt-2">
        <TabsList class="w-full">
          <TabsTrigger value="export" class="flex-1">Exportar</TabsTrigger>
          <TabsTrigger value="import" class="flex-1">Importar</TabsTrigger>
        </TabsList>

        <!-- Export -->
        <TabsContent value="export" class="space-y-4 pt-4">
          <p class="text-sm text-muted-foreground">
            Baixe todos os fluxos e palavras-chave desta organização em um único arquivo
            <code>.json</code>.
          </p>
          <Button class="w-full" :disabled="loading" @click="handleExport">
            <span v-if="loading">Exportando…</span>
            <span v-else>⬇ Baixar JSON</span>
          </Button>
        </TabsContent>

        <!-- Import -->
        <TabsContent value="import" class="space-y-4 pt-4">
          <div>
            <Label for="import-file">Arquivo de exportação (.json)</Label>
            <input
              id="import-file"
              type="file"
              accept=".json"
              class="mt-1 block w-full cursor-pointer text-sm
                     file:mr-3 file:rounded file:border-0
                     file:bg-primary file:px-3 file:py-1
                     file:text-primary-foreground"
              @change="onFileChange"
            />
            <p v-if="fileError" class="mt-1 text-xs text-destructive">{{ fileError }}</p>
          </div>

          <div
            v-if="filePreview"
            class="rounded-md border bg-muted/40 p-3 text-sm space-y-1"
          >
            <p>📂 <strong>{{ filePreview.flows }}</strong> fluxo(s) encontrado(s)</p>
            <p>🔑 <strong>{{ filePreview.keywords }}</strong> palavra(s)-chave encontrada(s)</p>
          </div>

          <div v-if="filePreview" class="flex items-center gap-2">
            <Switch id="overwrite" v-model:checked="overwrite" />
            <Label for="overwrite" class="text-sm">
              Sobrescrever registros com o mesmo nome
            </Label>
          </div>

          <div
            v-if="importResult"
            class="rounded-md border border-green-500/30 bg-green-50 p-3 text-sm space-y-1 dark:bg-green-950/20"
          >
            <p>✅ <strong>{{ importResult.imported_flows }}</strong> fluxo(s) importado(s)</p>
            <p>✅ <strong>{{ importResult.imported_keywords }}</strong> keyword(s) importada(s)</p>
            <p v-if="importResult.skipped_flows.length" class="text-muted-foreground">
              Ignorados: {{ importResult.skipped_flows.join(', ') }}
            </p>
            <p v-if="importResult.skipped_keywords.length" class="text-muted-foreground">
              Keywords ignoradas: {{ importResult.skipped_keywords.join(', ') }}
            </p>
          </div>

          <Button class="w-full" :disabled="!canImport || importing" @click="handleImport">
            <span v-if="importing">Importando…</span>
            <span v-else>⬆ Importar</span>
          </Button>
        </TabsContent>
      </Tabs>
    </DialogContent>
  </Dialog>
</template>
