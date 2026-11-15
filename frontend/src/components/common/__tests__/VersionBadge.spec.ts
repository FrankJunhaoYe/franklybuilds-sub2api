import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { UpstreamUpdateInfo } from '@/api/admin/system'
import VersionBadge from '../VersionBadge.vue'

const state = vi.hoisted(() => ({
  auth: { isAdmin: true },
  app: {
    currentVersion: '0.2.21', latestVersion: '0.2.21', hasUpdate: false,
    buildType: 'release', releaseInfo: null, versionWarning: '', versionLoading: false,
    upstreamUpdate: null as UpstreamUpdateInfo | null,
    fetchVersion: vi.fn().mockResolvedValue(null), clearVersionCache: vi.fn()
  }
}))
vi.mock('@/stores', async () => {
  const { reactive } = await import('vue')
  const app = reactive(state.app)
  const auth = reactive(state.auth)
  return { useAppStore: () => app, useAuthStore: () => auth }
})
vi.mock('vue-i18n', async (importOriginal) => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/admin/system', () => ({ performUpdate: vi.fn(), restartService: vi.fn(), getRollbackVersions: vi.fn(), rollback: vi.fn() }))

enableAutoUnmount(afterEach)
beforeEach(() => {
  setActivePinia(createPinia())
  vi.useFakeTimers()
  vi.clearAllMocks()
  state.auth.isAdmin = true
  state.app.upstreamUpdate = null
  state.app.versionWarning = ''
})
afterEach(() => vi.useRealTimers())

function render() {
  return mount(VersionBadge, { global: { stubs: { Icon: true } } })
}

describe('VersionBadge official release tracking', () => {
  it('shows a red dot and official notes without enabling installation of official assets', async () => {
    state.app.upstreamUpdate = { latest_version: '0.2.22', has_update: true, release_info: { name: 'v0.2.22', body: '', published_at: '', html_url: 'https://github.com/Wei-Shaw/sub2api/releases/tag/v0.2.22' } }
    const wrapper = render()
    expect(wrapper.get('[data-testid="update-dot"]').classes()).toContain('bg-red-500')
    await wrapper.get('button').trigger('click')
    expect(wrapper.text()).toContain('version.upstreamSyncHint')
    expect(wrapper.find('a[href="https://github.com/Wei-Shaw/sub2api/releases/tag/v0.2.22"]').exists()).toBe(true)
    expect(wrapper.find('a[href="https://github.com/FrankJunhaoYe/franklybuilds-sub2api"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('version.updateNow')
  })

  it('has no dot when the deployed version includes the latest official release', () => {
    state.app.upstreamUpdate = { latest_version: '0.2.21', has_update: false }
    expect(render().find('[data-testid="update-dot"]').exists()).toBe(false)
  })

  it('polls while mounted and clears the timer when unmounted', () => {
    const wrapper = render()
    expect(state.app.fetchVersion).toHaveBeenCalledTimes(1)
    vi.advanceTimersByTime(20 * 60 * 1000)
    expect(state.app.fetchVersion).toHaveBeenCalledTimes(2)
    wrapper.unmount()
    vi.advanceTimersByTime(20 * 60 * 1000)
    expect(state.app.fetchVersion).toHaveBeenCalledTimes(2)
  })

  it('does not check releases or show administrative notices for ordinary users', () => {
    state.auth.isAdmin = false
    state.app.upstreamUpdate = { latest_version: '0.2.22', has_update: true }
    const wrapper = render()
    expect(state.app.fetchVersion).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="update-dot"]').exists()).toBe(false)
  })
})
