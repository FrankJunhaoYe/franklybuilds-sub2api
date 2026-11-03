<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div v-if="selectedSupplier" class="space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="min-w-0">
              <button class="mb-2 flex items-center gap-1 text-sm text-gray-500 hover:text-primary-600" @click="closeDetail">
                <Icon name="chevronLeft" size="sm" />{{ t('admin.suppliers.back') }}
              </button>
              <div class="flex items-center gap-3">
                <h2 class="truncate text-lg font-medium text-gray-900 dark:text-white">{{ selectedSupplier.name }}</h2>
                <span :class="['badge', selectedSupplier.status === 'active' ? 'badge-success' : 'badge-gray']">{{ t(`admin.suppliers.${selectedSupplier.status}`) }}</span>
              </div>
              <p class="mt-1 break-all font-mono text-xs text-gray-500">{{ selectedSupplier.base_url }}</p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <RouterLink class="btn btn-secondary" :to="{ path: '/admin/accounts', query: { supplier_id: selectedSupplier.id } }">{{ t('admin.suppliers.viewAccounts') }}</RouterLink>
              <button class="btn btn-secondary" @click="openEditor(selectedSupplier)">{{ t('admin.suppliers.edit') }}</button>
              <button class="btn btn-primary" :disabled="!optionsReady" @click="showCreateKey = true"><Icon name="plus" size="sm" class="mr-2" />{{ t('admin.suppliers.addKey') }}</button>
            </div>
          </div>
          <div class="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-gray-200 bg-white px-5 py-4 dark:border-dark-700 dark:bg-dark-800">
            <div>
              <p class="text-xs text-gray-500">{{ t('admin.suppliers.sharedBalance') }}</p>
              <p class="mt-1 text-2xl font-medium tabular-nums text-gray-900 dark:text-white">{{ balanceText(selectedSupplier) }}</p>
              <p class="mt-1 text-xs text-gray-500">{{ t('admin.suppliers.sharedBalanceHint') }}</p>
            </div>
            <div class="flex items-center gap-4">
              <div class="text-right text-xs">
                <p :class="balanceStatusClass(selectedSupplier)">{{ balanceStatus(selectedSupplier) }}</p>
                <p v-if="selectedSupplier.balance_updated_at" class="mt-1 text-gray-500">{{ t('admin.suppliers.lastSuccess', { time: formatDateTime(selectedSupplier.balance_updated_at) }) }}</p>
                <p v-if="selectedSupplier.balance_checked_at" class="mt-1 text-gray-500">{{ t('admin.suppliers.lastChecked', { time: formatDateTime(selectedSupplier.balance_checked_at) }) }}</p>
                <p v-if="selectedSupplier.balance_error" class="mt-1 max-w-sm text-gray-500">{{ balanceErrorText(selectedSupplier.balance_error) }}</p>
              </div>
              <button class="btn btn-secondary" :disabled="refreshing.has(selectedSupplier.id)" @click="refreshBalance(selectedSupplier)">
                <Icon name="refresh" size="sm" class="mr-2" :class="refreshing.has(selectedSupplier.id) ? 'animate-spin' : ''" />{{ t('admin.suppliers.refreshBalance') }}
              </button>
            </div>
          </div>
          <p v-if="selectedSupplier.status === 'inactive'" class="text-sm text-amber-700 dark:text-amber-300">{{ t('admin.suppliers.disabledHint') }}</p>
          <p class="text-xs text-gray-500">{{ t('admin.suppliers.detailHint') }} {{ t('admin.suppliers.billingHint') }}</p>
        </div>
        <div v-else class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-3">
            <SearchInput v-model="search" class="w-64" :placeholder="t('admin.suppliers.search')" @search="searchSuppliers" />
            <Select v-model="status" class="w-40" :options="statusOptions" @change="searchSuppliers" />
          </div>
          <div class="flex items-center gap-2">
            <button class="btn btn-secondary" :disabled="loading" :aria-label="t('common.refresh')" @click="loadSuppliers"><Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" /></button>
            <button class="btn btn-primary" @click="openEditor()"><Icon name="plus" size="sm" class="mr-2" />{{ t('admin.suppliers.create') }}</button>
          </div>
        </div>
      </template>

      <template #table>
        <div v-if="loadError" class="p-6 text-center text-sm text-red-600" role="alert">
          {{ loadError }} <button class="ml-2 underline" @click="selectedSupplier ? loadKeys() : loadSuppliers()">{{ t('common.refresh') }}</button>
        </div>
        <DataTable v-else-if="selectedSupplier" :columns="keyColumns" :data="keys" :loading="keysLoading">
          <template #cell-name="{ row }">
            <div class="font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
            <div class="mt-1 text-xs text-gray-500">{{ row.platform }}</div>
          </template>
          <template #cell-status="{ row }"><span :class="['badge', row.status === 'active' ? 'badge-success' : 'badge-gray']">{{ t(`admin.accounts.status.${row.status}`) }}</span></template>
          <template #cell-groups="{ row }"><div class="flex max-w-xs flex-wrap gap-1"><span v-for="groupID in row.group_ids" :key="groupID" class="badge badge-gray">{{ groups.find(item => item.id === groupID)?.name ?? groupID }}</span><span v-if="!row.group_ids?.length" class="text-gray-400">—</span></div></template>
          <template #cell-rate_multiplier="{ row }"><span class="font-mono">{{ formatMultiplier(row.rate_multiplier ?? 1) }}×</span></template>
          <template #cell-upstream="{ row }"><UpstreamBillingRateCell :account="row" :now="now" :probing="refreshingKeys.has(row.id)" @probe="refreshKey(row)" /></template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-2">
              <button class="btn btn-secondary btn-sm" :disabled="!optionsReady || editingKeyID === row.id" @click="editKey(row)">{{ t('common.edit') }}</button>
              <button class="btn btn-danger btn-sm" @click="pendingAction = { kind: 'key', account: row }">{{ t('common.delete') }}</button>
            </div>
          </template>
          <template #empty><div class="py-12 text-center"><p class="font-medium text-gray-700 dark:text-gray-200">{{ t('admin.suppliers.noKeys') }}</p><p class="mt-2 text-sm text-gray-500">{{ t('admin.suppliers.noKeysHint') }}</p></div></template>
        </DataTable>
        <DataTable v-else :columns="supplierColumns" :data="suppliers" :loading="loading">
          <template #cell-name="{ row }">
            <button class="font-medium text-gray-900 hover:text-primary-600 dark:text-white" @click="openDetail(row.id)">{{ row.name }}</button>
            <p v-if="row.notes" class="mt-1 max-w-xs truncate text-xs text-gray-500" :title="row.notes">{{ row.notes }}</p>
          </template>
          <template #cell-base_url="{ value }"><span class="inline-block max-w-xs truncate font-mono text-xs text-gray-500" :title="value">{{ value }}</span></template>
          <template #cell-account_count="{ row }"><button class="text-primary-600 hover:underline" @click="openDetail(row.id)">{{ t('admin.suppliers.keyCount', { count: row.account_count }) }}</button></template>
          <template #cell-balance="{ row }"><span class="font-medium tabular-nums">{{ balanceText(row) }}</span><p v-if="row.balance_updated_at" class="mt-1 text-xs text-gray-500">{{ formatDateTime(row.balance_updated_at) }}</p></template>
          <template #cell-balance_status="{ row }"><span :class="['text-xs', balanceStatusClass(row)]" :title="balanceErrorText(row.balance_error)">{{ balanceStatus(row) }}</span></template>
          <template #cell-status="{ row }"><span :class="['badge', row.status === 'active' ? 'badge-success' : 'badge-gray']">{{ t(`admin.suppliers.${row.status}`) }}</span></template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button class="btn btn-secondary btn-sm" @click="openDetail(row.id)">{{ t('admin.suppliers.manageKeys') }}</button>
              <button class="btn btn-secondary btn-sm" :aria-label="t('admin.suppliers.refreshBalance')" :title="t('admin.suppliers.refreshBalance')" :disabled="refreshing.has(row.id)" @click="refreshBalance(row)"><Icon name="refresh" size="sm" :class="refreshing.has(row.id) ? 'animate-spin' : ''" /></button>
              <button class="btn btn-secondary btn-sm" :aria-label="t('common.edit')" @click="openEditor(row)"><Icon name="edit" size="sm" /></button>
              <button class="btn btn-secondary btn-sm" :disabled="busy" @click="toggleSupplier(row)">{{ t(`admin.suppliers.${row.status === 'active' ? 'inactive' : 'active'}`) }}</button>
              <button class="btn btn-danger btn-sm" :aria-label="t('common.delete')" @click="pendingAction = { kind: 'supplier', supplier: row }"><Icon name="trash" size="sm" /></button>
            </div>
          </template>
          <template #empty><div class="py-12 text-center"><p class="font-medium text-gray-700 dark:text-gray-200">{{ t(search || status ? 'admin.suppliers.noResults' : 'admin.suppliers.noSuppliers') }}</p><p v-if="!search && !status" class="mt-2 text-sm text-gray-500">{{ t('admin.suppliers.noSuppliersHint') }}</p></div></template>
        </DataTable>
      </template>
      <template #pagination>
        <Pagination v-if="selectedSupplier" :total="keyTotal" :page="keyPage" :page-size="20" :show-page-size-selector="false" @update:page="changeKeyPage" />
        <Pagination v-else :total="total" :page="page" :page-size="20" :show-page-size-selector="false" @update:page="changePage" />
      </template>
    </TablePageLayout>

    <BaseDialog :show="showEditor" :title="t(editingSupplier ? 'admin.suppliers.edit' : 'admin.suppliers.create')" @close="closeEditor">
      <form id="supplier-form" class="space-y-5" @submit.prevent="saveSupplier">
        <div><label for="supplier-name" class="input-label">{{ t('common.name') }}</label><input id="supplier-name" v-model="form.name" required maxlength="100" class="input" autocomplete="off" /></div>
        <div><label for="supplier-url" class="input-label">{{ t('admin.suppliers.url') }}</label><input id="supplier-url" v-model="form.base_url" type="url" required class="input font-mono" placeholder="https://api.example.com" /><p class="input-hint">{{ t('admin.suppliers.urlHint') }}</p></div>
        <div><label for="supplier-notes" class="input-label">{{ t('admin.suppliers.notes') }}</label><textarea id="supplier-notes" v-model="form.notes" rows="2" class="input" /></div>
        <p class="text-xs text-gray-500">{{ t('admin.suppliers.sourceAutoHint') }}</p>
        <details :open="form.source !== 'auto'" class="space-y-4">
          <summary class="cursor-pointer text-sm text-gray-600 dark:text-gray-300">{{ t('admin.suppliers.advancedBalanceSource') }}</summary>
          <div><label for="supplier-balance-source" class="input-label">{{ t('admin.suppliers.balanceSource') }}</label><select id="supplier-balance-source" v-model="form.source" class="input"><option value="auto">{{ t('admin.suppliers.sourceAuto') }}</option><option value="account" :disabled="!editingSupplier">{{ t('admin.suppliers.sourceAccount') }}</option><option value="secret">{{ t('admin.suppliers.sourceSecret') }}</option></select></div>
          <div v-if="form.source === 'account'"><label for="supplier-balance-account" class="input-label">{{ t('admin.suppliers.chooseKey') }}</label><select id="supplier-balance-account" v-model="form.balance_account_id" required class="input" :disabled="editorKeysLoading"><option :value="null">{{ t('admin.suppliers.chooseKey') }}</option><option v-for="account in editorKeys" :key="account.id" :value="account.id">{{ account.name }}</option></select></div>
          <div v-if="form.source === 'secret'"><label for="supplier-balance-key" class="input-label">{{ t('admin.suppliers.queryKey') }}</label><input id="supplier-balance-key" v-model="form.balance_api_key" type="password" class="input" autocomplete="new-password" :required="!editingSupplier?.has_balance_api_key" :placeholder="editingSupplier?.has_balance_api_key ? t('admin.suppliers.secretSaved') : ''" /><p class="input-hint">{{ t('admin.suppliers.queryKeyHint') }}</p></div>
        </details>
        <p v-if="formError" class="text-sm text-red-600" role="alert">{{ formError }}</p>
      </form>
      <template #footer><div class="flex justify-end gap-3"><button class="btn btn-secondary" :disabled="saving" @click="closeEditor">{{ t('common.cancel') }}</button><button form="supplier-form" type="submit" class="btn btn-primary" :disabled="saving || editorKeysLoading">{{ t(saving ? 'common.saving' : 'common.save') }}</button></div></template>
    </BaseDialog>

    <BaseDialog :show="!!pendingAction" :title="confirmationTitle" width="narrow" @close="!busy && (pendingAction = null)">
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ confirmationMessage }}</p>
      <template #footer><div class="flex justify-end gap-3"><button class="btn btn-secondary" :disabled="busy" @click="pendingAction = null">{{ t('common.cancel') }}</button><button class="btn btn-danger" :disabled="busy" @click="confirmAction">{{ t(busy ? 'common.processing' : 'common.confirm') }}</button></div></template>
    </BaseDialog>
    <CreateAccountModal :show="showCreateKey" :proxies="proxies" :groups="groups" :supplier="selectedSupplier" @close="showCreateKey = false" @created="keySaved" />
    <EditAccountModal :show="showEditKey" :account="editingKey" :proxies="proxies" :groups="groups" :supplier="selectedSupplier" @close="showEditKey = false" @updated="keySaved" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import suppliersAPI, { type Supplier, type SupplierInput } from '@/api/admin/suppliers'
