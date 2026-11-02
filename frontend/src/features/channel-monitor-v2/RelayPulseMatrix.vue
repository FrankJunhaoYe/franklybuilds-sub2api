<template>
  <section class="monitor-cards-section min-w-0" :aria-label="t('channelMonitorV2.cards.title')">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <h2 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
          <Icon name="grid" size="sm" class="text-primary-500" />
          {{ t('channelMonitorV2.cards.title') }}
        </h2>
        <p class="mt-1 text-xs leading-relaxed text-gray-500 dark:text-dark-400">
          {{ t('channelMonitorV2.cards.description') }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
        <span class="badge badge-gray">{{ bucketLabel }}</span>
        <button
          type="button"
          class="btn btn-secondary btn-sm !min-h-8 !px-2.5 !py-1 text-xs"
          :disabled="!zoomed"
          @click="resetMatrixZoom"
        >
          {{ t('channelMonitorV2.matrix.resetZoom') }}
        </button>
      </div>
    </div>

    <div v-if="rows.length" class="monitor-card-grid">
      <article
        v-for="entry in alignedRows"
        :key="rowKey(entry.row)"
        class="monitor-card card flex min-w-0 flex-col !rounded-2xl !p-5"
        :aria-label="rowLabel(entry.row)"
      >
        <div class="flex min-w-0 items-start gap-2.5">
          <span
            class="grid h-9 w-9 shrink-0 place-items-center rounded-xl"
            :class="platformBadgeLightClass(entry.row.platform)"
          >
            <ProviderIcon :provider="entry.row.platform" :size="20" />
          </span>
          <div class="min-w-0 flex-1">
            <h3 class="monitor-card-name text-base font-semibold leading-snug text-gray-900 dark:text-gray-100">
              {{ cardTitle(entry.row) }}
            </h3>
            <div class="mt-1 flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-1">
              <span
                v-if="entry.row.group_id || entry.row.group_name || entry.row.model"
                class="monitor-card-name rounded-md px-1.5 py-0.5 text-[10px] font-medium"
                :class="platformBadgeLightClass(entry.row.platform)"
              >{{ platformLabel(entry.row.platform) }}</span>
              <span
                v-if="(entry.row.group_id || entry.row.group_name) && entry.row.model"
                class="monitor-card-name min-w-0 text-xs text-gray-500 dark:text-gray-400"
              >{{ modelLabel(entry.row.model) }}</span>
            </div>
          </div>
          <span
            class="monitor-health-badge inline-flex max-w-[42%] shrink-0 items-center gap-1.5 rounded-full px-2 py-1 text-[11px] font-medium leading-tight"
            :class="badgeClass(entry.row.health, entry.row.metrics.request_count)"
            :title="t('channelMonitorV2.matrix.scoreLine', { score: formatScore(entry.row.health) })"
          >
            <i :class="['status-dot', cellClass(entry.row.health, entry.row.metrics.request_count)]" aria-hidden="true"></i>
            <span class="monitor-card-name">{{ t(`channelMonitorV2.cards.${healthState(entry.row.health, entry.row.metrics.request_count)}`) }}</span>
          </span>
        </div>

        <div class="monitor-card-summary mt-5 flex flex-1 flex-col">
          <dl class="grid grid-cols-2 gap-2">
            <div class="min-w-0 rounded-xl bg-gray-50 px-3 py-2.5 dark:bg-dark-900/50" :title="latencyPrivacy(entry.row.metrics.ttft)">
              <dt class="monitor-card-name text-[11px] text-gray-500 dark:text-gray-400">{{ t('channelMonitorV2.metrics.ttftP50') }}</dt>
              <dd class="mt-1 text-lg font-semibold tabular-nums text-gray-900 dark:text-gray-100">{{ formatMs(entry.row.metrics.ttft.p50_ms) }}</dd>
            </div>
            <div class="min-w-0 rounded-xl bg-gray-50 px-3 py-2.5 dark:bg-dark-900/50">
              <dt class="monitor-card-name text-[11px] text-gray-500 dark:text-gray-400">{{ t('channelMonitorV2.metrics.cacheRate') }}</dt>
              <dd class="mt-1 text-lg font-semibold tabular-nums text-gray-900 dark:text-gray-100">{{ formatPercent(entry.row.metrics.cache_rate) }}</dd>
            </div>
          </dl>
          <dl v-if="showThroughput" class="monitor-card-throughput mt-3 grid grid-cols-2 gap-3 px-1">
            <div class="min-w-0" :title="exactTps(entry.row.metrics.tpm)">
              <dt class="monitor-card-name text-[11px] text-gray-500 dark:text-gray-400">{{ t('channelMonitorV2.metrics.tps') }}</dt>
              <dd class="mt-0.5 text-sm font-medium tabular-nums text-gray-700 dark:text-gray-200">{{ formatTps(entry.row.metrics.tpm) }}</dd>
            </div>
            <div class="min-w-0">
              <dt class="monitor-card-name text-[11px] text-gray-500 dark:text-gray-400">{{ t('channelMonitorV2.metrics.rpm') }}</dt>
              <dd class="mt-0.5 text-sm font-medium tabular-nums text-gray-700 dark:text-gray-200">{{ formatRate(entry.row.metrics.rpm) }}</dd>
            </div>
          </dl>
          <div class="mt-auto pt-4">
            <div class="flex flex-wrap items-end justify-between gap-2 border-t border-gray-100 pt-4 dark:border-dark-700">
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('channelMonitorV2.metrics.successRate') }}</span>
              <strong class="monitor-success-rate text-[26px] font-semibold leading-none tracking-tight tabular-nums text-gray-900 dark:text-gray-100">
                {{ successRate(entry.row.metrics) }}
              </strong>
            </div>
          </div>
        </div>

        <div class="mt-4 min-w-0">
          <div
            class="pulse-track grid min-w-0 items-stretch"
            :style="pulseStyle"
            role="group"
            :aria-label="t('channelMonitorV2.cards.history')"
            @wheel="onMatrixWheel"
          >
            <button
              v-for="slot in entry.slots"
              :key="slot.start"
              type="button"
              class="pulse-cell relative min-w-0 rounded-sm border-0 p-0 outline-offset-1"
              :class="[
                slot.bucket ? cellClass(slot.bucket.health, slot.bucket.metrics.request_count) : 'health-unknown',
                slot.bucket ? 'has-data' : 'is-empty',
              ]"
              :title="slotTooltip(slot)"
              :aria-label="slotTooltip(slot)"
              :aria-describedby="isTooltipTarget(entry.row, slot) ? tooltipId : undefined"
              @mouseenter="showTooltip($event, entry.row, slot)"
              @mouseleave="hideTransientTooltip"
              @focus="showTooltip($event, entry.row, slot)"
              @blur="onAnchorBlur"
              @keydown.esc.stop="hideTooltip"
              @click="toggleTooltip($event, entry.row, slot)"
            ></button>
          </div>
          <div class="pulse-axis mt-2 flex min-w-0 justify-between gap-3 text-[10px] leading-snug text-gray-400 dark:text-dark-400">
            <span class="min-w-0 break-words">{{ axisStart }}</span>
            <span class="min-w-0 break-words text-right">{{ axisEnd }}</span>
          </div>
        </div>
      </article>
    </div>
    <div v-else class="card flex min-h-[200px] items-center justify-center !rounded-2xl !p-6">
      <EmptyState :title="t('channelMonitorV2.cards.emptyTitle')" :description="t('channelMonitorV2.empty.description')" />
    </div>

    <div class="mt-4 flex flex-wrap items-center justify-between gap-3 text-[11px] text-gray-500 dark:text-gray-400">
      <div class="flex flex-wrap gap-x-4 gap-y-2" :aria-label="t('channelMonitorV2.matrix.legendAria')">
        <span class="inline-flex items-center gap-1.5"><i class="status-dot health-score10" aria-hidden="true"></i>{{ t('channelMonitorV2.matrix.healthyLegend') }}</span>
        <span class="inline-flex items-center gap-1.5"><i class="status-dot health-score6" aria-hidden="true"></i>{{ t('channelMonitorV2.matrix.warningLegend') }}</span>
        <span class="inline-flex items-center gap-1.5"><i class="status-dot health-score2" aria-hidden="true"></i>{{ t('channelMonitorV2.matrix.criticalLegend') }}</span>
        <span class="inline-flex items-center gap-1.5"><i class="status-dot health-unknown" aria-hidden="true"></i>{{ t('channelMonitorV2.matrix.unknownLegend') }}</span>
      </div>
      <span>{{ t('channelMonitorV2.cards.zoomHint') }}</span>
    </div>

    <Teleport to="body">
      <div
        v-if="floatingTooltip.visible"
        :id="tooltipId"
        ref="tooltipRef"
        class="matrix-floating-tooltip rounded-xl border border-gray-200 bg-white p-3 text-gray-600 shadow-lg dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300"
        :class="floatingTooltip.pinned ? 'pointer-events-auto' : 'pointer-events-none'"
        :style="{ left: `${floatingTooltip.x}px`, top: `${floatingTooltip.y}px` }"
        role="tooltip"
      >
        <span
          v-for="(line, index) in floatingTooltip.lines"
          :key="`${index}:${line}`"
          class="matrix-floating-tooltip-line"
          :class="index === 0 ? 'mb-1 font-semibold text-gray-900 dark:text-gray-100' : ''"
        >{{ line }}</span>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, useId, watch } from 'vue'
