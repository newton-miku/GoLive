const BASE = '';
const TOKEN_KEY = 'golive_token';

function getToken() {
  return localStorage.getItem(TOKEN_KEY) || '';
}

function setToken(token) {
  localStorage.setItem(TOKEN_KEY, token);
}

function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

function authHeaders() {
  const token = getToken();
  return token ? { 'Authorization': `Bearer ${token}` } : {};
}

export async function apiLogin(username, password) {
  const res = await fetch(`${BASE}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  const data = await res.json();
  if (!data.success) throw new Error(data.message);
  if (data.token) setToken(data.token);
  return data;
}

export async function apiRegister(username, password) {
  const res = await fetch(`${BASE}/api/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  const data = await res.json();
  if (!data.success) throw new Error(data.message);
  if (data.token) setToken(data.token);
  return data;
}

export async function apiLogout() {
  clearToken();
  const res = await fetch(`${BASE}/api/auth/logout`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
  });
  return res.json().catch(() => ({}));
}

export async function apiMe() {
  const res = await fetch(`${BASE}/api/auth/me`, {
    headers: authHeaders(),
  });
  const data = await res.json();
  // 如果后端返回了新 token，更新本地存储
  if (data.token) setToken(data.token);
  return data;
}

export async function apiResetKey() {
  const res = await fetch(`${BASE}/api/auth/reset-key`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
  });
  const data = await res.json();
  if (!data.success) throw new Error(data.message);
  if (data.token) setToken(data.token);
  return data;
}

export async function apiRooms() {
  const res = await fetch(`${BASE}/recommend`, {
    headers: authHeaders(),
  });
  return res.json();
}

export async function apiStatus(roomId) {
  const res = await fetch(`${BASE}/status?room=${roomId}`, {
    headers: authHeaders(),
  });
  return { ok: res.ok, status: res.status, data: res.status === 200 ? res.json() : null };
}