import type { Account, AccountListItem, AdminGroup, Proxy } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import { formatMultiplier } from '@/utils/formatters'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import CreateAccountModal from '@/components/account/CreateAccountModal.vue'
import EditAccountModal from '@/components/account/EditAccountModal.vue'
import UpstreamBillingRateCell from '@/components/account/UpstreamBillingRateCell.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const app = useAppStore()
const suppliers = ref<Supplier[]>([])
const selectedSupplier = ref<Supplier | null>(null)
const keys = ref<AccountListItem[]>([])
const groups = ref<AdminGroup[]>([])
const proxies = ref<Proxy[]>([])
const optionsReady = ref(false)
const search = ref('')
const status = ref('')
const page = ref(1)
const total = ref(0)
const keyPage = ref(1)
const keyTotal = ref(0)
const loading = ref(false)
const keysLoading = ref(false)
const loadError = ref('')
const now = ref(Date.now())
const refreshing = reactive(new Set<number>())
const refreshingKeys = reactive(new Set<number>())
const showCreateKey = ref(false)
const showEditKey = ref(false)
const editingKey = ref<Account | null>(null)
const editingKeyID = ref<number | null>(null)
const showEditor = ref(false)
const editingSupplier = ref<Supplier | null>(null)
const editorKeys = ref<AccountListItem[]>([])
const editorKeysLoading = ref(false)
const saving = ref(false)
const formError = ref('')
const form = reactive({ name: '', base_url: '', notes: '', source: 'auto', balance_account_id: null as number | null, balance_api_key: '' })
const pendingAction = ref<{ kind: 'supplier'; supplier: Supplier } | { kind: 'disable'; supplier: Supplier } | { kind: 'key'; account: AccountListItem } | null>(null)
const busy = ref(false)
let listRequest = 0
let keyRequest = 0
let detailRequest = 0