import type {
  LatencyMetric,
  MonitorCoverage,
  MonitorHealth,
  MonitorMatrixBucket,
  MonitorMatrixRow,
  MonitorMetric,
} from '@/api/channelMonitorV2'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import ProviderIcon from '@/components/user/monitor/ProviderIcon.vue'
import { platformBadgeLightClass, platformLabel } from '@/utils/platformColors'
import {
  formatLatencyPrivacy,
  formatMonitorMs,
  formatMonitorPercent,
  formatMonitorSuccessRateFromError,
  formatMonitorThroughput,
  formatMonitorTokensPerSecond,
  tokensPerSecondFromTpm,
  healthModeScore,
  healthScoreClass,
} from '@/features/channel-monitor-v2/monitorFormat'
import {
  applyWheelZoom,
  clientXRatio,
  isZoomed,
  resetZoom,
  sliceByZoom,
  type ZoomState,
} from '@/features/channel-monitor-v2/monitorZoom'

type HealthMode = 'overall' | 'success' | 'ttft' | 'cache'
const { t, locale } = useI18n()

const props = withDefaults(
  defineProps<{
    rows: MonitorMatrixRow[]
    coverage: MonitorCoverage
    healthMode: HealthMode
    /** When false, RPM/TPM are omitted from tooltips (user scale privacy). */
    showThroughput?: boolean
  }>(),
  { showThroughput: true },
)

