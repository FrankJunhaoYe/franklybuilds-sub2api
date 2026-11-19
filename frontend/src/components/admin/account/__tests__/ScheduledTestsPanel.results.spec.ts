import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, shallowMount } from '@vue/test-utils'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key }),
}))
enableAutoUnmount(afterEach)

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

import ScheduledTestsPanel from '../ScheduledTestsPanel.vue'
const { listByAccount, listResults, create, update, showError } = vi.hoisted(() => ({
  create: vi.fn(), update: vi.fn(), listByAccount: vi.fn(), listResults: vi.fn(), showError: vi.fn(),
}))
vi.mock('@/api/admin', () => ({ adminAPI: { scheduledTests: { listByAccount, listResults, create, update } } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError, showSuccess: vi.fn() }) }))

async function openPanel() {
  const wrapper = shallowMount(ScheduledTestsPanel, {
    props: { show: false, accountId: 1, modelOptions: [] },
    global: { stubs: { BaseDialog: { template: '<div><slot /></div>' } } },
  })
  await wrapper.setProps({ show: true })
  await flushPromises()
  return wrapper
}
const result = (id: number) => ({ id, latency_ms: id, status: 'success', started_at: '2026-10-03T00:00:00Z' })

describe('scheduled test result selection', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    listByAccount.mockResolvedValue([1, 2].map(id => ({ id, model_id: `model-${id}`, enabled: true })))
  })

  it('creates a quality plan using the selected models and renders its evidence', async () => {
    const rule = { action: 'observe', models: ['m1', 'm2'], failures: 3, passes: 2 }
    const wrapper = await openPanel()
    await wrapper.find('button').trigger('click')
    wrapper.getComponent({ name: 'AccountQualityFields' }).vm.$emit('update:modelValue', rule)
    await flushPromises()
    wrapper.findAllComponents({ name: 'Input' }).find(input => input.props('placeholder') === '*/30 * * * *')!.vm.$emit('update:modelValue', '*/30 * * * *')
    await flushPromises()
    const save = wrapper.findAll('button').find(button => button.text().includes('common.save'))!
    listByAccount.mockResolvedValue([{ id: 9, model_id: 'm1', auto_recover: false, cron_expression: '*/30 * * * *', quality_rule: rule, enabled: true }])
    await save.trigger('click')
    await flushPromises()
    expect(create).toHaveBeenCalledWith(expect.objectContaining({ quality_rule: rule, model_id: 'm1', auto_recover: false }))
    listResults.mockResolvedValue([{ ...result(1), quality_evidence: [{ model: 'm2', verdict: 'failed', answer: 'bad', expected: '18', question: 'Q' }] }])
    await wrapper.get('.cursor-pointer').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('bad')
    expect(wrapper.text()).toContain('18')
    await wrapper.get('button[title="admin.scheduledTests.editPlan"]').trigger('click')
    wrapper.getComponent({ name: 'AccountQualityFields' }).vm.$emit('update:modelValue', undefined)
    await flushPromises()
    const editSave = wrapper.findAll('button').find(button => button.text().includes('common.save'))!
    update.mockResolvedValue({ id: 9, model_id: 'm1', enabled: true, auto_recover: false })
    await editSave.trigger('click')
    await flushPromises()
    expect(update).toHaveBeenCalledWith(9, expect.objectContaining({ clear_quality: true }))
  })

  it.each(['resolve', 'reject'] as const)('ignores an old plan request that later %ss', async outcome => {
    const old = deferred<ReturnType<typeof result>[]>()
    listResults.mockReturnValueOnce(old.promise).mockResolvedValueOnce([result(22)])
    const wrapper = await openPanel()
    const headers = wrapper.findAll('.cursor-pointer')
    await headers[0].trigger('click')
    await headers[1].trigger('click')
    await flushPromises()
    const current = wrapper.html()
    if (outcome === 'resolve') old.resolve([result(11)])
    else old.reject(new Error('obsolete error'))
    await flushPromises()
    expect(wrapper.html()).toBe(current)
    expect(showError).not.toHaveBeenCalled()
  })

  it('keeps loading the current plan when the previous request completes', async () => {
    const old = deferred<ReturnType<typeof result>[]>()
    const current = deferred<ReturnType<typeof result>[]>()
    listResults.mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise)
    const wrapper = await openPanel()
    await wrapper.findAll('.cursor-pointer')[0].trigger('click')
    await wrapper.findAll('.cursor-pointer')[1].trigger('click')
    old.resolve([result(11)])
    await flushPromises()
    expect(wrapper.text()).toContain('common.loading')
    current.resolve([result(22)])
    await flushPromises()
    expect(wrapper.text()).not.toContain('common.loading')
  })

  it('invalidates a request when the same plan is collapsed and reopened', async () => {
    const old = deferred<ReturnType<typeof result>[]>()
    listResults.mockReturnValueOnce(old.promise).mockResolvedValueOnce([result(22)])
    const wrapper = await openPanel()
    const header = wrapper.findAll('.cursor-pointer')[0]
    await header.trigger('click')
    await header.trigger('click')
    await header.trigger('click')
    await flushPromises()
    const current = wrapper.html()
    old.resolve([result(11)])
    await flushPromises()
    expect(wrapper.html()).toBe(current)
  })
})
