<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { PageHeader, SearchInput, DataTable, DeleteConfirmDialog, ErrorState, type Column } from '@/components/shared'
import { useTeamsStore } from '@/stores/teams'
import { useAuthStore } from '@/stores/auth'
import { useOrganizationsStore } from '@/stores/organizations'
import { teamsService, type Team } from '@/services/api'
import { toast } from 'vue-sonner'
import { Plus, Pencil, Trash2, Users, RotateCcw, Scale, Hand, Download, Upload } from 'lucide-vue-next'
import { getErrorMessage } from '@/lib/api-utils'
import { formatDate } from '@/lib/utils'
import { ASSIGNMENT_STRATEGIES, getLabelFromValue } from '@/lib/constants'
import { useDebounceFn } from '@vueuse/core'

const { t } = useI18n()

const teamsStore    = useTeamsStore()
const authStore     = useAuthStore()
const organizationsStore = useOrganizationsStore()

const teams      = ref<Team[]>([])
const searchQuery = ref('')
const isLoading  = ref(true)
const fetchError = ref(false)

// Pagination
const currentPage = ref(1)
const totalItems  = ref(0)
const pageSize    = 20

// Delete
const deleteDialogOpen = ref(false)
const teamToDelete     = ref<Team | null>(null)
const isDeletingTeam   = ref(false)

// Sorting
const sortKey       = ref('name')
const sortDirection = ref<'asc' | 'desc'>('asc')

// Export / Import
const isExporting     = ref(false)
const isImporting     = ref(false)
const importFileInput = ref<HTMLInputElement | null>(null)

const canWriteTeams  = computed(() => authStore.hasPermission('teams', 'write'))
const canDeleteTeams = computed(() => authStore.hasPermission('teams', 'delete'))
const breadcrumbs    = computed(() => [{ label: t('nav.settings'), href: '/settings' }, { label: t('nav.teams') }])

const debouncedSearch = useDebounceFn(() => { currentPage.value = 1; fetchTeams() }, 300)
watch(searchQuery, () => debouncedSearch())

const columns = computed<Column<Team>[]>(() => [
  { key: 'team',     label: t('teams.team'),     sortable: true, sortKey: 'name' },
  { key: 'strategy', label: t('teams.strategy'), sortable: true, sortKey: 'assignment_strategy' },
  { key: 'members',  label: t('teams.members'),  sortable: true, sortKey: 'member_count' },
  { key: 'status',   label: t('teams.status'),   sortable: true, sortKey: 'is_active' },
  { key: 'created',  label: t('teams.created'),  sortable: true, sortKey: 'created_at' },
  { key: 'actions',  label: t('common.actions'), align: 'right' },
])

function getStrategyLabel(strategy: string) { return getLabelFromValue(ASSIGNMENT_STRATEGIES, strategy) }
function getStrategyIcon(strategy: string)  { return { round_robin: RotateCcw, load_balanced: Scale, manual: Hand }[strategy] || RotateCcw }
function handlePageChange(page: number)     { currentPage.value = page; fetchTeams() }

function openDeleteDialog(team: Team) {
  teamToDelete.value   = team
  deleteDialogOpen.value = true
}

watch(() => organizationsStore.selectedOrgId, () => fetchTeams())
onMounted(() => fetchTeams())

async function fetchTeams() {
  isLoading.value  = true
  fetchError.value = false
  try {
    const response   = await teamsStore.fetchTeams({ search: searchQuery.value || undefined, page: currentPage.value, limit: pageSize })
    teams.value      = response.teams
    totalItems.value = response.total
  } catch {
    fetchError.value = true
    toast.error(t('common.failedLoad', { resource: t('resources.teams') }))
  } finally {
    isLoading.value = false
  }
}