type AlignedSlot = { start: string; bucket?: MonitorMatrixBucket }

const floatingTooltip = reactive({
  visible: false,
  pinned: false,
  key: '',
  x: 0,
  y: 0,
  lines: [] as string[],
})

const tooltipRef = ref<HTMLElement | null>(null)
const tooltipId = useId()
let tooltipAnchor: HTMLElement | null = null
let pointerInsideTooltip = false
const zoom = ref<ZoomState>(resetZoom())
const zoomed = computed(() => isZoomed(zoom.value))

const allBucketStarts = computed(() => {
  // X-axis always spans the UI-selected range [requested_start, requested_end).
  // Partial backfill leaves empty cells until coverage_start/data_through fill in.
  const step = Math.max(60, props.coverage.bucket_seconds) * 1000
  const requestedStart = new Date(props.coverage.requested_start).getTime()
  const requestedEndRaw = props.coverage.requested_end
    ? new Date(props.coverage.requested_end).getTime()
    : NaN
  // Fallback for older payloads without requested_end.
  const dataThrough = new Date(props.coverage.data_through).getTime()
  const end = Number.isFinite(requestedEndRaw) && requestedEndRaw > requestedStart
    ? requestedEndRaw
    : dataThrough
  if (![requestedStart, end].every(Number.isFinite) || requestedStart >= end) return []
  const starts: string[] = []
  for (let cursor = Math.floor(requestedStart / step) * step; cursor < end; cursor += step) {
    starts.push(new Date(cursor).toISOString())
  }
  return starts
})
/** Visible bucket window after X zoom (cursor-centered), not always the tail. */
const bucketStarts = computed(() => sliceByZoom(allBucketStarts.value, zoom.value))
// Fewer visible buckets make each block wider without overflowing its card.
const pulseStyle = computed(() => ({
  gridTemplateColumns: `repeat(${Math.max(1, bucketStarts.value.length)}, minmax(0, 1fr))`,
  gap: bucketStarts.value.length > 24 ? '2px' : '3px',
}))
const axisStart = computed(() =>
  bucketStarts.value.length ? formatAxisTime(bucketStarts.value[0]) : '—'
)
const axisEnd = computed(() =>
  bucketStarts.value.length ? formatAxisTime(bucketStarts.value[bucketStarts.value.length - 1]) : ''
)
const bucketLabel = computed(() => {
  const minutes = props.coverage.bucket_seconds / 60
  if (minutes < 60) return t('channelMonitorV2.bucket.minutes', { count: minutes })
  const hours = minutes / 60
  if (hours < 24) return t('channelMonitorV2.bucket.hours', { count: hours })
  return t('channelMonitorV2.bucket.days', { count: hours / 24 })
})

