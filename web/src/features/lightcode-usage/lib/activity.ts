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

export interface ActivityDay {
  date: string
  total_tokens: number
}

export interface HeatmapCell {
  date: string
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

const HEATMAP_DAYS = 90

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

/** Show the last 90 supplied days, oldest first, in Sunday-first week columns. */
export function buildHeatmap(history: ActivityDay[]): Heatmap {
  const days = history.slice(-HEATMAP_DAYS)
  if (days.length === 0) return { columns: [], months: [] }
  const thresholds = levelThresholds(days.map((day) => day.total_tokens))
  const columns: (HeatmapCell | null)[][] = []
  let col: (HeatmapCell | null)[] = Array.from(
    { length: weekdayOf(days[0].date) },
    () => null
  )
  days.forEach((d) => {
    col.push({
      date: d.date,
      tokens: d.total_tokens,
      level: levelOf(d.total_tokens, thresholds),
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
  const months: Heatmap['months'] = []
  let last = -1
  columns.forEach((c, i) => {
    const first = c.find((cell) => cell !== null)
    if (!first) return
    const month = Number(first.date.slice(5, 7)) - 1
    if (month !== last) {
      months.push({ column: i, month })
      last = month
    }
  })
  return { columns, months }
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
