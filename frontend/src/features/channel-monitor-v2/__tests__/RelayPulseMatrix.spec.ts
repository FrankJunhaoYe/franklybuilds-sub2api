const i18nT = (key: string, params?: Record<string, unknown>) => {
  const map: Record<string, string> = {
    'channelMonitorV2.matrix.title': '色块矩阵',
    'channelMonitorV2.matrix.description': '趋势色块视图',
    'channelMonitorV2.bucket.minutes': '{count}分钟',
    'channelMonitorV2.bucket.hours': '{count}小时',
    'channelMonitorV2.bucket.days': '{count}天',
    'channelMonitorV2.matrix.wheelZoomX': '滚轮缩放横向',
    'channelMonitorV2.matrix.resetZoom': '重置缩放',
    'channelMonitorV2.matrix.dimension': '维度',
    'channelMonitorV2.metrics.successRate': '成功率',
    'channelMonitorV2.metrics.ttft': '首 Token',
    'channelMonitorV2.metrics.ttftP50': '首 Token P50',
    'channelMonitorV2.metrics.tps': '每秒 Token',
    'channelMonitorV2.metrics.rpm': 'RPM',
    'channelMonitorV2.metrics.cacheRate': '缓存率',
    'channelMonitorV2.matrix.scoreLine': '评分 {score}',
    'channelMonitorV2.metrics.successRateValue': '成功率 {value}',
    'channelMonitorV2.metrics.errorRateValue': '错误率 {value}',
    'channelMonitorV2.metrics.rpmValue': 'RPM {value}',
    'channelMonitorV2.metrics.tpmValue': 'TPM {value}',
    'channelMonitorV2.metrics.tpsValue': '每秒 Token {value}',
    'channelMonitorV2.metrics.ttftValue': '首 Token {value}',
    'channelMonitorV2.metrics.durationValue': '时长 {value}',
    'channelMonitorV2.metrics.cacheRateValue': '缓存率 {value}',
    'channelMonitorV2.matrix.noTrafficAt': '{time} 无流量',
    'channelMonitorV2.matrix.noTraffic': '无流量',
    'channelMonitorV2.matrix.legendAria': '图例',
    'channelMonitorV2.matrix.bad': '差',
    'channelMonitorV2.matrix.good': '好',
    'channelMonitorV2.matrix.healthyLegend': '健康',
    'channelMonitorV2.matrix.warningLegend': '警告',
    'channelMonitorV2.matrix.criticalLegend': '异常',
    'channelMonitorV2.matrix.unknownLegend': '未知',
  }
  const template = map[key] || key
  return template.replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? ''))
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: i18nT,
      te: (key: string) => key.startsWith('channelMonitorV2.'),
      locale: { value: 'zh' },
    }),
  }
})

import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import RelayPulseMatrix from '../RelayPulseMatrix.vue'
import type { MonitorCoverage, MonitorHealth, MonitorMatrixRow, MonitorMetric } from '@/api/channelMonitorV2'

enableAutoUnmount(afterEach)

const health: MonitorHealth = {
  overall: 'warning',
  error_rate: 'warning',
  ttft: 'healthy',
  cache: 'warning',
  score: 52,
  error_rate_score: 40,
  ttft_score: 100,
  cache_score: 50,
  minimum_sample: 20,
}

function metrics(requestCount: number): MonitorMetric {
  return {
    success_requests: requestCount ? requestCount - 1 : 0,
    error_requests: requestCount ? 1 : 0,
    request_count: requestCount,
    token_count: 100,
    rpm: 1.5,
    tpm: 10.2,
    error_rate: requestCount ? 1 / requestCount : 0,
    cache_rate: 0.5,
    cache_rate_numerator: 50,
    cache_rate_denominator: 100,
    ttft: { sample_count: requestCount, p50_ms: 100, p95_ms: 300, avg_ms: 150 },
    duration: { sample_count: requestCount, p50_ms: 500, p95_ms: 900, avg_ms: 600 },
    upstream_affected_requests: 2,
    upstream_attempt_count: 3,
  }
}

function coverage(overrides: Partial<MonitorCoverage> = {}): MonitorCoverage {
  return {
    requested_start: '2026-08-01T00:00:00Z',
    requested_end: '2026-08-01T00:03:00Z',
    coverage_start: '2026-08-01T00:00:00Z',
    data_through: '2026-08-01T00:03:00Z',
    computed_at: '2026-08-01T00:03:00Z',
    aggregation_lag_seconds: 0,
    coverage_complete: true,
    bucket_seconds: 60,
    ...overrides,
  }
}

function row(overrides: Partial<MonitorMatrixRow> = {}): MonitorMatrixRow {
  return {
    platform: 'openai',
    group_id: 7,
    group_name: '默认组',
    model: 'gpt-5',
    metrics: metrics(10),
    health,
    buckets: [
      { bucket_start: '2026-08-01T00:00:00Z', metrics: metrics(10), health },
      { bucket_start: '2026-08-01T00:01:00Z', metrics: metrics(0), health },
    ],
    ...overrides,
  }
}