/** Shared ISO start → column index for the visible window (rebuilt when zoom/coverage changes). */
const bucketStartIndex = computed(() => {
  const map = new Map<string, number>()
  bucketStarts.value.forEach((start, index) => map.set(start, index))
  return map
})

/** Pre-aligned sparse slots per row so wheel zoom does not rebuild Maps every paint. */
const alignedRows = computed(() => {
  const starts = bucketStarts.value
  const indexByStart = bucketStartIndex.value
  return props.rows.map((row) => {
    const slots: AlignedSlot[] = starts.map((start) => ({ start }))
    for (const bucket of row.buckets || []) {
      const key = new Date(bucket.bucket_start).toISOString()
      const index = indexByStart.get(key)
      if (index != null) slots[index] = { start: starts[index], bucket }
    }
    return { row, slots }
  })
})

function onMatrixWheel(event: WheelEvent) {
  const track = event.currentTarget as HTMLElement | null
  const next = applyWheelZoom(zoom.value, event, clientXRatio(event.clientX, track))
  if (next.span === zoom.value.span && next.offset === zoom.value.offset) return
  // Only the history strip captures zoom/pan. The rest of the card scrolls normally.
  event.preventDefault()
  hideTooltip()
  zoom.value = next
}

function resetMatrixZoom() {
  hideTooltip()
  zoom.value = resetZoom()
}

watch(
  () => [
    props.coverage.requested_start,
    props.coverage.requested_end,
    props.coverage.coverage_start,
    props.coverage.data_through,
    props.coverage.bucket_seconds,
  ],
  () => {
    resetMatrixZoom()
  },
)

function cellClass(health: MonitorHealth, requestCount: number): string {
  return healthScoreClass(health, props.healthMode, requestCount)
}

function healthState(health: MonitorHealth, requestCount: number): 'healthy' | 'warning' | 'critical' | 'unknown' {
  if (cellClass(health, requestCount) === 'health-unknown') return 'unknown'
  const score = healthModeScore(health, props.healthMode)
  if (score != null) return score >= 80 ? 'healthy' : score >= 50 ? 'warning' : 'critical'
  if (props.healthMode === 'success') return health.error_rate
  if (props.healthMode === 'ttft') return health.ttft
  if (props.healthMode === 'cache') return health.cache || 'unknown'
  return health.overall
}

function badgeClass(health: MonitorHealth, requestCount: number): string {
  switch (healthState(health, requestCount)) {
    case 'healthy': return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
    case 'warning': return 'bg-amber-50 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
    case 'critical': return 'bg-red-50 text-red-700 dark:bg-red-500/15 dark:text-red-300'
    default: return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
  }
}

function modelLabel(model: string): string {
  return model === '__other__' ? t('channelMonitorV2.otherModels') : model
}

function cardTitle(row: MonitorMatrixRow): string {
  if (row.group_name || row.group_id) return row.group_name || `#${row.group_id}`
  return row.model ? modelLabel(row.model) : platformLabel(row.platform)
}

function rowLabel(row: MonitorMatrixRow): string {
  const parts = [row.platform]
  if (row.group_name || row.group_id) parts.push(row.group_name || `#${row.group_id}`)
  if (row.model) parts.push(modelLabel(row.model))
  return parts.join(' / ')
}

function rowKey(row: MonitorMatrixRow): string {
  return [row.platform, row.group_id || 0, row.model || ''].join(':')
}

function successRate(metrics: MonitorMetric): string {
  // Empty traffic: no request count and no throughput signal.
  // When throughput is hidden for privacy, still show success from error_rate.
  const noCount = metrics.request_count <= 0
  const noTP = (metrics.rpm || 0) <= 0 && (metrics.tpm || 0) <= 0
  if (noCount && noTP && props.showThroughput) return '-'
  return formatMonitorSuccessRateFromError(metrics.error_rate)
}

function formatScore(health: MonitorHealth): string {
  const score = healthModeScore(health, props.healthMode)
  if (score == null) return '—'
  return `${Math.round(score)}`
}

