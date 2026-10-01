// Helper runner skenario KCE — backend LOKAL (localhost:8090, DB kce_test).
import { execFileSync } from 'node:child_process';
import { randomBytes } from 'node:crypto';

export const BASE = process.env.KCE_API ?? 'http://127.0.0.1:8090/api/v1';
const PSQL = process.env.PSQL_PATH ?? 'psql';

if (!/^http:\/\/(localhost|127\.0\.0\.1)[:/]/.test(BASE)) throw new Error('Runner hanya boleh ke localhost');

export const RUN = Date.now().toString(36).slice(-5);
export const PASSWORD = 'Tes-' + randomBytes(9).toString('base64url'); // hanya di memori

// ── DB (hanya kce_test) ───────────────────────────────────────────────────
export function sql(q) {
  return execFileSync(PSQL, ['-h', 'localhost', '-U', 'postgres', '-d', 'kce_test', '-tAc', q], {
    env: { ...process.env, PGPASSWORD: process.env.KCE_PGPASS },
  }).toString().trim();
}
export const sqlInt = (q) => { const v = sql(q); return v === '' ? null : Number(v.split('\n')[0]); };

// ── HTTP ──────────────────────────────────────────────────────────────────
export async function api(method, path, { token, body, form, headers = {} } = {}) {
  const h = { ...headers };
  if (token) h.Authorization = `Bearer ${token}`;
  let payload;
  if (form) {
    const fd = new FormData();
    for (const [k, v] of Object.entries(form)) {
      if (v === undefined || v === null) continue;
      if (v instanceof Blob) fd.append(k, v, 'foto.jpg'); else fd.append(k, String(v));
    }
    payload = fd;
  } else if (body !== undefined) {
    h['Content-Type'] = 'application/json';
    payload = JSON.stringify(body);
  }
  const res = await fetch(BASE + path, { method, headers: h, body: payload });
  let json = null;
  try { json = await res.json(); } catch { /* bukan JSON */ }
  return {
    status: res.status, json, data: json?.data,
    msg: json?.message ?? json?.error?.message ?? '',
    changed: res.headers.get('x-data-changed'),
  };
}
export const ok2 = (r) => r.status >= 200 && r.status < 300;
export const fakePhoto = () => new Blob([Buffer.from('ffd8ffe000104a464946', 'hex')], { type: 'image/jpeg' });

// ── Waktu (WIB) ───────────────────────────────────────────────────────────
export const inMin = (m) => new Date(Date.now() + m * 60000).toISOString();
/** Instant untuk jam dinding WIB `hh:mm` pada hari ini + `days`. */
export function wib(days, hh, mm = 0) {
  const now = new Date(Date.now() + 7 * 3600e3); // jam dinding WIB
  return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() + days, hh - 7, mm)).toISOString();
}

// ── Hasil ─────────────────────────────────────────────────────────────────
export const results = [];
/** status: PASS | FAIL | INFO (perilaku perlu keputusan) | ERROR */
export function record(id, status, note = '') {
  results.push({ id, status, note });
  const icon = { PASS: '✅', FAIL: '❌', INFO: 'ℹ️ ', ERROR: '💥', SKIP: '⏭️ ' }[status] ?? '?';
  console.log(`${icon} ${id.padEnd(7)} ${note}`);
}
export const check = (id, cond, passNote, failNote) =>
  record(id, cond ? 'PASS' : 'FAIL', cond ? passNote : (failNote ?? passNote));

export async function scenario(id, fn) {
  try { await fn(); } catch (e) { record(id, 'ERROR', String(e?.stack ?? e).split('\n').slice(0, 3).join(' | ')); }
}

/** Request mentah (mis. unduh PDF): status, content-type, isi biner. */
export async function apiRaw(method, path, { token } = {}) {
  const res = await fetch(BASE + path, { method, headers: token ? { Authorization: `Bearer ${token}` } : {} });
  const bytes = Buffer.from(await res.arrayBuffer());
  return { status: res.status, contentType: res.headers.get('content-type') ?? '', disposition: res.headers.get('content-disposition') ?? '', bytes };
}