async function confirmDelete() {
  if (!teamToDelete.value) return
  isDeletingTeam.value = true
  try {
    await teamsStore.deleteTeam(teamToDelete.value.id)
    toast.success(t('common.deletedSuccess', { resource: t('resources.Team') }))
    deleteDialogOpen.value = false
    teamToDelete.value     = null
    await fetchTeams()
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedDelete', { resource: t('resources.team') })))
  } finally {
    isDeletingTeam.value = false
  }
}

// ── Export ────────────────────────────────────────────────────────
async function handleExport() {
  if (isExporting.value) return
  isExporting.value = true
  try {
    const response = await teamsService.exportAll()
    const blob     = new Blob([JSON.stringify(response.data)], { type: 'application/json' })
    const url      = URL.createObjectURL(blob)
    const a        = document.createElement('a')
    const date     = new Date().toISOString().slice(0, 10)
    a.href         = url
    a.download     = `whatomate-teams-${date}.json`
    a.click()
    URL.revokeObjectURL(url)
    toast.success(t('teams.exportSuccess', 'Times exportados com sucesso!'))
  } catch (e) {
    toast.error(getErrorMessage(e, t('teams.exportError', 'Falha ao exportar times')))
  } finally {
    isExporting.value = false
  }
}

// ── Import ────────────────────────────────────────────────────────
function triggerImport() {
  importFileInput.value?.click()
}