function bucketTooltip(bucket: MonitorMatrixBucket): string {
  return bucketTooltipLines(bucket).join('\n')
}

function bucketTooltipLines(bucket: MonitorMatrixBucket): string[] {
  const metrics = bucket.metrics
  const lines = [
    formatBucketRange(bucket.bucket_start),
    t('channelMonitorV2.matrix.scoreLine', { score: formatScore(bucket.health) }),
    t('channelMonitorV2.metrics.successRateValue', { value: successRate(metrics) }),
    t('channelMonitorV2.metrics.ttftValue', { value: latencyPrivacy(metrics.ttft) }),
  ]
  if (props.showThroughput) {
    lines.push(t('channelMonitorV2.metrics.tpsValue', { value: formatTps(metrics.tpm) }))
  }
  lines.push(
    t('channelMonitorV2.metrics.cacheRateValue', { value: formatPercent(metrics.cache_rate) }),
    t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(metrics.error_rate) }),
  )
  if (props.showThroughput) {
    lines.push(t('channelMonitorV2.metrics.rpmValue', { value: formatRate(metrics.rpm) }))
  }
  lines.push(t('channelMonitorV2.metrics.durationValue', { value: latencyPrivacy(metrics.duration) }))
  return lines
}
function emptyTooltipLines(start: string): string[] {
  return [formatBucketRange(start), t('channelMonitorV2.matrix.noTraffic')]
}

function slotTooltip(slot: AlignedSlot): string {
  return slot.bucket ? bucketTooltip(slot.bucket) : emptyTooltipLines(slot.start).join('\n')
}

function tooltipKey(row: MonitorMatrixRow, slot: AlignedSlot): string {
  return `${rowKey(row)}:${slot.start}`
}

function isTooltipTarget(row: MonitorMatrixRow, slot: AlignedSlot): boolean {
  return floatingTooltip.visible && floatingTooltip.key === tooltipKey(row, slot)
}

function showTooltip(event: MouseEvent | FocusEvent, row: MonitorMatrixRow, slot: AlignedSlot) {
  if (floatingTooltip.pinned) return
  openTooltip(event, row, slot, false)
}

function toggleTooltip(event: MouseEvent, row: MonitorMatrixRow, slot: AlignedSlot) {
  if (floatingTooltip.pinned && isTooltipTarget(row, slot)) {
    hideTooltip()
    return
  }
  openTooltip(event, row, slot, true)
}

function openTooltip(event: MouseEvent | FocusEvent, row: MonitorMatrixRow, slot: AlignedSlot, pinned: boolean) {
  tooltipAnchor = event.currentTarget as HTMLElement | null
  floatingTooltip.lines = slot.bucket ? bucketTooltipLines(slot.bucket) : emptyTooltipLines(slot.start)
  floatingTooltip.key = tooltipKey(row, slot)
  floatingTooltip.pinned = pinned
  floatingTooltip.visible = true
  void nextTick(positionTooltip)
}

function hideTransientTooltip() {
  if (!floatingTooltip.pinned) hideTooltip()
}

function hideTooltip() {
  floatingTooltip.visible = false
  floatingTooltip.pinned = false
  floatingTooltip.key = ''
  tooltipAnchor = null
  pointerInsideTooltip = false
}

function onAnchorBlur(event: FocusEvent) {
  const nextTarget = event.relatedTarget
  const focusInsideTooltip = nextTarget instanceof Node && tooltipRef.value?.contains(nextTarget)
  // A pinned tooltip can receive a pointer interaction without a focusable target
  // (text selection, touch scrolling or its scrollbar). Let that blur complete.
  const keepPinned = floatingTooltip.pinned && (pointerInsideTooltip || focusInsideTooltip)
  pointerInsideTooltip = false
  if (!keepPinned) hideTooltip()
}

function positionTooltip() {
  if (!floatingTooltip.visible || !tooltipAnchor || !tooltipRef.value) return
  const rect = tooltipAnchor.getBoundingClientRect()
  const tooltip = tooltipRef.value.getBoundingClientRect()
  const margin = 12
  const x = rect.left + rect.width / 2 - tooltip.width / 2
  const above = rect.top - tooltip.height - 8
  const y = above >= margin ? above : rect.bottom + 8
  floatingTooltip.x = Math.max(margin, Math.min(x, window.innerWidth - tooltip.width - margin))
  floatingTooltip.y = Math.max(margin, Math.min(y, window.innerHeight - tooltip.height - margin))
}

