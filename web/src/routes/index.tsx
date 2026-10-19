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
import { createFileRoute } from '@tanstack/react-router'

import { Home } from '@/features/home'

export const Route = createFileRoute('/')({
  // In production the server answers "/" with the static Lightcode site, so in-app
  // navigation to the homepage reloads the page instead of rendering the SPA home.
  // If the server still serves the SPA at "/", the guard stops a reload loop.
  beforeLoad: () => {
    if (!import.meta.env.PROD) return
    const key = 'lightcode-home-reload'
    const last = Number(sessionStorage.getItem(key) || 0)
    if (Date.now() - last < 10_000) return
    sessionStorage.setItem(key, String(Date.now()))
    window.location.assign('/')
  },
  component: Home,
})
