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
  transformFormDataToCreatePayload,
} from '../channel-form'

function channelForm(overrides: Record<string, unknown> = {}) {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'test-channel',
    type: 1,
    force_responses_to_chat_completions: true,
    ...overrides,
  }
}

describe('channel Responses tool conversion mode', () => {
  test('persists the channel override when Responses uses Chat Completions upstream', () => {
    const result = transformFormDataToCreatePayload(
      channelForm({ responses_to_chat_tool_mode: 'loose' })
    )

    assert.equal(
      JSON.parse(result.channel.settings as string).responses_to_chat_tool_mode,
      'loose'
    )
  })

  test('does not persist a namespace override when Responses conversion is disabled', () => {
    const result = transformFormDataToCreatePayload(
      channelForm({
        force_responses_to_chat_completions: false,
        responses_to_chat_tool_mode: 'strict',
      })
    )

    assert.equal(
      'responses_to_chat_tool_mode' in
        JSON.parse(result.channel.settings as string),
      false
    )
  })
})
