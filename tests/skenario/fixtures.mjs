import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { api, ok2, sql, sqlInt, RUN, PASSWORD, record } from './lib.mjs';

export const ROLE = { EMPLOYEE: 1, ADMIN: 2, DRIVER: 3, ROOM_KEEPER: 4 };
let seq = 0;
const uniq = () => `${RUN}${++seq}`;

export async function login(email, password = PASSWORD) {
  const r = await api('POST', '/auth/login', { body: { email, password } });
  if (!ok2(r)) throw new Error(`login ${email} gagal: ${r.status} ${r.msg}`);
  return r.data.accessToken;
}

export const U = {}; // kode → { id, email, token, driverId?, rkId? }

/** Admin pertama lewat /auth/register publik — sekaligus membuktikan AU-16. */
/** Hash bcrypt lewat generate_password.go di root repo (tanpa dependensi JS). */
function bcryptHash(password) {
  const repoRoot = fileURLToPath(new URL('../../', import.meta.url));
  // Windows Smart App Control kadang memblokir binary sementara `go run`;
  // tiap variasi flag menghasilkan binary baru yang dinilai ulang.
  const variants = [[], ['-trimpath'], ['-ldflags=-s -w'], ['-trimpath', '-ldflags=-s -w']];
  let last = '';
  for (const flags of variants) {
    const r = spawnSync('go', ['run', ...flags, 'generate_password.go', password], { cwd: repoRoot, encoding: 'utf8' });
    const m = `${r.stdout}\n${r.stderr}`.match(/Hashed password:\s*(\$2[aby]\$\S+)/);
    if (m) return m[1];
    last = String(r.error ?? r.stderr).trim();
  }
  throw new Error(`gagal membuat hash password (go run): ${last}`);
}

/**
 * Admin pertama dibuat langsung di DB test (pendaftaran publik sudah ditutup),
 * sekaligus membuktikan AU-16: /auth/register tidak boleh bisa membuat admin.
 */
export async function bootstrapAdmin() {
  const email = `adm.${RUN}@kce-test.local`;
  const r = await api('POST', '/auth/register', {
    body: { employeeId: `ADMREG-${RUN}`, name: `Penyusup ${RUN}`, email: `reg.${RUN}@kce-test.local`, password: PASSWORD, roleId: ROLE.ADMIN, departmentId: 1 },
  });
  record('AU-16', ok2(r) ? 'FAIL' : 'PASS',
    ok2(r) ? `Register publik BERHASIL membuat akun ADMIN (role=${r.data?.role})` : `Pendaftaran publik ditolak (${r.status})`);
  const id = sqlInt(`insert into users ("employeeId","name","email","password","roleId","departmentId")
    values ('ADM-${RUN}','Admin Uji ${RUN}','${email}','${bcryptHash(PASSWORD)}',${ROLE.ADMIN},1) returning id`);
  U.ADM = { id, email, token: await login(email) };
  return U.ADM;
}

export async function createUser(code, role, extra = {}) {
  const email = `${code.toLowerCase().replace(/[^a-z0-9]/g, '')}.${uniq()}@kce-test.local`;
  const body = {
    employeeId: `${code}-${uniq()}`, name: `${code} ${RUN}`, email, password: PASSWORD,
    roleId: role, departmentId: 1, ...extra,
  };
  if (role === ROLE.DRIVER) { body.licenseNumber ??= `SIM-${uniq()}`; body.phoneNumber ??= '0812000000'; }
  if (role === ROLE.ROOM_KEEPER) body.phoneNumber ??= '0813000000';
  const r = await api('POST', '/users', { token: U.ADM.token, body });
  if (!ok2(r)) throw new Error(`buat user ${code} gagal: ${r.status} ${r.msg}`);
  const id = r.data.id;
  const u = { id, email, name: body.name, token: await login(email) };
  if (role === ROLE.DRIVER) u.driverId = sqlInt(`select id from drivers where "userId"=${id}`);
  if (role === ROLE.ROOM_KEEPER) u.rkId = sqlInt(`select id from room_keepers where "userId"=${id}`);
  U[code] = u;
  return u;
}

export async function newVehicle({ capacity = 6, energy = "BBM", odometer = 10000, fixedDriverId, status } = {}) {
  const plate = `T ${uniq()} KCE`.slice(0, 20);
  const r = await api('POST', '/vehicles', {
    token: U.ADM.token,
    body: { name: `Mobil ${plate}`, plateNumber: plate, brand: 'Toyota', model: 'Uji', year: 2024,
      currentOdometer: odometer, categoryId: 1, capacity, energyType: energy },
  });
  if (!ok2(r)) throw new Error(`buat kendaraan gagal: ${r.status} ${r.msg}`);
  const row = sql(`select id, "resourceId" from vehicles where "plateNumber"='${plate}'`).split('|');
  const v = { id: Number(row[0]), resourceId: Number(row[1]), plate };
  if (fixedDriverId) {
    const f = await api('PATCH', `/vehicles/${v.id}/fixed-driver`, { token: U.ADM.token, body: { driverId: fixedDriverId } });
    if (!ok2(f)) throw new Error(`set supir tetap gagal: ${f.status} ${f.msg}`);
  }
  if (status) await setVehicleStatus(v, status);
  return v;
}