function mountMatrix(overrides: Partial<{
  rows: MonitorMatrixRow[]
  coverage: MonitorCoverage
  healthMode: 'overall' | 'success' | 'ttft' | 'cache'
  showThroughput: boolean
}> = {}) {
  return mount(RelayPulseMatrix, {
    props: { rows: [row()], coverage: coverage(), healthMode: 'overall', ...overrides },
  })
}

function floatingTooltip() {
  return document.body.querySelector('.matrix-floating-tooltip')
}

describe('RelayPulseMatrix', () => {
  it('shows privacy-safe hover tooltips and multi-band colors without click modal', async () => {
    const wrapper = mountMatrix()

    const cells = wrapper.findAll('.pulse-cell')
    expect(cells).toHaveLength(3)
    await cells[0].trigger('mouseenter')
    // Inspect the displayed tooltip, rather than hidden duplicate cell markup.
    const tip = floatingTooltip()?.textContent || ''
    expect(tip).toContain('成功率')
    expect(tip).toContain('90.0%')
    expect(tip).toContain('首 Token')
    expect(tip).toContain('每秒 Token')
    expect(tip).toContain('缓存率')
    expect(tip).toContain('RPM')
    expect(tip).not.toContain('请求数')
    expect(tip).not.toContain('用户错误')
    expect(tip).not.toContain('上游受影响')
    const summary = wrapper.get('.monitor-card-summary').text()
    expect(summary).toContain('成功率')
    expect(summary).toContain('90.0%')
    expect(summary).toContain('首 Token')
    expect(summary).toContain('每秒 Token')
    expect(summary).toContain('缓存率')
    // Multi-band class from score 52 → score5
    expect(cells[0].classes().some((c) => c.startsWith('health-score'))).toBe(true)
    // Redacted user payloads may have request_count=0 but still include score.
    expect(cells[1].classes().some((c) => c.startsWith('health-score'))).toBe(true)
    expect(cells[2].classes()).toContain('health-unknown')

    // No click-to-open modal
    await cells[0].trigger('click')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })

  it('preserves data with redacted counts and hides throughput across cards and tooltip content', async () => {
    const redactedMetrics: MonitorMetric = {
      ...metrics(0),
      token_count: 0,
      rpm: 0,
      tpm: 0,
      error_rate: 0.125,
      cache_rate_numerator: 0,
      cache_rate_denominator: 0,
      upstream_affected_requests: 0,
      upstream_attempt_count: 0,
    }
    const wrapper = mountMatrix({
      showThroughput: false,
      rows: [row({
        metrics: redactedMetrics,
        buckets: [{ bucket_start: '2026-08-01T00:00:00Z', metrics: redactedMetrics, health }],
      })],
    })

    const cell = wrapper.get('.pulse-cell')
    expect(wrapper.get('.monitor-card-summary').text()).toContain('87.5%')
    expect(cell.classes()).toContain('has-data')
    expect(cell.classes()).toContain('health-score5')
    await cell.trigger('focus')
    for (const content of [wrapper.text(), cell.attributes('aria-label'), cell.attributes('title'), floatingTooltip()?.textContent]) {
      expect(content).not.toMatch(/RPM|TPM|每秒 Token/)
    }
    expect(floatingTooltip()?.textContent).toContain('87.5%')
    expect(floatingTooltip()?.textContent).not.toContain('无流量')

    // The visibility flag also applies when a response still contains throughput.
    await wrapper.setProps({ rows: [row()] })
    await wrapper.get('.pulse-cell').trigger('focus')
    expect(wrapper.text()).not.toContain('每秒 Token')
    expect(floatingTooltip()?.textContent).not.toMatch(/RPM|TPM|每秒 Token/)
  })

  it('updates history health and tooltip score when the selected health metric changes', async () => {
    const wrapper = mountMatrix()
    expect(wrapper.get('.pulse-cell').classes()).toContain('health-score5')

    await wrapper.setProps({ healthMode: 'ttft' })
    const cell = wrapper.get('.pulse-cell')
    expect(cell.classes()).toContain('health-score10')
    await cell.trigger('focus')
    expect(floatingTooltip()?.textContent).toContain('评分 100')

    await wrapper.setProps({
      rows: [row({
        health: { ...health, ttft_score: null },
        buckets: [{ bucket_start: '2026-08-01T00:00:00Z', metrics: metrics(10), health: { ...health, ttft_score: null } }],
      })],
    })
    expect(wrapper.get('.pulse-cell').classes()).toContain('health-unknown')
  })

  it('shows keyboard focus details and dismisses them on blur or Escape', async () => {
    const wrapper = mountMatrix()
    const cell = wrapper.get('.pulse-cell')
    await cell.trigger('focus')
    expect(floatingTooltip()?.textContent).toContain('成功率 90.0%')
    await cell.trigger('blur')
    expect(floatingTooltip()).toBeNull()

    await cell.trigger('focus')
    await cell.trigger('keydown', { key: 'Escape' })
    expect(floatingTooltip()).toBeNull()
  })

  it('keeps tapped bucket details open until another tap or an outside interaction', async () => {
    const wrapper = mountMatrix()
    const cells = wrapper.findAll('.pulse-cell')
    await cells[0].trigger('click')
    await cells[0].trigger('mouseleave')
    expect(floatingTooltip()?.textContent).toContain('成功率 90.0%')

    await cells[2].trigger('click')
    expect(floatingTooltip()?.textContent).toContain('无流量')
    await cells[2].trigger('click')
    expect(floatingTooltip()).toBeNull()

    await cells[0].trigger('click')
    document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    await nextTick()
    expect(floatingTooltip()).toBeNull()
  })

  it('keeps pinned details visible during an internal pointer interaction and closes them on Tab', async () => {
    const wrapper = mount(RelayPulseMatrix, {
      attachTo: document.body,
      props: { rows: [row()], coverage: coverage(), healthMode: 'overall' },
    })
    const cell = wrapper.get<HTMLButtonElement>('.pulse-cell')
    cell.element.focus()
    await cell.trigger('click')
    expect(document.activeElement).toBe(cell.element)

    const tooltip = floatingTooltip()!
    const pointerDown = new Event('pointerdown', { bubbles: true, cancelable: true })
    tooltip.querySelector('.matrix-floating-tooltip-line')!.dispatchEvent(pointerDown)
    // jsdom does not perform the browser's default focus transfer on pointerdown.
    // Clicking tooltip text or its scrollbar can blur the previously focused button.
    cell.element.blur()
    await nextTick()
    expect(floatingTooltip()).toBe(tooltip)
    expect(tooltip.textContent).toContain('成功率 90.0%')
    expect(pointerDown.defaultPrevented).toBe(false)

    const tab = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    document.dispatchEvent(tab)
    await nextTick()
    expect(floatingTooltip()).toBeNull()
    expect(tab.defaultPrevented).toBe(false)
  })
})

