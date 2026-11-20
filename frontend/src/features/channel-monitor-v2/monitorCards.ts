import type { MonitorCandyHistory, MonitorCandyResult, MonitorMatrixRow } from '@/api/channelMonitorV2'

export const CANDY_HISTORY_LIMIT = 100
export const CANDY_HISTORY_LIMIT_MIN = 15
export const CANDY_HISTORY_LIMIT_MAX = 100
export function candyHistoryLimit(history: { history_limit?: number } | null | undefined): number {
  const limit = history?.history_limit
  if (typeof limit === 'number' && Number.isInteger(limit) && limit >= CANDY_HISTORY_LIMIT_MIN && limit <= CANDY_HISTORY_LIMIT_MAX) return limit
  return CANDY_HISTORY_LIMIT
}

export function monitorCandyHistories(row: Pick<MonitorMatrixRow, 'candy' | 'candy_checks'>): MonitorCandyHistory[] {
  if (row.candy_checks && row.candy_checks.length > 0) return row.candy_checks
  return row.candy ? [row.candy] : []
}

export function candyHistorySlots(history: MonitorCandyHistory): Array<MonitorCandyResult | null> {
  const limit = candyHistoryLimit(history)
  const results = history.results.slice(-limit)
  return [...Array<null>(limit - results.length).fill(null), ...results]
}

export function monitorRefreshSeconds(configSeconds: number | undefined, rows: readonly MonitorMatrixRow[], bootstrap: boolean): number {
  if (bootstrap) return 10
  let seconds = configSeconds && Number.isFinite(configSeconds) && configSeconds > 0 ? configSeconds : 300
  for (const row of rows) {
    for (const history of monitorCandyHistories(row)) {
      const interval = history.interval_minutes
      if (interval && Number.isFinite(interval) && interval > 0) {
        seconds = Math.min(seconds, interval * 60)
      }
    }
  }
  // Refresh before a healthy minute probe can appear stale in a five-minute snapshot.
  return Math.max(60, seconds)
}

export function candyDisplayState(history: MonitorCandyHistory, now: number): 'correct' | 'incorrect' | 'error' | 'unknown' | 'stale' {
  const latest = history.results.at(-1)
  if (!latest) return 'unknown'
  const age = now - Date.parse(latest.checked_at)
  if (!Number.isFinite(age) || age > Math.max(180, history.interval_minutes * 60 + 120) * 1000) return 'stale'
  return latest.verdict
}