function onOutsidePointerDown(event: PointerEvent) {
  const target = event.target as Node | null
  pointerInsideTooltip = Boolean(target && tooltipRef.value?.contains(target))
  if (target && !tooltipAnchor?.contains(target) && !pointerInsideTooltip) hideTooltip()
}

function onKeyDown(event: KeyboardEvent) {
  pointerInsideTooltip = false
  if (event.key === 'Escape' || event.key === 'Tab') hideTooltip()
}

function onViewportChange(event: Event) {
  if (event.target instanceof Node && tooltipRef.value?.contains(event.target)) return
  hideTooltip()
}

// Refreshes and display changes must not leave an old bucket's tooltip on screen.
watch(() => [props.rows, props.healthMode, props.showThroughput, locale.value], hideTooltip)
onMounted(() => {
  document.addEventListener('pointerdown', onOutsidePointerDown)
  document.addEventListener('keydown', onKeyDown)
  window.addEventListener('resize', onViewportChange)
  window.addEventListener('scroll', onViewportChange, true)
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onOutsidePointerDown)
  document.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('resize', onViewportChange)
  window.removeEventListener('scroll', onViewportChange, true)
})

function latencyPrivacy(metric: LatencyMetric) {
  return formatLatencyPrivacy(metric.p50_ms, metric.p90_ms, metric.avg_ms, metric.p95_ms)
}

function formatPercent(value: number) {
  return formatMonitorPercent(value)
}

function formatRate(value: number) {
  return formatMonitorThroughput(value)
}

function formatTps(tpm: number | null | undefined) {
  return formatMonitorTokensPerSecond(tpm)
}

function exactTps(tpm: number | null | undefined) {
  const tps = tokensPerSecondFromTpm(tpm)
  return Intl.NumberFormat(locale.value || undefined, { maximumFractionDigits: 3 }).format(tps)
}

function formatMs(value: number | null) {
  return formatMonitorMs(value)
}

function formatAxisTime(value: string) {
  return new Intl.DateTimeFormat(locale.value || undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function formatBucketRange(value: string) {
  const start = new Date(value)
  const end = new Date(start.getTime() + props.coverage.bucket_seconds * 1000)
  return `${formatAxisTime(start.toISOString())} - ${new Intl.DateTimeFormat(locale.value || undefined, { hour: '2-digit', minute: '2-digit' }).format(end)}`
}
</script>

<style scoped>
.monitor-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 18rem), 1fr));
  gap: 1.25rem;
}
.monitor-card {
  min-height: 280px;
}
.monitor-card-name {
  overflow-wrap: anywhere;
}
.pulse-track {
  min-height: 28px;
}
.status-dot {
  display: inline-block;
  height: 0.375rem;
  width: 0.375rem;
  flex: none;
  border-radius: 9999px;
}

/* Preserve the score bands used by the passive monitor. */
.health-score10 { background: #16a34a; }
.health-score9  { background: #22c55e; }
.health-score8  { background: #4ade80; }
.health-score7  { background: #a3e635; }
.health-score6  { background: #facc15; }
.health-score5  { background: #fbbf24; }
.health-score4  { background: #f59e0b; }
.health-score3  { background: #f97316; }
.health-score2  { background: #fb7185; }
.health-score1  { background: #f87171; }
.health-score0  { background: rgb(239, 67, 67); }
.health-healthy  { background: #22c55e; }
.health-warning  { background: #f59e0b; }
.health-critical { background: #ef4444; }
.health-unknown  { background: #9ca3af; }

.pulse-cell {
  cursor: pointer;
  touch-action: manipulation;
}
.pulse-cell.is-empty {
  opacity: 0.55;
}
.pulse-cell:hover,
.pulse-cell:focus-visible {
  outline: 2px solid currentColor;
  outline-offset: 2px;
  z-index: 1;
}
.matrix-floating-tooltip {
  position: fixed;
  z-index: 9999;
  width: max-content;
  max-width: min(20rem, calc(100vw - 24px));
  max-height: calc(100dvh - 24px);
  overflow-y: auto;
  overflow-wrap: anywhere;
  white-space: normal;
}
.matrix-floating-tooltip-line {
  display: block;
  font-size: 11px;
  line-height: 1.6;
}
@media (pointer: coarse) {
  .pulse-track {
    min-height: 36px;
  }
}
</style>