const statusOptions = computed(() => [{ value: '', label: t('admin.suppliers.allStatus') }, ...(['active', 'inactive'] as const).map(value => ({ value, label: t(`admin.suppliers.${value}`) }))])
const supplierColumns = computed(() => [
  { key: 'name', label: t('common.name') }, { key: 'base_url', label: t('admin.suppliers.url') },
  { key: 'account_count', label: t('admin.suppliers.keys') }, { key: 'balance', label: t('admin.suppliers.sharedBalance') },
  { key: 'balance_status', label: t('admin.suppliers.queryStatus') }, { key: 'status', label: t('admin.suppliers.status') }, { key: 'actions', label: t('common.actions') }
])
const keyColumns = computed(() => [
  { key: 'name', label: t('common.name') }, { key: 'status', label: t('common.status') }, { key: 'groups', label: t('admin.suppliers.keyGroups') },
  { key: 'rate_multiplier', label: t('admin.suppliers.accountRate') }, { key: 'upstream', label: t('admin.suppliers.upstreamRate') }, { key: 'actions', label: t('common.actions') }
])
const confirmationTitle = computed(() => t(pendingAction.value?.kind === 'key' ? 'admin.suppliers.deleteKeyTitle' : pendingAction.value?.kind === 'disable' ? 'admin.suppliers.disableTitle' : 'admin.suppliers.deleteTitle'))
const confirmationMessage = computed(() => {
  const action = pendingAction.value
  if (!action) return ''
  if (action.kind === 'key') return t('admin.suppliers.deleteKeyMessage', { name: action.account.name })
  return t(action.kind === 'disable' ? 'admin.suppliers.disableMessage' : 'admin.suppliers.deleteMessage', { name: action.supplier.name, count: action.supplier.account_count })
})

