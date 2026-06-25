<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { chatbotService } from '@/services/api'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Download, Upload, FileJson, CheckSquare, Square } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

// ─── Props / Emits ─────────────────────────────────────────────
const props = defineProps<{
  visible: boolean
  mode: 'export' | 'import'
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'imported'): void
}>()

// ─── Estado compartilhado ──────────────────────────────────────
const loading        = ref(false)
const isOpen         = computed(() => props.visible)

// ─── Estado de EXPORT ──────────────────────────────────────────
const flows          = ref<Array<{ id: string; name: string }>>([])
const keywords       = ref<Array<{ id: string; name: string }>>([])
const selectedFlows  = ref<string[]>([])
const selectedKws    = ref<string[]>([])
const loadingLists   = ref(false)

const allFlowsSel  = computed(() => flows.value.length > 0 && selectedFlows.value.length === flows.value.length)
const allKwsSel    = computed(() => keywords.value.length > 0 && selectedKws.value.length === keywords.value.length)
const nothingSelected = computed(() => !selectedFlows.value.length && !selectedKws.value.length)

async function loadLists() {
  loadingLists.value = true
  try {
    const [fRes, kRes] = await Promise.all([
      chatbotService.listFlows({ limit: 200 }),
      chatbotService.listKeywords({ limit: 200 }),
    ])
    const fd = (fRes.data as any).data ?? fRes.data
    const kd = (kRes.data as any).data ?? kRes.data
    flows.value    = (fd.flows    ?? []).map((f: any) => ({ id: f.id, name: f.name }))
    keywords.value = (kd.rules    ?? []).map((k: any) => ({ id: k.id, name: k.name }))
    selectedFlows.value = flows.value.map(f => f.id)
    selectedKws.value   = keywords.value.map(k => k.id)
  } catch {
    toast.error('Erro ao carregar listas de fluxos e palavras-chave')
  } finally {
    loadingLists.value = false
  }
}

function toggleFlow(id: string) {
  const idx = selectedFlows.value.indexOf(id)
  if (idx >= 0) selectedFlows.value.splice(idx, 1)
  else selectedFlows.value.push(id)
}

function toggleKw(id: string) {
  const idx = selectedKws.value.indexOf(id)
  if (idx >= 0) selectedKws.value.splice(idx, 1)
  else selectedKws.value.push(id)
}

function toggleAllFlows() {
  selectedFlows.value = allFlowsSel.value ? [] : flows.value.map(f => f.id)
}

function toggleAllKws() {
  selectedKws.value = allKwsSel.value ? [] : keywords.value.map(k => k.id)
}

