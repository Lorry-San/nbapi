/*
Copyright (C) 2025 QuantumNous

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

const MOFANG_TOKEN_KEYS = ['access_token', 'jwt', 'token'];

function readTokenFromParams(params) {
  for (const key of MOFANG_TOKEN_KEYS) {
    const value = params.get(key)?.trim();
    if (value) return value;
  }
  return '';
}

function readMofangToken() {
  const searchToken = readTokenFromParams(
    new URLSearchParams(window.location.search),
  );
  if (searchToken) return searchToken;

  const hash = window.location.hash.startsWith('#')
    ? window.location.hash.slice(1)
    : window.location.hash;
  return readTokenFromParams(new URLSearchParams(hash));
}

function stripTokenFromUrl() {
  try {
    const url = new URL(window.location.href);
    let changed = false;

    for (const key of MOFANG_TOKEN_KEYS) {
      if (url.searchParams.has(key)) {
        url.searchParams.delete(key);
        changed = true;
      }
    }

    if (url.hash) {
      const hashParams = new URLSearchParams(url.hash.slice(1));
      let hashChanged = false;
      for (const key of MOFANG_TOKEN_KEYS) {
        if (hashParams.has(key)) {
          hashParams.delete(key);
          hashChanged = true;
        }
      }
      if (hashChanged) {
        const nextHash = hashParams.toString();
        url.hash = nextHash ? `#${nextHash}` : '';
        changed = true;
      }
    }

    if (changed) {
      window.history.replaceState(
        window.history.state,
        document.title,
        `${url.pathname}${url.search}${url.hash}`,
      );
    }
  } catch (error) {}
}

function renderCallbackFallback() {
  const root = document.getElementById('root');
  if (!root) return;
  root.innerHTML =
    '<div style="font-family: system-ui, sans-serif; padding: 24px; color: #111827;">Mofang login completed. You can close this window.</div>';
}

export function consumeMofangAccessTokenCallback() {
  const token = readMofangToken();
  if (!token) return false;

  stripTokenFromUrl();

  if (!window.opener || window.opener.closed) {
    return false;
  }

  try {
    window.opener.postMessage(
      {
        type: 'mofang-jwt',
        jwt: token,
      },
      window.location.origin,
    );
  } catch (error) {
    return false;
  }

  renderCallbackFallback();
  window.setTimeout(() => window.close(), 100);
  return true;
}
