import { defineComponent, reactive } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  list: vi.fn(), getById: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn(), refreshBalance: vi.fn(),
  listAccounts: vi.fn(), getAccount: vi.fn(), deleteAccount: vi.fn(), probe: vi.fn(), warning: vi.fn(), error: vi.fn(),
  groups: vi.fn(), proxies: vi.fn()
}))

vi.mock('@/api/admin/suppliers', () => ({ default: {
  list: mocks.list, getById: mocks.getById, create: mocks.create, update: mocks.update, delete: mocks.remove, refreshBalance: mocks.refreshBalance
} }))
vi.mock('@/api/admin', () => ({ adminAPI: {
  accounts: { list: mocks.listAccounts, getById: mocks.getAccount, delete: mocks.deleteAccount, probeUpstreamBilling: mocks.probe },
  groups: { getAll: mocks.groups }, proxies: { getAll: mocks.proxies }
} }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showWarning: mocks.warning, showError: mocks.error }) }))
vi.mock('vue-i18n', async () => ({ ...(await vi.importActual<typeof import('vue-i18n')>('vue-i18n')), useI18n: () => ({ t: (key: string, values?: object) => values ? `${key} ${JSON.stringify(values)}` : key }) }))
vi.mock('@/components/account/CreateAccountModal.vue', () => ({ default: { name: 'CreateAccountModal', props: ['show', 'supplier'], template: '<div />' } }))
vi.mock('@/components/account/EditAccountModal.vue', () => ({ default: { name: 'EditAccountModal', props: ['show', 'account', 'supplier'], template: '<div />' } }))

const route = reactive<{ query: Record<string, unknown> }>({ query: {} })
vi.mock('vue-router', async () => ({ ...(await vi.importActual<typeof import('vue-router')>('vue-router')), useRoute: () => route, useRouter: () => ({ push: ({ query }: { query: Record<string, unknown> }) => { route.query = query } }) }))

import SuppliersView from '../SuppliersView.vue'

const supplier = {
  id: 7, name: 'Relay A', base_url: 'https://relay.example/v1', status: 'active', notes: '', account_count: 3,
  balance: 12.5, balance_status: 'success', balance_updated_at: '2026-11-03T01:00:00Z',
  balance_account_id: null, has_balance_api_key: true
}

const Dialog = defineComponent({ props: ['show', 'title'], template: '<div v-if="show" role="dialog"><h2>{{ title }}</h2><slot /><slot name="footer" /></div>' })
const Table = defineComponent({
  props: ['columns', 'data', 'loading'],
  template: '<div><div v-for="row in data" :key="row.id" :data-row="row.id"><div v-for="column in columns" :key="column.key"><slot :name="`cell-${column.key}`" :row="row" :value="row[column.key]">{{ row[column.key] }}</slot></div></div><slot v-if="!data.length && !loading" name="empty" /></div>'
})

function render() {
  return mount(SuppliersView, { global: { stubs: {
    AppLayout: { template: '<main><slot /></main>' },
    TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
    DataTable: Table, BaseDialog: Dialog, Pagination: true, Select: true, SearchInput: true, Icon: true, UpstreamBillingRateCell: true, RouterLink: true
  } } })
}

beforeEach(() => {
  vi.clearAllMocks()
  route.query = {}
  mocks.list.mockResolvedValue({ items: [{ ...supplier }], total: 1, page: 1, page_size: 20, pages: 1 })
  mocks.getById.mockResolvedValue({ ...supplier })
  mocks.listAccounts.mockResolvedValue({ items: [{ id: 8, name: 'Standard', platform: 'openai', type: 'apikey', status: 'active', supplier_id: 7, group_ids: [2] }], total: 1 })
  mocks.groups.mockResolvedValue([{ id: 2, name: 'Local group' }])
  mocks.proxies.mockResolvedValue([])
  mocks.update.mockResolvedValue({ ...supplier })
  mocks.refreshBalance.mockResolvedValue({ ...supplier })
  mocks.remove.mockResolvedValue(undefined)
})

