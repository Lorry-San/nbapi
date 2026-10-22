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
// ============================================================================
// Display Name Validation
// ============================================================================

/** Mirrors model.LightcodeDisplayNameMax and the users.display_name limit. */
export const DISPLAY_NAME_MAX = 20

/**
 * Count characters the way the server does: trimmed, then Unicode runes, so
 * one emoji is one character rather than two UTF-16 units.
 */
export function getDisplayNameLength(value: string): number {
  return [...value.trim()].length
}

/**
 * Whether a draft display name can be saved. The server trims and validates
 * the same way, so an all-whitespace draft is rejected before uploading.
 */
export function isValidDisplayName(value: string): boolean {
  const length = getDisplayNameLength(value)
  return length > 0 && length <= DISPLAY_NAME_MAX
}
