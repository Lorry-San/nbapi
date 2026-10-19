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
import assert from 'node:assert/strict'
import { test } from 'node:test'

import {
  avatarHue,
  buildHeatmap,
  formatTokens,
  initials,
  levelOf,
  levelThresholds,
} from '../activity'

function makeDays(start: string, count: number, tokens: (i: number) => number) {
  const base = Date.parse(`${start}T00:00:00Z`)
  return Array.from({ length: count }, (_, i) => ({
    date: new Date(base + i * 86400000).toISOString().slice(0, 10),
    total_tokens: tokens(i),
  }))
}

test('formatTokens is compact with a number', () => {
  assert.equal(formatTokens(0), '0')
  assert.equal(formatTokens(999), '999')
  assert.equal(formatTokens(1234), '1.2K')
  assert.equal(formatTokens(5_600_000), '5.6M')
  assert.equal(formatTokens(2_000_000_000), '2B')
  assert.equal(formatTokens(250_000), '250K')
})

test('levels: zero is empty, quartiles map to 1..4', () => {
  const values = [0, 1, 2, 3, 4, 5, 6, 7, 8]
  const th = levelThresholds(values)
  assert.equal(levelOf(0, th), 0)
  assert.equal(levelOf(1, th), 1)
  assert.equal(levelOf(8, th), 4)
  assert.deepEqual(levelThresholds([0, 0]), [])
})

test('buildHeatmap keeps exactly the last 90 daily values and includes the latest day', () => {
  const days = makeDays('2025-10-13', 371, (i) => i)
  const map = buildHeatmap(days)
  const cells = map.columns.flat().filter((cell) => cell !== null)
  assert.equal(cells.length, 90)
  assert.equal(cells[0].date, '2026-07-21')
  assert.equal(cells[0].tokens, 281)
  assert.equal(cells.at(-1)?.date, '2026-10-18')
  assert.equal(cells.at(-1)?.tokens, 370)
  for (const column of map.columns) assert.equal(column.length, 7)
  assert.equal(map.columns.at(-1)?.[0]?.date, '2026-10-18')
  assert.equal(map.columns.at(-1)?.[1], null)
  assert.ok(map.months.length <= 4)
})

test('buildHeatmap handles no history and shorter history without inventing days', () => {
  assert.deepEqual(buildHeatmap([]), { columns: [], months: [] })
  const map = buildHeatmap(makeDays('2026-10-03', 2, (i) => 10 + i))
  const cells = map.columns.flat().filter((cell) => cell !== null)
  assert.equal(cells.length, 2)
  assert.equal(map.columns[0][6]?.tokens, 10)
  assert.equal(map.columns[1][0]?.tokens, 11)
})

test('avatar helpers are deterministic', () => {
  assert.equal(avatarHue(42), avatarHue(42))
  assert.ok(avatarHue(7) >= 0 && avatarHue(7) < 360)
  assert.equal(initials('Ada Lovelace'), 'AL')
  assert.equal(initials('lorry'), 'LO')
  assert.equal(initials(''), '?')
})
