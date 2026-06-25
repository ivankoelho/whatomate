<script setup lang="ts">
import { ref } from 'vue'
import { chatbotService } from '@/services/api'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { toast } from '@/components/ui/toast'

const props = defineProps<{ open: boolean }>()
const emit  = defineEmits<{
  (e: 'update:open', v: boolean): void
  (e: 'imported'): void
}>()

const file    = ref<File | null>(null)
const preview = ref<{ flows: number; keywords: number } | null>(null)
const loading = ref(false)

function onFileChange(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  file.value = f

  const reader = new FileReader()
  reader.onload = (ev) => {
    try {
      const parsed = JSON.parse(ev.target?.result as string)
      preview.value = {
        flows:    Array.isArray(parsed.flows)    ? parsed.flows.length    : 0,
        keywords: Array.isArray(parsed.keywords) ? parsed.keywords.length : 0,
      }
    } catch {
      toast({ title: 'Arquivo JSON inválido', variant: 'destructive' })
      file.value    = null
      preview.value = null
    }
  }
  reader.readAsText(f)
}

async function handleImport() {
  if (!file.value) return
  loading.value = true
  try {
    const res = await chatbotService.importChatbot(file.value)
    const { flows_imported, keywords_imported } = res.data
    toast({
      title: 'Importação concluída!',
      description: `${flows_imported} fluxo(s) e ${keywords_imported} palavra(s)-chave importada(s).`,
    })
    emit('imported')
    emit('update:open', false)
  } catch (err: any) {
    const msg = err?.response?.data?.message || 'Erro ao importar arquivo'
    toast({ title: msg, variant: 'destructive' })
  } finally {
    loading.value = false
  }
}

function reset() {
  file.value    = null
  preview.value = null
}
</script>

<template>
  <Dialog :open="open" @update:open="(v) => { emit('update:open', v); if (!v) reset() }">
    <DialogContent class="max-w-md">
      <DialogHeader>
        <DialogTitle>Importar Fluxos e Palavras-chave</DialogTitle>
      </DialogHeader>

      <div class="space-y-4 py-2">
        <p class="text-sm text-muted-foreground">
          Selecione um arquivo <code class="font-mono">.json</code> gerado pela exportação do Whatomate.
          Os itens serão adicionados sem substituir os existentes.
        </p>

        <label
          class="flex flex-col items-center justify-center gap-2 border-2 border-dashed border-border rounded-lg p-6 cursor-pointer hover:bg-muted/40 transition-colors"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 16v-8m0 0-3 3m3-3 3 3M4 16v1a3 3 0 0 0 3 3h10a3 3 0 0 0 3-3v-1" />
          </svg>
          <span class="text-sm text-muted-foreground">
            {{ file ? file.name : 'Clique para selecionar o arquivo JSON' }}
          </span>
          <input type="file" accept=".json,application/json" class="hidden" @change="onFileChange" />
        </label>

        <!-- Preview -->
        <div v-if="preview" class="rounded-lg bg-muted/60 p-4 text-sm space-y-1">
          <p class="font-medium">Conteúdo encontrado no arquivo:</p>
          <p>🔄 <strong>{{ preview.flows }}</strong> fluxo(s)</p>
          <p>🔑 <strong>{{ preview.keywords }}</strong> palavra(s)-chave</p>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)">Cancelar</Button>
        <Button :disabled="!file" :loading="loading" @click="handleImport">
          Importar
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