describe('RelayPulseMatrix axis range', () => {
  it('uses selected requested range for the X axis even with partial coverage', () => {
    const wrapper = mountMatrix({
      rows: [row({
        buckets: [{ bucket_start: '2026-08-01T00:02:00Z', metrics: metrics(10), health }],
      })],
      coverage: coverage({
        requested_end: '2026-08-01T00:05:00Z',
        coverage_start: '2026-08-01T00:02:00Z',
        coverage_complete: false,
      }),
    })
    // 5 minutes @ 60s buckets → 5 cells spanning the selected range
    expect(wrapper.findAll('.pulse-cell')).toHaveLength(5)
  })

  it.each([
    { range: '90m', bucketSeconds: 300, count: 18 },
    { range: '24h', bucketSeconds: 3600, count: 24 },
    { range: '7d', bucketSeconds: 43200, count: 14 },
    { range: '30d', bucketSeconds: 86400, count: 30 },
  ])('aligns sparse card histories across the full $range range', ({ bucketSeconds, count }) => {
    const start = Date.parse('2026-08-01T00:00:00Z')
    const stamp = (index: number) => new Date(start + index * bucketSeconds * 1000).toISOString()
    const longGroupName = '跨地区共享渠道分组 / '.repeat(8)
    const wrapper = mountMatrix({
      rows: [
        row({
          group_name: longGroupName,
          buckets: [{ bucket_start: stamp(2), metrics: metrics(10), health }],
        }),
        row({
          group_id: 8,
          group_name: '第二分组',
          buckets: [{ bucket_start: stamp(count - 1), metrics: metrics(20), health }],
        }),
      ],
      coverage: coverage({ requested_end: stamp(count), bucket_seconds: bucketSeconds }),
    })

    const cards = wrapper.findAll('.monitor-card')
    expect(cards).toHaveLength(2)
    expect(cards[0].text()).toContain(longGroupName.trim())
    expect(cards[0].text()).toContain('gpt-5')
    for (const card of cards) expect(card.findAll('.pulse-cell')).toHaveLength(count)
    expect(cards[0].findAll('.pulse-cell')[2].classes()).toContain('has-data')
    expect(cards[0].findAll('.pulse-cell')[count - 1].classes()).toContain('is-empty')
    expect(cards[1].findAll('.pulse-cell')[2].classes()).toContain('is-empty')
    expect(cards[1].findAll('.pulse-cell')[count - 1].classes()).toContain('has-data')
  })

  it('zooms card histories together and resets the range when coverage changes', async () => {
    const wrapper = mountMatrix({
      rows: [row(), row({ group_id: 8, group_name: '第二分组' })],
      coverage: coverage({ requested_end: '2026-08-01T00:20:00Z' }),
    })
    await wrapper.get('.pulse-track').trigger('wheel', { deltaY: -100, deltaX: 0 })
    let tracks = wrapper.findAll('.pulse-track')
    const zoomedCount = tracks[0].findAll('.pulse-cell').length
    expect(zoomedCount).toBeLessThan(20)
    expect(tracks[1].findAll('.pulse-cell')).toHaveLength(zoomedCount)

    await wrapper.setProps({ coverage: coverage({ requested_end: '2026-08-01T00:10:00Z' }) })
    tracks = wrapper.findAll('.pulse-track')
    for (const track of tracks) expect(track.findAll('.pulse-cell')).toHaveLength(10)
  })
})
