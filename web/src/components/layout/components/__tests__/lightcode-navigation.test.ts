import assert from 'node:assert/strict'
import { test } from 'node:test'

import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

import { TopNav } from '../top-nav'

test('Lightcode Client uses same-tab document navigation', () => {
  const html = renderToStaticMarkup(
    createElement(TopNav, {
      links: [{ title: 'Lightcode Client', href: '/lightcode', document: true }],
    })
  )
  assert.match(html, /href="\/lightcode"/)
  assert.doesNotMatch(html, /target="_blank"/)
})

test('API Status opens the external status site in a new tab', () => {
  const html = renderToStaticMarkup(
    createElement(TopNav, {
      links: [
        {
          title: 'API Status',
          href: 'https://status.lcapi.online/',
          external: true,
        },
      ],
    })
  )
  assert.match(html, /href="https:\/\/status\.lcapi\.online\/"/)
  assert.match(html, /target="_blank"/)
  assert.match(html, /rel="noopener noreferrer"/)
})
