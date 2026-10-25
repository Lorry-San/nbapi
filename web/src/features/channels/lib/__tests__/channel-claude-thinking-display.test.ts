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

import {
  CHANNEL_FORM_DEFAULT_VALUES,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
} from '../channel-form'
import type { Channel } from '../../types'

function anthropicForm(overrides: Record<string, unknown> = {}) {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'claude-channel',
    type: 14,
    ...overrides,
  }
}

function savedSettings(overrides: Record<string, unknown> = {}) {
  const result = transformFormDataToCreatePayload(anthropicForm(overrides))
  return JSON.parse(result.channel.settings as string) as Record<
    string,
    unknown
  >
}

describe('Claude thinking display setting', () => {
  test('saves summarized when the Anthropic channel selects it', () => {
    assert.equal(
      savedSettings({ claude_thinking_display: 'summarized' })
        .claude_thinking_display,
      'summarized'
    )
  })

  test('omits the field when the model default is selected', () => {
    assert.equal(
      'claude_thinking_display' in
        savedSettings({ claude_thinking_display: '' }),
      false
    )
  })

  test('drops the field when the channel is not Anthropic', () => {
    const result = transformFormDataToCreatePayload(
      anthropicForm({
        type: 1,
        claude_thinking_display: 'summarized',
        settings: JSON.stringify({ claude_thinking_display: 'summarized' }),
      })
    )
    assert.equal(
      'claude_thinking_display' in
        JSON.parse(result.channel.settings as string),
      false
    )
  })

  test('loads a saved value and falls back to the model default for unknown values', () => {
    const channel = (settings: string) =>
      ({ type: 14, settings, channel_info: {} }) as unknown as Channel
    assert.equal(
      transformChannelToFormDefaults(
        channel(JSON.stringify({ claude_thinking_display: 'omitted' }))
      ).claude_thinking_display,
      'omitted'
    )
    assert.equal(
      transformChannelToFormDefaults(
        channel(JSON.stringify({ claude_thinking_display: 'full' }))
      ).claude_thinking_display,
      ''
    )
  })
})
