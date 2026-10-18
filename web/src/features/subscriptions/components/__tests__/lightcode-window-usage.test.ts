import assert from 'node:assert/strict'
import { test } from 'node:test'

import { createInstance } from 'i18next'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'

import { LightcodeWindowUsage } from '../lightcode-window-usage'

test('Lightcode window shows percentage and reset time without monetary quota', async () => {
  const i18n = createInstance()
  await i18n.init({ lng: 'en', resources: { en: { translation: { Used: 'Used', 'Resets at': 'Resets at' } } } })
  const html = renderToStaticMarkup(createElement(I18nextProvider, { i18n }, createElement(LightcodeWindowUsage, {
    label: '5h', active: true,
    window: { used_percent: 27, reset_at: 1800000000 },
  })))
  assert.match(html, /27/)
  assert.match(html, /%/)
  assert.match(html, /Resets at/)
  assert.match(html, /progressbar/)
  assert.doesNotMatch(html, /[$¥]|USD|remaining_usd|limit_usd/)
})
