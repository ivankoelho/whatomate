<script setup lang="ts">
import { ref, computed } from 'vue'
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
import { toast } from '@/components/ui/toast'

const props = defineProps<{
  open: boolean
  flows: Array<{ id: string; name: string }>
  keywords: Array<{ id: string; name: string }>
}>()

const emit = defineEmits<{ (e: 'update:open', v: boolean): void }>()

const selectedFlowIds    = ref<string[]>([])
const selectedKeywordIds = ref<string[]>([])
const loading            = ref(false)

const allFlowsSelected    = computed(() => selectedFlowIds.value.length === props.flows.length)
const allKeywordsSelected = computed(() => selectedKeywordIds.value.length === props.keywords.length)

function toggleAll(type: 'flows' | 'keywords') {
  if (type === 'flows') {
    selectedFlowIds.value = allFlowsSelected.value ? [] : props.flows.map(f => f.id)
  } else {
    selectedKeywordIds.value = allKeywordsSelected.value ? [] : props.keywords.map(k => k.id)
  }
}

function toggleItem(id: string, type: 'flows' | 'keywords') {
  const list = type === 'flows' ? selectedFlowIds : selectedKeywordIds
  const idx  = list.value.indexOf(id)
  if (idx >= 0) list.value.splice(idx, 1)
  else list.value.push(id)
}

async function handleExport() {
  if (!selectedFlowIds.value.length && !selectedKeywordIds.value.length) {
    toast({ title: 'Selecione ao menos um item para exportar', variant: 'destructive' })
    return
  }

  loading.value = true
  try {
    const res = await chatbotService.exportChatbot({
      flow_ids:    selectedFlowIds.value,
      keyword_ids: selectedKeywordIds.value,
    })
    const blob = new Blob([res.data], { type: 'application/json' })
    const url  = URL.createObjectURL(blob)
    const a    = document.createElement('a')
    a.href     = url
    a.download = `whatomate-export-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    toast({ title: 'Exportação concluída com sucesso!' })
    emit('update:open', false)
  } catch {
    toast({ title: 'Erro ao exportar', variant: 'destructive' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-w-lg">
      <DialogHeader>
        <DialogTitle>Exportar Fluxos e Palavras-chave</DialogTitle>
      </DialogHeader>

      <div class="space-y-4 py-2 max-h-[60vh] overflow-y-auto pr-2">
        <!-- Flows -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <span class="font-medium text-sm">Fluxos ({{ flows.length }})</span>
            <button class="text-xs text-primary underline" @click="toggleAll('flows')">
              {{ allFlowsSelected ? 'Desmarcar todos' : 'Selecionar todos' }}
            </button>
          </div>
          <div v-if="!flows.length" class="text-sm text-muted-foreground italic">
            Nenhum fluxo disponível
          </div>
          <div
            v-for="flow in flows"
            :key="flow.id"
            class="flex items-center gap-3 py-1.5"
          >
            <Checkbox
              :id="`flow-${flow.id}`"
              :checked="selectedFlowIds.includes(flow.id)"
              @update:checked="toggleItem(flow.id, 'flows')"
            />
            <Label :for="`flow-${flow.id}`" class="cursor-pointer">{{ flow.name }}</Label>
          </div>
        </div>

        <hr class="border-border" />

        <!-- Keywords -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <span class="font-medium text-sm">Palavras-chave ({{ keywords.length }})</span>
            <button class="text-xs text-primary underline" @click="toggleAll('keywords')">
              {{ allKeywordsSelected ? 'Desmarcar todos' : 'Selecionar todos' }}
            </button>
          </div>
          <div v-if="!keywords.length" class="text-sm text-muted-foreground italic">
            Nenhuma palavra-chave disponível
          </div>
          <div
            v-for="kw in keywords"
            :key="kw.id"
            class="flex items-center gap-3 py-1.5"
          >
            <Checkbox
              :id="`kw-${kw.id}`"
              :checked="selectedKeywordIds.includes(kw.id)"
              @update:checked="toggleItem(kw.id, 'keywords')"
            />
            <Label :for="`kw-${kw.id}`" class="cursor-pointer">{{ kw.name }}</Label>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)">Cancelar</Button>
        <Button :loading="loading" @click="handleExport">
          Baixar JSON
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