async function handleImportFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file  = input.files?.[0]
  if (!file) return

  // Reset input so same file can be re-selected
  input.value = ''

  isImporting.value = true
  try {
    const text   = await file.text()
    const result = await teamsService.importAll(text)
    const data   = result.data as { teams_imported: number; canned_responses_imported: number }
    toast.success(
      t('teams.importSuccess', {
        teams:   data.teams_imported,
        canned:  data.canned_responses_imported,
      },
      `${data.teams_imported} time(s) e ${data.canned_responses_imported} resposta(s) importados!`)
    )
    await fetchTeams()
  } catch (e) {
    toast.error(getErrorMessage(e, t('teams.importError', 'Falha ao importar arquivo. Verifique o formato.')))
  } finally {
    isImporting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      :title="$t('teams.title')"
      :icon="Users"
      icon-gradient="bg-gradient-to-br from-cyan-500 to-blue-600 shadow-cyan-500/20"
      back-link="/settings"
      :breadcrumbs="breadcrumbs"
    >
      <template #actions>
        <!-- Import (hidden file input) -->
        <input
          ref="importFileInput"
          type="file"
          accept=".json,application/json"
          class="hidden"
          @change="handleImportFile"
        />

        <Tooltip v-if="canWriteTeams">
          <TooltipTrigger as-child>
            <Button variant="outline" size="sm" :disabled="isImporting" @click="triggerImport">
              <Upload class="h-4 w-4 mr-2" />
              {{ isImporting ? $t('common.loading', 'Importando...') : $t('teams.import', 'Importar') }}
            </Button>
          </TooltipTrigger>
          <TooltipContent>{{ $t('teams.importTooltip', 'Importar times de um arquivo JSON') }}</TooltipContent>
        </Tooltip>

        <Tooltip v-if="canWriteTeams">
          <TooltipTrigger as-child>
            <Button variant="outline" size="sm" :disabled="isExporting" @click="handleExport">
              <Download class="h-4 w-4 mr-2" />
              {{ isExporting ? $t('common.loading', 'Exportando...') : $t('teams.export', 'Exportar') }}
            </Button>
          </TooltipTrigger>
          <TooltipContent>{{ $t('teams.exportTooltip', 'Exportar todos os times para JSON') }}</TooltipContent>
        </Tooltip>

        <RouterLink v-if="canWriteTeams" to="/settings/teams/new">
          <Button variant="outline" size="sm">
            <Plus class="h-4 w-4 mr-2" />{{ $t('teams.addTeam') }}
          </Button>
        </RouterLink>
      </template>
    </PageHeader>

    <ErrorState
      v-if="fetchError && !isLoading"
      :title="$t('teams.fetchErrorTitle')"
      :description="$t('teams.fetchErrorDescription')"
      class="flex-1"
    >
      <template #action>
        <Button size="sm" @click="fetchTeams">{{ $t('common.retry') }}</Button>
      </template>
    </ErrorState>

    <ScrollArea v-else class="flex-1">
      <div class="p-6">
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <div>
                <CardTitle>{{ $t('teams.yourTeams') }}</CardTitle>
                <CardDescription>{{ $t('teams.yourTeamsDesc') }}</CardDescription>
              </div>
              <SearchInput v-model="searchQuery" :placeholder="$t('teams.searchTeams') + '...'" class="w-64" />
            </div>
          </CardHeader>
          <CardContent>
            <DataTable
              :items="teams"
              :columns="columns"
              :is-loading="isLoading"
              :empty-icon="Users"
              :empty-title="searchQuery ? $t('teams.noMatchingTeams') : $t('teams.noTeamsYet')"
              :empty-description="searchQuery ? $t('teams.noMatchingTeamsDesc') : $t('teams.noTeamsYetDesc')"
              v-model:sort-key="sortKey"
              v-model:sort-direction="sortDirection"
              server-pagination
              :current-page="currentPage"
              :total-items="totalItems"
              :page-size="pageSize"
              item-name="teams"
              @page-change="handlePageChange"
            >
              <template #empty-action>
                <RouterLink v-if="canWriteTeams" to="/settings/teams/new">
                  <Button variant="outline" size="sm"><Plus class="h-4 w-4 mr-2" />{{ $t('teams.addTeam') }}</Button>
                </RouterLink>
              </template>

              <template #cell-team="{ item: team }">
                <RouterLink :to="`/settings/teams/${team.id}`" class="flex items-center gap-3 text-inherit no-underline hover:opacity-80">
                  <div class="h-9 w-9 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                    <Users class="h-4 w-4 text-primary" />
                  </div>
                  <p class="font-medium truncate">{{ team.name }}</p>
                </RouterLink>
              </template>

              <template #cell-strategy="{ item: team }">
                <div class="flex items-center gap-2">
                  <component :is="getStrategyIcon(team.assignment_strategy)" class="h-4 w-4 text-muted-foreground" />
                  <span class="text-sm">{{ getStrategyLabel(team.assignment_strategy) }}</span>
                </div>
              </template>

              <template #cell-members="{ item: team }">
                <Badge variant="outline">{{ team.member_count || 0 }}</Badge>
              </template>

              <template #cell-status="{ item: team }">
                <Badge variant="outline" :class="team.is_active ? 'border-green-600 text-green-600' : ''">
                  {{ team.is_active ? $t('common.active') : $t('common.inactive') }}
                </Badge>
              </template>

              <template #cell-created="{ item: team }">
                <span class="text-muted-foreground">{{ formatDate(team.created_at) }}</span>
              </template>

              <template #cell-actions="{ item: team }">
                <div class="flex items-center justify-end gap-1">
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <RouterLink :to="`/settings/teams/${team.id}`">
                        <Button variant="ghost" size="icon" class="h-8 w-8"><Pencil class="h-4 w-4" /></Button>
                      </RouterLink>
                    </TooltipTrigger>
                    <TooltipContent>{{ $t('teams.editTeamTooltip') }}</TooltipContent>
                  </Tooltip>

                  <Tooltip v-if="canDeleteTeams">
                    <TooltipTrigger as-child>
                      <Button variant="ghost" size="icon" class="h-8 w-8" @click="openDeleteDialog(team)">
                        <Trash2 class="h-4 w-4 text-destructive" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{{ $t('teams.deleteTeamTooltip') }}</TooltipContent>
                  </Tooltip>
                </div>
              </template>
            </DataTable>
          </CardContent>
        </Card>
      </div>
    </ScrollArea>

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="$t('teams.deleteTeam')"
      :item-name="teamToDelete?.name"
      :description="$t('teams.deleteTeamWarning')"
      :is-submitting="isDeletingTeam"
      @confirm="confirmDelete"
    />
  </div>
</template>