function balanceText(supplier: Supplier) {
  return supplier.balance == null ? t('admin.suppliers.noBalance') : `$${supplier.balance.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 4 })}`
}
function balanceStatus(supplier: Supplier) {
  if (refreshing.has(supplier.id)) return t('admin.suppliers.refreshing')
  const labels = { unknown: 'balanceUnknown', success: 'balanceSuccess', error: 'balanceError', unavailable: 'balanceUnavailable' }
  return t(`admin.suppliers.${labels[supplier.balance_status] ?? 'balanceUnknown'}`)
}
function balanceErrorText(code?: string) {
  if (!code) return ''
  const knownErrors = ['transport_unavailable', 'balance_account_unavailable', 'balance_accounts_unavailable', 'proxy_unavailable', 'balance_key_missing', 'invalid_base_url', 'request_build_failed', 'request_failed', 'empty_response', 'unsupported', 'http_error', 'invalid_response', 'wallet_not_provided']
  return knownErrors.includes(code) ? t(`admin.suppliers.balanceErrors.${code}`) : t('admin.suppliers.operationFailed')
}
function balanceStatusClass(supplier: Supplier) {
  return supplier.balance_status === 'error' || supplier.balance_status === 'unavailable' ? 'text-amber-700 dark:text-amber-300' : 'text-gray-500 dark:text-gray-400'
}
function updateSupplier(supplier: Supplier) {
  suppliers.value = suppliers.value.map(item => item.id === supplier.id ? supplier : item)
  if (selectedSupplier.value?.id === supplier.id) selectedSupplier.value = supplier
}
async function loadSuppliers() {
  const request = ++listRequest
  loading.value = true
  loadError.value = ''
  try {
    const result = await suppliersAPI.list(page.value, 20, { search: search.value, status: status.value })
    if (request !== listRequest) return
    suppliers.value = result.items
    total.value = result.total
  } catch (error) {
    if (request === listRequest) loadError.value = extractApiErrorMessage(error, t('admin.suppliers.loadFailed'))
  } finally { if (request === listRequest) loading.value = false }
}
async function loadKeys() {
  const supplierID = selectedSupplier.value?.id
  if (!supplierID) return
  const request = ++keyRequest
  keysLoading.value = true
  loadError.value = ''
  try {
    const result = await adminAPI.accounts.list(keyPage.value, 20, { supplier_id: supplierID })
    if (request !== keyRequest || selectedSupplier.value?.id !== supplierID) return
    keys.value = result.items
    keyTotal.value = result.total
    now.value = Date.now()
  } catch (error) {
    if (request === keyRequest) loadError.value = extractApiErrorMessage(error, t('admin.suppliers.loadFailed'))
  } finally { if (request === keyRequest) keysLoading.value = false }
}
function searchSuppliers() { page.value = 1; void loadSuppliers() }
function changePage(value: number) { page.value = value; void loadSuppliers() }
function changeKeyPage(value: number) { keyPage.value = value; void loadKeys() }
function openDetail(id: number) { void router.push({ query: { supplier: id } }) }
function closeDetail() { void router.push({ query: {} }) }

