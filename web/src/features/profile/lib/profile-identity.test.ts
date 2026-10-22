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
import { describe, test } from 'node:test'

import { validateAvatarFile } from './avatar-upload'
import { getDisplayNameLength, isValidDisplayName } from './display-name'

/** Minimal File stand-in; node has no DOM File constructor for size-only checks. */
function picked(name: string, type: string, size: number): File {
  return { name, type, size } as File
}

describe('avatar upload validation', () => {
  test('accepts every format the server sniffs', () => {
    for (const type of ['image/png', 'image/jpeg', 'image/webp', 'image/gif']) {
      assert.equal(validateAvatarFile(picked('a', type, 1024)), null)
    }
  })

  test('rejects a format the server would refuse', () => {
    assert.equal(
      validateAvatarFile(picked('a.bmp', 'image/bmp', 1024)),
      'Only PNG, JPEG, WebP or GIF images are supported'
    )
    // A renamed text file still carries its real type here.
    assert.equal(
      validateAvatarFile(picked('a.png', 'text/plain', 1024)),
      'Only PNG, JPEG, WebP or GIF images are supported'
    )
  })

  test('accepts a picture exactly at the limit and rejects one byte over', () => {
    const limit = 256 * 1024
    assert.equal(validateAvatarFile(picked('a', 'image/png', limit)), null)
    assert.equal(
      validateAvatarFile(picked('a', 'image/png', limit + 1)),
      'Avatar must be 256KB or smaller'
    )
  })
})

describe('display name validation', () => {
  test('counts runes so one emoji is one character', () => {
    assert.equal(getDisplayNameLength('abc'), 3)
    assert.equal(getDisplayNameLength('春日野穹'), 4)
    assert.equal(getDisplayNameLength('😀😀'), 2)
  })

  test('trims before measuring, matching the server', () => {
    assert.equal(getDisplayNameLength('  春日野穹  '), 4)
  })

  test('accepts 1 to 20 characters and refuses anything else', () => {
    assert.equal(isValidDisplayName('a'), true)
    assert.equal(isValidDisplayName('a'.repeat(20)), true)
    assert.equal(isValidDisplayName('a'.repeat(21)), false)
    assert.equal(isValidDisplayName(''), false)
    assert.equal(isValidDisplayName('   '), false)
  })

  test('counts emoji as one character toward the limit', () => {
    assert.equal(isValidDisplayName('😀'.repeat(20)), true)
    assert.equal(isValidDisplayName('😀'.repeat(21)), false)
  })
})