describe('supplier management', () => {
  it('defaults to automatic lookup without requiring a separate balance key', async () => {
    mocks.create.mockResolvedValue({ ...supplier, account_count: 0, has_balance_api_key: false })
    const wrapper = render()
    await flushPromises()
    await wrapper.get('.btn-primary').trigger('click')
    expect(wrapper.get<HTMLSelectElement>('#supplier-balance-source').element.value).toBe('auto')
    expect(wrapper.get('details').attributes('open')).toBeUndefined()
    await wrapper.get('#supplier-name').setValue('Relay B')
    await wrapper.get('#supplier-url').setValue('https://relay-b.example')
    await wrapper.get('#supplier-form').trigger('submit')
    await flushPromises()
    expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ balance_account_id: null, balance_api_key: '' }))
    expect(mocks.refreshBalance).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('refreshes an automatic source after saving a supplier that already has keys', async () => {
    mocks.update.mockResolvedValue({ ...supplier, has_balance_api_key: false })
    const wrapper = render()
    await flushPromises()
    await wrapper.get('[aria-label="common.edit"]').trigger('click')
    await flushPromises()
    await wrapper.get('#supplier-balance-source').setValue('auto')
    await wrapper.get('#supplier-form').trigger('submit')
    await flushPromises()
    expect(mocks.update).toHaveBeenCalledWith(7, expect.objectContaining({ balance_account_id: null, balance_api_key: '' }))
    expect(mocks.refreshBalance).toHaveBeenCalledWith(7)
    wrapper.unmount()
  })
  it('preserves a saved query key when an edit leaves its password field empty', async () => {
    const wrapper = render()
    await flushPromises()
    await wrapper.get('[aria-label="common.edit"]').trigger('click')
    await flushPromises()
    expect(wrapper.get<HTMLInputElement>('#supplier-balance-key').element.value).toBe('')
    await wrapper.get('#supplier-form').trigger('submit')
    await flushPromises()
    expect(mocks.update).toHaveBeenCalledWith(7, expect.not.objectContaining({ balance_api_key: expect.anything() }))
    expect(mocks.update.mock.calls[0][1]).toMatchObject({ balance_account_id: null, name: 'Relay A' })
    wrapper.unmount()
  })

  it('explicitly clears the dedicated key when choosing an existing account', async () => {
    const wrapper = render()
    await flushPromises()
    await wrapper.get('[aria-label="common.edit"]').trigger('click')
    await flushPromises()
    await wrapper.get('#supplier-balance-source').setValue('account')
    await wrapper.get('#supplier-balance-account').setValue('8')
    await wrapper.get('#supplier-form').trigger('submit')
    await flushPromises()
    expect(mocks.update).toHaveBeenCalledWith(7, expect.objectContaining({ balance_account_id: 8, balance_api_key: '' }))
    wrapper.unmount()
  })

  it('retains the last successful balance and time when a refresh is unavailable', async () => {
    mocks.refreshBalance.mockResolvedValue({ ...supplier, balance_status: 'unavailable', balance_error: 'wallet_balance_unavailable' })
    const wrapper = render()
    await flushPromises()
    await wrapper.get('[aria-label="admin.suppliers.refreshBalance"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('$12.50')
    expect(wrapper.text()).toContain('admin.suppliers.balanceUnavailable')
    expect(mocks.warning).toHaveBeenCalled()
    wrapper.unmount()
  })

  it('confirms the supplier name and all key counts before cascading deletion', async () => {
    const wrapper = render()
    await flushPromises()
    await wrapper.get('[aria-label="common.delete"]').trigger('click')
    expect(wrapper.get('[role="dialog"]').text()).toContain('"name":"Relay A","count":3')
    expect(mocks.remove).not.toHaveBeenCalled()
    await wrapper.get('[role="dialog"] .btn-danger').trigger('click')
    await flushPromises()
    expect(mocks.remove).toHaveBeenCalledWith(7)
    wrapper.unmount()
  })

  it('loads only the selected supplier keys and reloads the shared records after a save', async () => {
    route.query = { supplier: '7' }
    const wrapper = render()
    await flushPromises()
    expect(mocks.listAccounts).toHaveBeenCalledWith(1, 20, { supplier_id: 7 })
    expect(wrapper.text()).toContain('Standard')
    expect(wrapper.text()).toContain('Local group')
    expect(wrapper.getComponent({ name: 'RouterLink' }).props('to')).toEqual({ path: '/admin/accounts', query: { supplier_id: 7 } })
    const calls = mocks.listAccounts.mock.calls.length
    wrapper.findComponent({ name: 'CreateAccountModal' }).vm.$emit('created')
    await flushPromises()
    expect(mocks.listAccounts.mock.calls.length).toBeGreaterThan(calls)
    expect(mocks.getById).toHaveBeenCalledTimes(2)
    expect(mocks.probe).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
