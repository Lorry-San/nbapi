/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export type ActivityMode = 'daily' | 'weekly' | 'cumulative'

export interface ActivityDay {
  date: string
  total_tokens: number
}

export interface HeatmapCell {
  date: string
  /** Value used for coloring in the selected mode. */
  value: number
  /** The day's own tokens, for the tooltip. */
  tokens: number
  level: number
}

export interface Heatmap {
  /** columns[week][weekday 0=Sunday..6]; null pads days outside the range. */
  columns: (HeatmapCell | null)[][]
  /** Month label for the first column that starts a month. */
  months: { column: number; month: number }[]
}

export const HEATMAP_WEEKS = 53

/** Compact token format: 1234 -> 1.2K, 5600000 -> 5.6M. */
export function formatTokens(n: number): string {
  const units: [number, string][] = [
    [1e9, 'B'],
    [1e6, 'M'],
    [1e3, 'K'],
  ]
  for (const [size, unit] of units) {
    if (Math.abs(n) >= size) {
      const v = n / size
      return `${v >= 100 ? Math.round(v) : Number(v.toFixed(1))}${unit}`
    }
  }
  return String(Math.round(n))
}

function weekdayOf(date: string): number {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(Date.UTC(y, m - 1, d)).getUTCDay()
}

/** Value per day for the given mode, aligned with `days`. */
export function modeValues(days: ActivityDay[], mode: ActivityMode): number[] {
  if (mode === 'daily') return days.map((d) => d.total_tokens)
  if (mode === 'cumulative') {
    let sum = 0
    return days.map((d) => (sum += d.total_tokens))
  }
  // weekly: weeks start on Sunday
  const out: number[] = days.map(() => 0)
  let start = 0
  while (start < days.length) {
    let end = start
    let sum = days[start].total_tokens
    while (end + 1 < days.length && weekdayOf(days[end + 1].date) !== 0) {
      end++
      sum += days[end].total_tokens
    }
    for (let i = start; i <= end; i++) out[i] = sum
    start = end + 1
  }
  return out
}

/** Quartile thresholds over the non-zero values. */
export function levelThresholds(values: number[]): number[] {
  const nz = values.filter((v) => v > 0).sort((a, b) => a - b)
  if (nz.length === 0) return []
  const q = (p: number) =>
    nz[Math.min(nz.length - 1, Math.floor(p * nz.length))]
  return [q(0.25), q(0.5), q(0.75)]
}

/** 0 for empty, otherwise 1..4. */
export function levelOf(value: number, thresholds: number[]): number {
  if (value <= 0 || thresholds.length === 0) return 0
  let level = 1
  for (const t of thresholds) if (value > t) level++
  return Math.min(4, level)
}

/** Lay days (oldest first) out as Sunday-first week columns, keeping the last HEATMAP_WEEKS. */
export function buildHeatmap(days: ActivityDay[], mode: ActivityMode): Heatmap {
  if (days.length === 0) return { columns: [], months: [] }
  const values = modeValues(days, mode)
  const thresholds = levelThresholds(values)
  const columns: (HeatmapCell | null)[][] = []
  let col: (HeatmapCell | null)[] = Array.from(
    { length: weekdayOf(days[0].date) },
    () => null
  )
  days.forEach((d, i) => {
    col.push({
      date: d.date,
      value: values[i],
      tokens: d.total_tokens,
      level: levelOf(values[i], thresholds),
    })
    if (col.length === 7) {
      columns.push(col)
      col = []
    }
  })
  if (col.length > 0) {
    while (col.length < 7) col.push(null)
    columns.push(col)
  }
  const kept = columns.slice(-HEATMAP_WEEKS)
  const months: Heatmap['months'] = []
  let last = -1
  kept.forEach((c, i) => {
    const first = c.find((cell) => cell !== null)
    if (!first) return
    const month = Number(first.date.slice(5, 7)) - 1
    if (month !== last) {
      if (last !== -1 || first.date.slice(8) <= '07') {
        months.push({ column: i, month })
      }
      last = month
    }
  })
  return { columns: kept, months }
}

/** Deterministic avatar hue from the user id. */
export function avatarHue(id: number): number {
  return Math.abs(Math.imul(id | 0, 2654435761) >>> 0) % 360
}

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) {
    return [...parts[0]].slice(0, 2).join('').toUpperCase()
  }
  return ([...parts[0]][0] + [...parts[1]][0]).toUpperCase()
}
