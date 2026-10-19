import assert from 'node:assert/strict'
import { test } from 'node:test'

import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

import { TopNav } from '../top-nav'

test('Lightcode product site uses same-tab document navigation', () => {
  const html = renderToStaticMarkup(
    createElement(TopNav, {
      links: [{ title: 'Lightcode', href: '/#agent', document: true }],
    })
  )
  assert.match(html, /href="\/#agent"/)
  assert.doesNotMatch(html, /target="_blank"/)
})