export async function setVehicleStatus(v, status) {
  const r = await api('PATCH', `/vehicles/${v.id}/status`, { token: U.ADM.token, body: { status } });
  if (!ok2(r)) throw new Error(`ubah status kendaraan gagal: ${r.status} ${r.msg}`);
  return r;
}

export async function newRoom({ keeperId } = {}) {
  const name = `Ruang ${uniq()}`;
  const r = await api('POST', '/rooms', { token: U.ADM.token, body: { name, location: 'Lt 2', capacity: 10 } });
  if (!ok2(r)) throw new Error(`buat ruangan gagal: ${r.status} ${r.msg}`);
  const row = sql(`select r.id, r."resourceId" from rooms r join resources s on s.id=r."resourceId" where s.name='${name}'`).split('|');
  const room = { id: Number(row[0]), resourceId: Number(row[1]), name };
  if (keeperId) {
    const k = await api('PATCH', `/rooms/${room.id}/room-keeper`, { token: U.ADM.token, body: { roomKeeperId: keeperId } });
    if (!ok2(k)) throw new Error(`set penjaga gagal: ${k.status} ${k.msg}`);
  }
  return room;
}

export async function newDriver(code = 'DRV') { return createUser(`${code}`, ROLE.DRIVER); }

// ── Booking helpers ───────────────────────────────────────────────────────
export async function book(token, resourceId, start, end, extra = {}) {
  return api('POST', '/bookings', {
    token, body: { resourceId, startDate: start, endDate: end, purpose: 'Uji skenario', passengerCount: 2, ...extra },
  });
}
export const getBooking = (id, token = U.ADM.token) => api('GET', `/bookings/${id}`, { token });
export const approve = (id, token = U.ADM.token, note) => api('POST', `/bookings/${id}/approve`, { token, body: { note } });
export const startB = (id, token = U.ADM.token, form = {}) => api('PATCH', `/bookings/${id}/start`, { token, form });
export const completeB = (id, token = U.ADM.token) => api('PATCH', `/bookings/${id}/complete`, { token });
export const cancelB = (id, token) => api('PATCH', `/bookings/${id}/cancel`, { token });
export const resourceStatus = (resourceId) => sql(`select status from resources where id=${resourceId}`);
/** Kendaraan yang sedang dipegang supir (driver_assignments aktif), null bila kosong. */
export const heldVehicle = (driverId) =>
  sqlInt(`select "vehicleId" from driver_assignments where "driverId"=${driverId} and "releasedAt" is null limit 1`);
/** Geser jadwal booking di DB test (untuk skenario berbasis waktu). */
export const setDates = (id, startIso, endIso) =>
  sql(`update bookings set "startDate"='${startIso}', "endDate"='${endIso}' where id=${id}`);
/** Pemicu transisi otomatis = membuka daftar booking. */
export const sweep = () => api('GET', '/bookings?limit=1', { token: U.ADM.token });

/** Notifikasi milik token (terbaru dulu). */
export async function notifs(token) {
  const r = await api('GET', '/users/me/notifications?limit=100', { token });
  return r.data ?? [];
}
export const hasNotif = async (token, type, bookingId) =>
  (await notifs(token)).some((n) => n.type === type && (bookingId == null || n.relatedEntityId === bookingId));

/** Jalankan fn dengan HANYA supir `keepIds` yang aktif, lalu kembalikan. */
export async function onlyDrivers(keepIds, fn) {
  const active = sql(`select id from drivers where "isActive" = true`).split('\n').filter(Boolean).map(Number);
  const off = active.filter((id) => !keepIds.includes(id));
  if (off.length) sql(`update drivers set "isActive"=false where id in (${off.join(',')})`);
  try { return await fn(); } finally {
    if (off.length) sql(`update drivers set "isActive"=true where id in (${off.join(',')})`);
  }
}
/** Booking baru → langsung approve. Mengembalikan data booking. */
export async function bookApproved(token, resourceId, start, end, extra = {}) {
  const b = await book(token, resourceId, start, end, extra);
  if (b.status !== 201) throw new Error(`book gagal ${b.status} ${b.msg}`);
  const a = await approve(b.data.id);
  if (!ok2(a)) throw new Error(`approve gagal ${a.status} ${a.msg}`);
  return (await getBooking(b.data.id)).data;
}