async function handleExport() {
  if (nothingSelected.value) {
    toast.warning('Selecione ao menos um item para exportar')
    return
  }
  loading.value = true
  try {
    const res = await chatbotService.exportChatbot({
      flow_ids:    selectedFlows.value,
      keyword_ids: selectedKws.value,
    })
    const blob = new Blob([res.data], { type: 'application/json' })
    const url  = URL.createObjectURL(blob)
    const a    = document.createElement('a')
    a.href     = url
    a.download = `whatomate-export-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    toast.success(`Exportado: ${selectedFlows.value.length} fluxo(s) e ${selectedKws.value.length} palavra(s)-chave`)
    emit('close')
  } catch {
    toast.error('Erro ao exportar dados')
  } finally {
    loading.value = false
  }
}

// ─── Estado de IMPORT ──────────────────────────────────────────
const file        = ref<File | null>(null)
const importPreview = ref<{ flows: number; keywords: number } | null>(null)

function onFileChange(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  file.value = f
  const reader = new FileReader()
  reader.onload = (ev) => {
    try {
      const parsed = JSON.parse(ev.target?.result as string)
      if (!parsed.version || !Array.isArray(parsed.flows)) {
        throw new Error('invalid')
      }
      importPreview.value = {
        flows:    parsed.flows.length,
        keywords: Array.isArray(parsed.keywords) ? parsed.keywords.length : 0,
      }
    } catch {
      toast.error('Arquivo inválido. Use um JSON exportado pelo Whatomate.')
      file.value        = null
      importPreview.value = null
    }
  }
  reader.readAsText(f)
}

async function handleImport() {
  if (!file.value) return
  loading.value = true
  try {
    const res = await chatbotService.importChatbot(file.value)
    const { flows_imported, keywords_imported } = (res.data as any).data ?? res.data
    toast.success(`Importado: ${flows_imported} fluxo(s) e ${keywords_imported} palavra(s)-chave`)
    emit('imported')
    emit('close')
  } catch (err: any) {
    const msg = err?.response?.data?.message || 'Erro ao importar arquivo'
    toast.error(msg)
  } finally {
    loading.value = false
  }
}

// ─── Reset ao fechar / trocar modo ────────────────────────────
watch(() => props.visible, (v) => {
  if (v && props.mode === 'export') loadLists()
  if (!v) {
    file.value        = null
    importPreview.value = null
  }
})
watch(() => props.mode, (m) => {
  if (m === 'export' && props.visible) loadLists()
})
</script>

<template>
  <Dialog :open="isOpen" @update:open="(v) => { if (!v) emit('close') }">
    <DialogContent class="max-w-lg">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <component
            :is="mode === 'export' ? Download : Upload"
            class="h-5 w-5"
          />
          {{ mode === 'export' ? 'Exportar' : 'Importar' }} Fluxos e Palavras-chave
        </DialogTitle>
      </DialogHeader>

      <!-- ═══ EXPORT ═══ -->
      <template v-if="mode === 'export'">
        <div class="space-y-4 py-2 max-h-[55vh] overflow-y-auto pr-1">
          <p class="text-sm text-muted-foreground">
            Selecione os itens que deseja exportar. O arquivo JSON poderá ser importado em outra instância do Whatomate.
          </p>

          <!-- Flows -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <span class="font-medium text-sm">Fluxos ({{ flows.length }})</span>
              <button
                class="flex items-center gap-1 text-xs text-primary hover:underline"
                @click="toggleAllFlows"
              >
                <component :is="allFlowsSel ? CheckSquare : Square" class="h-3.5 w-3.5" />
                {{ allFlowsSel ? 'Desmarcar todos' : 'Selecionar todos' }}
              </button>
            </div>
            <div v-if="loadingLists" class="text-sm text-muted-foreground italic">Carregando…</div>
            <div v-else-if="!flows.length" class="text-sm text-muted-foreground italic">Nenhum fluxo disponível</div>
            <div v-for="flow in flows" :key="flow.id" class="flex items-center gap-3 py-1.5">
              <Checkbox
                :id="`flow-${flow.id}`"
                :checked="selectedFlows.includes(flow.id)"
                @update:checked="toggleFlow(flow.id)"
              />
              <Label :for="`flow-${flow.id}`" class="cursor-pointer text-sm">{{ flow.name }}</Label>
            </div>
          </div>

          <hr class="border-border" />

          <!-- Keywords -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <span class="font-medium text-sm">Palavras-chave ({{ keywords.length }})</span>
              <button
                class="flex items-center gap-1 text-xs text-primary hover:underline"
                @click="toggleAllKws"
              >
                <component :is="allKwsSel ? CheckSquare : Square" class="h-3.5 w-3.5" />
                {{ allKwsSel ? 'Desmarcar todos' : 'Selecionar todos' }}
              </button>
            </div>
            <div v-if="loadingLists" class="text-sm text-muted-foreground italic">Carregando…</div>
            <div v-else-if="!keywords.length" class="text-sm text-muted-foreground italic">Nenhuma palavra-chave disponível</div>
            <div v-for="kw in keywords" :key="kw.id" class="flex items-center gap-3 py-1.5">
              <Checkbox
                :id="`kw-${kw.id}`"
                :checked="selectedKws.includes(kw.id)"
                @update:checked="toggleKw(kw.id)"
              />
              <Label :for="`kw-${kw.id}`" class="cursor-pointer text-sm">{{ kw.name }}</Label>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="emit('close')">Cancelar</Button>
          <Button :disabled="nothingSelected || loadingLists" :loading="loading" @click="handleExport">
            <Download class="h-4 w-4 mr-2" />
            Baixar JSON
          </Button>
        </DialogFooter>
      </template>

      <!-- ═══ IMPORT ═══ -->
      <template v-else>
        <div class="space-y-4 py-2">
          <p class="text-sm text-muted-foreground">
            Selecione um arquivo <code class="font-mono bg-muted px-1 rounded">.json</code> gerado pela exportação do Whatomate.
            Os itens importados serão adicionados sem substituir os existentes.
          </p>

          <label
            class="flex flex-col items-center justify-center gap-3 border-2 border-dashed border-border rounded-lg p-8 cursor-pointer hover:bg-muted/40 transition-colors"
          >
            <FileJson class="h-10 w-10 text-muted-foreground" />
            <span class="text-sm text-muted-foreground text-center">
              {{ file ? file.name : 'Clique para selecionar o arquivo JSON' }}
            </span>
            <input type="file" accept=".json,application/json" class="hidden" @change="onFileChange" />
          </label>

          <!-- Preview -->
          <div v-if="importPreview" class="rounded-lg border border-border bg-muted/40 p-4 text-sm space-y-2">
            <p class="font-medium">Conteúdo identificado:</p>
            <div class="flex items-center gap-2">
              <span class="text-green-500">✓</span>
              <span><strong>{{ importPreview.flows }}</strong> fluxo(s)</span>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-green-500">✓</span>
              <span><strong>{{ importPreview.keywords }}</strong> palavra(s)-chave</span>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="emit('close')">Cancelar</Button>
          <Button :disabled="!file" :loading="loading" @click="handleImport">
            <Upload class="h-4 w-4 mr-2" />
            Importar
          </Button>
        </DialogFooter>
      </template>

    </DialogContent>
  </Dialog>
</template>
