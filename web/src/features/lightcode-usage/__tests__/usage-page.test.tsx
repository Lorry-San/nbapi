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
// bun:test is needed for mock.module; the project has no bun type declarations.
// @ts-expect-error -- resolved by the bun test runner
import { afterAll as after, describe, mock, test } from 'bun:test'
import assert from 'node:assert/strict'

import { Window } from 'happy-dom'

const domWindow = new Window({
  url: 'https://lightcode.lcapi.online/lightcode/usage',
})
for (const key of [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
  'localStorage',
  'matchMedia',
  'ResizeObserver',
] as const) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key as keyof Window],
  })
}

/** Optional profile extras a test can turn on: reset cards and a scheduled reset. */
let extras: {
  reset_cards?: { id: number; kind: 'short' | 'full'; expires_at: number }[]
  scheduled_reset?: { run_at: number } | null
} = {}

/** A year of days ending today, shaped like the real /api/lightcode/profile response. */
function profile() {
  const days = Array.from({ length: 371 }, (_, i) => {
    const d = new Date(Date.UTC(2026, 9, 19) - (370 - i) * 86400000)
    const tokens = i > 360 ? 1000 * (i - 360) : 0
    return {
      date: d.toISOString().slice(0, 10),
      requests: tokens ? 3 : 0,
      total_tokens: tokens,
    }
  })
  return {
    success: true,
    data: {
      user: { id: 1, username: 'LorrySan', display_name: 'LorrySan' },
      subscription: {
        plan: 'Lightcode Plus',
        end_time: 1795000000,
        window_short: null,
        window_long: { seconds: 604800, used_percent: 1, reset_at: 1793000000 },
        scheduled_reset: extras.scheduled_reset ?? null,
      },
      reset_cards: extras.reset_cards ?? [],
      stats: {
        total_tokens: 55000,
        requests: 30,
        active_days: 10,
        peak_day_tokens: 10000,
        peak_day: '2026-10-19',
        longest_streak_days: 10,
        current_streak_days: 10,
      },
      days,
      models: [
        {
          model_id: 'deepseek-v4.1-flash',
          model: 'DeepSeek V4.1 Flash',
          requests: 30,
          total_tokens: 55000,
        },
      ],
      truncated: false,
    },
  }
}

// Only the network call is replaced; the page, hooks and formatting run for real.
await mock.module('@/features/lightcode/api', () => ({
  getLightcodeProfile: () => Promise.resolve(profile()),
  useLightcodeResetCard: () => Promise.resolve({ success: true, data: { kind: 'short', used_at: 1 } }),
  updateLightcodeDisplayName: () => Promise.resolve({ success: true, data: { display_name: 'x' } }),
  uploadLightcodeAvatar: () => Promise.resolve({ success: true, data: { avatar_url: '' } }),
  deleteLightcodeAvatar: () => Promise.resolve({ success: true, data: null }),
}))

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { createRootRoute, createRouter, RouterProvider, createMemoryHistory } =
  await import('@tanstack/react-router')

const i18n = createInstance()
await i18n
  .use(initReactI18next)
  .init({ lng: 'zhCN', resources: { zhCN: { translation: {} } } })
;(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true

const { LightcodeUsage } = await import('../index')

/** Renders the page against the current extras and returns its container once loaded. */
async function renderPage() {
  const errors: unknown[] = []
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container, {
    onUncaughtError: (e) => errors.push(e),
    onCaughtError: (e) => errors.push(e),
  })
  const router = createRouter({
    routeTree: createRootRoute({ component: LightcodeUsage }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <QueryClientProvider client={client}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </I18nextProvider>
    )
  })
  for (let i = 0; i < 20 && !container.textContent?.includes('LorrySan'); i++) {
    await act(() => new Promise((r) => setTimeout(r, 25)))
  }
  return { container, errors, unmount: () => act(() => root.unmount()) }
}

describe('Lightcode usage page', () => {
  after(() => domWindow.happyDOM.close())

  test('a user holding reset cards sees how many and can spend one', async () => {
    extras = {
      reset_cards: [
        { id: 7, kind: 'short', expires_at: 1795000000 },
        { id: 8, kind: 'full', expires_at: 1795000000 },
      ],
    }
    const page = await renderPage()
    try {
      const slot = page.container.querySelector<HTMLElement>('[data-slot="lightcode-reset-card"]')
      assert.ok(slot, 'the card section renders where the reserved slot was')
      assert.equal(slot.hidden, false)
      assert.match(slot.textContent ?? '', /2/)
      const buttons = [...slot.querySelectorAll('button')].map((b) => b.textContent)
      assert.equal(buttons.filter((label) => label === 'Use card').length, 2, 'each card has its own button')
    } finally {
      page.unmount()
      extras = {}
    }
  })

  test('a scheduled operator reset is announced on the plan card', async () => {
    extras = { scheduled_reset: { run_at: 1794000000 } }
    const page = await renderPage()
    try {
      assert.match(page.container.textContent ?? '', /Usage will be reset at/)
    } finally {
      page.unmount()
      extras = {}
    }
  })

  test('renders profile, stats, plan, heatmap and models from real-shaped data', async () => {
    extras = {}
    const errors: unknown[] = []
    const originalError = console.error
    console.error = (...args: unknown[]) => {
      errors.push(args)
    }
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container, {
      onUncaughtError: (e) => errors.push(e),
      onCaughtError: (e) => errors.push(e),
    })
    const router = createRouter({
      routeTree: createRootRoute({ component: LightcodeUsage }),
      history: createMemoryHistory({ initialEntries: ['/'] }),
    })
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    try {
      await act(async () => {
        root.render(
          <I18nextProvider i18n={i18n}>
            <QueryClientProvider client={client}>
              <RouterProvider router={router} />
            </QueryClientProvider>
          </I18nextProvider>
        )
      })
      for (
        let i = 0;
        i < 20 && !container.textContent?.includes('LorrySan');
        i++
      ) {
        await act(() => new Promise((r) => setTimeout(r, 25)))
      }
      const text = container.textContent ?? ''
      assert.deepEqual(
        errors.filter((e) => !String(e).includes('act(')),
        [],
        `render errors:\n${errors.map(String).join('\n')}`
      )
      assert.match(text, /LorrySan/)
      assert.match(text, /55K/)
      assert.match(text, /Lightcode Plus/)
      assert.match(text, /DeepSeek V4\.1 Flash/)
      assert.doesNotMatch(text, /deepseek-v4\.1-flash/)
      const heatmap = container.querySelector(
        '[role="img"][aria-label="Token activity over the last 90 days"]'
      )
      assert.ok(heatmap)
      const dates = [...heatmap.querySelectorAll('[title]')].map((cell) =>
        cell.getAttribute('title')?.slice(0, 10)
      )
      assert.equal(dates.length, 90)
      assert.equal(dates[0], '2026-07-22')
      assert.equal(dates.at(-1), '2026-10-19')
      assert.equal(container.querySelectorAll('[aria-pressed]').length, 0)
      const scrollArea = heatmap.parentElement?.parentElement
      assert.ok(
        scrollArea?.classList.contains('no-scrollbar'),
        'activity must hide the native horizontal scrollbar'
      )
      // Without any card the section stays out of the page entirely.
      assert.equal(container.querySelector('[data-slot="lightcode-reset-card"]'), null)
      assert.doesNotMatch(text, /Usage will be reset at/)
    } finally {
      console.error = originalError
      act(() => root.unmount())
    }
  })
})