async function refreshBalance(supplier: Supplier, notify = true) {
  if (refreshing.has(supplier.id)) return
  refreshing.add(supplier.id)
  try {
    const updated = await suppliersAPI.refreshBalance(supplier.id)
    updateSupplier(updated)
    if (notify) {
      if (updated.balance_status === 'success') app.showSuccess(t('admin.suppliers.refreshed'))
      else app.showWarning(t('admin.suppliers.refreshUnavailable'))
    }
  } catch (error) {
    if (notify) app.showError(extractApiErrorMessage(error, t('admin.suppliers.operationFailed')))
  } finally { refreshing.delete(supplier.id) }
}
async function refreshKey(account: AccountListItem) {
  if (refreshingKeys.has(account.id)) return
  refreshingKeys.add(account.id)
  const supplier = selectedSupplier.value
  try {
    const results = await Promise.allSettled([
      adminAPI.accounts.probeUpstreamBilling(account.id),
      supplier ? refreshBalance(supplier, false) : Promise.resolve()
    ])
    if (results[0].status === 'rejected') app.showWarning(t('admin.suppliers.keySavedProbeFailed'))
    await loadKeys()
  } finally { refreshingKeys.delete(account.id) }
}
async function editKey(account: AccountListItem) {
  editingKeyID.value = account.id
  try { editingKey.value = await adminAPI.accounts.getById(account.id); showEditKey.value = true }
  catch (error) { app.showError(extractApiErrorMessage(error, t('admin.suppliers.operationFailed'))) }
  finally { editingKeyID.value = null }
}
async function keySaved() {
  showCreateKey.value = false
  showEditKey.value = false
  const supplierID = selectedSupplier.value?.id
  await loadKeys()
  if (supplierID) {
    try { updateSupplier(await suppliersAPI.getById(supplierID)) }
    catch (error) { app.showError(extractApiErrorMessage(error, t('admin.suppliers.loadFailed'))) }
  }
  await loadSuppliers()
}
async function openEditor(supplier?: Supplier) {
  editingSupplier.value = supplier ?? null
  Object.assign(form, { name: supplier?.name ?? '', base_url: supplier?.base_url ?? '', notes: supplier?.notes ?? '', source: supplier?.has_balance_api_key ? 'secret' : supplier?.balance_account_id ? 'account' : 'auto', balance_account_id: supplier?.balance_account_id ?? null, balance_api_key: '' })
  formError.value = ''
  editorKeys.value = []
  showEditor.value = true
  if (!supplier) return
  editorKeysLoading.value = true
  try {
    let nextPage = 1
    const items: AccountListItem[] = []
    while (true) {
      const result = await adminAPI.accounts.list(nextPage++, 100, { supplier_id: supplier.id })
      items.push(...result.items)
      if (items.length >= result.total || !result.items.length) break
    }
    if (editingSupplier.value?.id === supplier.id) editorKeys.value = items
  } catch (error) { formError.value = extractApiErrorMessage(error, t('admin.suppliers.loadFailed')) }
  finally { editorKeysLoading.value = false }
}
function closeEditor() {
  if (saving.value) return
  showEditor.value = false
  form.balance_api_key = ''
}
async function saveSupplier() {
  if (saving.value) return
  formError.value = ''
  if (form.source === 'account' && !form.balance_account_id) { formError.value = t('admin.suppliers.selectKeyRequired'); return }
  if (form.source === 'secret' && !form.balance_api_key.trim() && !editingSupplier.value?.has_balance_api_key) { formError.value = t('admin.suppliers.secretRequired'); return }
  const input: SupplierInput = { name: form.name.trim(), base_url: form.base_url.trim(), notes: form.notes.trim() }
  if (form.source === 'account') { input.balance_account_id = form.balance_account_id; input.balance_api_key = '' }
  else if (form.source === 'secret') { input.balance_account_id = null; if (form.balance_api_key.trim()) input.balance_api_key = form.balance_api_key.trim() }
  else { input.balance_account_id = null; input.balance_api_key = '' }
  saving.value = true
  try {
    const supplier = editingSupplier.value ? await suppliersAPI.update(editingSupplier.value.id, input) : await suppliersAPI.create(input)
    updateSupplier(supplier)
    showEditor.value = false
    form.balance_api_key = ''
    app.showSuccess(t('admin.suppliers.saved'))
    await loadSuppliers()
    if (selectedSupplier.value?.id === supplier.id) await loadKeys()
    if (supplier.account_count > 0 || supplier.balance_account_id || supplier.has_balance_api_key) await refreshBalance(supplier, false)
  } catch (error) { formError.value = extractApiErrorMessage(error, t('admin.suppliers.saveFailed')) }
  finally { saving.value = false }
}
async function toggleSupplier(supplier: Supplier) {
  if (supplier.status === 'active') { pendingAction.value = { kind: 'disable', supplier }; return }
  busy.value = true
  try { updateSupplier(await suppliersAPI.update(supplier.id, { status: 'active' })) }
  catch (error) { app.showError(extractApiErrorMessage(error, t('admin.suppliers.operationFailed'))) }
  finally { busy.value = false }
}
async function confirmAction() {
  const action = pendingAction.value
  if (!action || busy.value) return
  busy.value = true
  try {
    if (action.kind === 'disable') updateSupplier(await suppliersAPI.update(action.supplier.id, { status: 'inactive' }))
    else if (action.kind === 'supplier') {
      await suppliersAPI.delete(action.supplier.id)
      if (selectedSupplier.value?.id === action.supplier.id) closeDetail()
      if (suppliers.value.length === 1 && page.value > 1) page.value--
      await loadSuppliers()
      app.showSuccess(t('admin.suppliers.deleted'))
    } else { await adminAPI.accounts.delete(action.account.id); if (keys.value.length === 1 && keyPage.value > 1) keyPage.value--; await keySaved() }
    pendingAction.value = null
  } catch (error) { app.showError(extractApiErrorMessage(error, t('admin.suppliers.operationFailed'))) }
  finally { busy.value = false }
}

watch(() => route.query.supplier, async value => {
  const request = ++detailRequest
  ++keyRequest
  keys.value = []
  keyPage.value = 1
  keyTotal.value = 0
  loadError.value = ''
  const id = Number(value)
  if (!Number.isSafeInteger(id) || id <= 0) { selectedSupplier.value = null; return }
  keysLoading.value = true
  try {
    const supplier = await suppliersAPI.getById(id)
    if (request !== detailRequest) return
    selectedSupplier.value = supplier
    await loadKeys()
  } catch (error) { if (request === detailRequest) loadError.value = extractApiErrorMessage(error, t('admin.suppliers.loadFailed')) }
  finally { if (request === detailRequest) keysLoading.value = false }
}, { immediate: true })

onMounted(async () => {
  void loadSuppliers()
  const results = await Promise.allSettled([adminAPI.groups.getAll(), adminAPI.proxies.getAll()])
  if (results[0].status === 'fulfilled') groups.value = results[0].value
  if (results[1].status === 'fulfilled') proxies.value = results[1].value
  optionsReady.value = results.every(result => result.status === 'fulfilled')
  if (!optionsReady.value) app.showError(t('admin.suppliers.loadFailed'))
})
</script>
