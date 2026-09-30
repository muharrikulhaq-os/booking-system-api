// Batch 4: MT, VH, RM, DU, FL, SY (header), TZ, DL
import { api, ok2, sql, sqlInt, record, check, scenario, wib, inMin, fakePhoto, PASSWORD } from './lib.mjs';
import {
  U, ROLE, createUser, newVehicle, newRoom, newDriver, book, getBooking, approve, startB, completeB,
  bookApproved, heldVehicle, resourceStatus, setVehicleStatus,
} from './fixtures.mjs';

const A = () => U.ADM.token;
const mBody = (vehicleId, start, end, status = 'pending') => ({ vehicleId, type: 'REPAIR', status, description: 'Uji', location: 'Bengkel', startDate: start, endDate: end });
const maint = (vehicleId, start, end, status) => api('POST', '/maintenance', { token: A(), body: mBody(vehicleId, start, end, status) });
const maintId = (vehicleId) => sqlInt(`select id from maintenance_records where "vehicleId"=${vehicleId} order by id desc limit 1`);
const fuel = (token, v, before, after, extra = {}) => api('POST', '/fuel-expenses', {
  token, form: { vehicleId: v.id, fuelTypeId: 1, liter: 10, odometerBefore: before, odometerAfter: after, proofPhoto: fakePhoto(), ...extra } });
const vehicleBody = (v, odometer) => ({ name: `Mobil ${v.plate}`, plateNumber: v.plate, brand: 'Toyota', model: 'Uji', year: 2024, currentOdometer: odometer, categoryId: 1, capacity: 6 });

export async function runBatch4() {
  // ── MT ────────────────────────────────────────────────────────────────
  await scenario('MT-01/02', async () => {
    const v = await newVehicle();
    const m = await maint(v.id, inMin(-5), inMin(600));
    const b = await book(U.EMPA.token, v.resourceId, inMin(60), inMin(120));
    check('MT-01', m.status === 201 && resourceStatus(v.resourceId) === 'MAINTENANCE' && b.status === 409 && m.changed === 'maintenance',
      `Maintenance hari ini → kendaraan ${resourceStatus(v.resourceId)}, booking ${b.status}, X-Data-Changed=${m.changed}`);
    const m2 = await maint(v.id, inMin(-5), inMin(600));
    check('MT-02', m2.status === 409, `Maintenance kedua saat masih terbuka → ${m2.status}`);
  });
  await scenario('MT-03', async () => {
    const d = await newDriver('DRVMT3'); const v = await newVehicle();
    await bookApproved(U.EMPA.token, v.resourceId, wib(60, 9), wib(60, 12), { driverId: d.driverId });
    const m = await maint(v.id, wib(60, 7), wib(60, 17));
    check('MT-03', m.status === 201 && !!m.json?.warning, `Maintenance bentrok dengan booking disetujui → dibuat + peringatan "${m.json?.warning ?? '-'}"`);
  });
  await scenario('MT-04', async () => {
    const v = await newVehicle();
    await maint(v.id, wib(7, 8), wib(7, 17));
    const st = resourceStatus(v.resourceId);
    const b = await book(U.EMPA.token, v.resourceId, wib(1, 9), wib(1, 10));
    check('MT-04', st === 'AVAILABLE' && b.status === 201, `Maintenance minggu depan tidak mengunci kendaraan hari ini`,
      `Maintenance untuk MINGGU DEPAN langsung membuat kendaraan ${st}; booking BESOK ditolak (${b.status})`);
  });
  await scenario('MT-05/06/14', async () => {
    const v = await newVehicle({ odometer: 30000 });
    await maint(v.id, inMin(-5), null);
    const far = await book(U.EMPA.token, v.resourceId, wib(90, 9), wib(90, 10));
    check('MT-13', far.status === 409, `Maintenance tanpa tanggal selesai memblokir tanggal jauh (${far.status})`);
    sql(`update vehicles set "currentOdometer"=31000 where id=${v.id}`);
    const id = maintId(v.id);
    const c = await api('PATCH', `/maintenance/${id}/complete`, { token: A(), form: { 'photos[]': fakePhoto() } });
    const base = sqlInt(`select "lastMaintenanceOdometer" from vehicles where id=${v.id}`);
    check('MT-05', ok2(c) && resourceStatus(v.resourceId) === 'AVAILABLE' && base === 31000, `Selesai → ${resourceStatus(v.resourceId)}, baseline servis ${base}`);
    const c2 = await api('PATCH', `/maintenance/${id}/complete`, { token: A(), form: { 'photos[]': fakePhoto() } });
    check('MT-06', c2.status === 400, `Selesaikan ulang → ${c2.status}`);
    const b = await book(U.EMPA.token, v.resourceId, wib(90, 9), wib(90, 10));
    check('MT-14', b.status === 201, `Setelah maintenance selesai, tanggal yang tadinya terblokir bisa dibooking (${b.status})`);
  });
  await scenario('MT-07', async () => {
    const v = await newVehicle();
    await maint(v.id, inMin(-5), null);
    const id = maintId(v.id);
    const u = await api('PUT', `/maintenance/${id}`, { token: A(), body: mBody(v.id, inMin(-5), null, 'completed') });
    check('MT-07', ok2(u) && resourceStatus(v.resourceId) === 'AVAILABLE', `Edit status → selesai membebaskan kendaraan`,
      `Edit maintenance jadi "completed" (${u.status}) tapi kendaraan tetap ${resourceStatus(v.resourceId)}`);
  });
  await scenario('MT-08', async () => {
    const v = await newVehicle();
    const m = await maint(v.id, inMin(-60), null, 'completed');
    const st = resourceStatus(v.resourceId);
    const c = await api('PATCH', `/maintenance/${maintId(v.id)}/complete`, { token: A(), form: {} });
    check('MT-08', st === 'AVAILABLE', `Maintenance yang dibuat sudah selesai tidak mengunci kendaraan`,
      `Maintenance dibuat berstatus "completed" (${m.status}) tapi kendaraan ${st}; diselesaikan lagi → ${c.status} "${c.msg}" (terkunci)`);
  });
  await scenario('MT-09/10', async () => {
    const d = await newDriver('DRVMT9'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(120), { driverId: d.driverId });
    await startB(b.id, d.token);
    await maint(v.id, inMin(-5), inMin(600));
    const afterCreate = resourceStatus(v.resourceId);
    await api('DELETE', `/maintenance/${maintId(v.id)}`, { token: A() });
    const afterDelete = resourceStatus(v.resourceId);
    check('MT-09', afterDelete === 'IN_USE', `Hapus maintenance saat trip berjalan → tetap IN_USE`,
      `Trip berjalan: buat maintenance → ${afterCreate}, hapus maintenance → ${afterDelete} (seharusnya IN_USE)`);
    const v2 = await newVehicle();
    await maint(v2.id, inMin(-5), inMin(600));
    await api('DELETE', `/maintenance/${maintId(v2.id)}`, { token: A() });
    const b2 = await book(U.EMPA.token, v2.resourceId, wib(2, 9), wib(2, 10));
    check('MT-10', resourceStatus(v2.resourceId) === 'AVAILABLE' && b2.status === 201, `Hapus maintenance terbuka → AVAILABLE & bisa dibooking`);
  });
  await scenario('MT-11/12/FL-05', async () => {
    const d = await newDriver('DRVMT11'); const v = await newVehicle({ odometer: 10000 });
    const f = await fuel(d.token, v, 10000, 20100);
    const n1 = sqlInt(`select count(*) from maintenance_records where "vehicleId"=${v.id} and "isAutoGenerated"`);
    check('MT-11', ok2(f) && n1 === 1 && resourceStatus(v.resourceId) === 'MAINTENANCE', `Isi BBM melewati interval → maintenance otomatis ${n1}, kendaraan ${resourceStatus(v.resourceId)}`);
    record('FL-05', n1 === 1 ? 'PASS' : 'FAIL', 'Sama dengan MT-11');
    await fuel(d.token, v, 20100, 30300);
    const n2 = sqlInt(`select count(*) from maintenance_records where "vehicleId"=${v.id} and "isAutoGenerated"`);
    check('MT-12', n2 === 1, `Batas terlewati lagi saat masih terbuka → tetap ${n2} maintenance`);
    const st = sql(`select status from maintenance_records where "vehicleId"=${v.id} limit 1`);
    record('MT-15', 'INFO', `Maintenance otomatis berstatus "${st}" — pastikan mobile menampilkannya "Berlangsung" (enum mobile hanya pending/completed)`);
  });
  await scenario('MT-17/VH-10', async () => {
    const v = await newVehicle({ odometer: 10000 });
    sql(`update vehicles set "currentOdometer"=12000 where id=${v.id}`);
    const s = await api('GET', `/vehicles/${v.id}/maintenance-status`, { token: A() });
    check('MT-17', s.data?.kmUntilDue === 8000, `Sisa km sampai servis = ${s.data?.kmUntilDue}`);
    const vNew = await newVehicle({ odometer: 15000, keepZeroBaseline: true });
    const s2 = await api('GET', `/vehicles/${vNew.id}/maintenance-status`, { token: A() });
    check('VH-10', s2.data?.isDue === false, `Kendaraan baru (odometer 15.000) tidak langsung jatuh tempo servis`,
      `Kendaraan baru didaftarkan di 15.000 km langsung "jatuh tempo" (baseline ${s2.data?.lastMaintenanceOdometer}, sisa ${s2.data?.kmUntilDue} km) → isi BBM pertama memicu maintenance`);
  });

  // ── VH ────────────────────────────────────────────────────────────────
  await scenario('VH-02', async () => {
    const d = await newDriver('DRVVH2'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(120), { driverId: d.driverId });
    await startB(b.id, d.token);
    const r = await api('PATCH', `/vehicles/${v.id}/status`, { token: A(), body: { status: 'AVAILABLE' } });
    check('VH-02', r.status >= 400, `Ubah manual ke AVAILABLE saat IN_USE ditolak (${r.status})`,
      `Kendaraan sedang dipakai trip, admin ubah manual ke AVAILABLE → ${r.status}, status kini ${resourceStatus(v.resourceId)}`);
  });
  await scenario('VH-04/05', async () => {
    const v = await newVehicle({ odometer: 10000 });
    const r4 = await api('PUT', `/vehicles/${v.id}`, { token: A(), body: vehicleBody(v, 9000) });
    check('VH-04', r4.status === 400, `Odometer diturunkan → ${r4.status} "${r4.msg}"`);
    const r5 = await api('PUT', `/vehicles/${v.id}`, { token: A(), body: vehicleBody(v, 20500) });
    const n = sqlInt(`select count(*) from maintenance_records where "vehicleId"=${v.id} and "isAutoGenerated"`);
    check('VH-05', ok2(r5) && n === 1, `Odometer dinaikkan melewati interval → maintenance otomatis ${n}`);
  });
  await scenario('VH-06/07', async () => {
    const v = await newVehicle();
    const r6 = await api('POST', '/vehicles', { token: A(), body: { ...vehicleBody(v, 1000), name: 'Duplikat' } });
    check('VH-06', r6.status === 409, `Plat duplikat → ${r6.status}`);
    const d = await newDriver('DRVVH7');
    await book(U.EMPA.token, v.resourceId, wib(3, 9), wib(3, 10), { driverId: d.driverId });
    const r7 = await api('DELETE', `/vehicles/${v.id}`, { token: A() });
    check('VH-07', r7.status >= 400 && r7.status < 500, `Hapus kendaraan berriwayat booking → ${r7.status} "${r7.msg}"`,
      `Hapus kendaraan berriwayat booking → ${r7.status} "${r7.msg}" (error server, bukan pesan jelas)`);
  });
  record('VH-01', 'FAIL', 'Lihat BC-08: kendaraan INACTIVE masih bisa dibooking');

  // ── RM ────────────────────────────────────────────────────────────────
  await scenario('RM-01', async () => {
    const room = await newRoom({ keeperId: U.RK2.rkId });
    const g = await api('GET', `/rooms/${room.id}`, { token: A() });
    const k = await api('GET', `/room-keepers/${U.RK2.rkId}`, { token: A() });
    const inKeeper = JSON.stringify(k.data ?? {}).includes(room.name);
    check('RM-01', JSON.stringify(g.data ?? {}).includes(U.RK2.name) && inKeeper, `Penjaga tampil di ruangan & ruangan tampil di penjaga (${inKeeper})`);
  });
  await scenario('RM-02', async () => {
    const rk = await createUser('RKOFF', ROLE.ROOM_KEEPER);
    const room = await newRoom({ keeperId: rk.rkId });
    const b = await bookApproved(U.EMPA.token, room.resourceId, inMin(5), inMin(60));
    await api('PATCH', `/room-keepers/${rk.rkId}/toggle-active`, { token: A() });
    const s = await startB(b.id, rk.token);
    check('RM-02', s.status === 403, `Penjaga nonaktif memulai booking ruangan → ${s.status}`);
  });
  await scenario('RM-03', async () => {
    const room = await newRoom({ keeperId: U.RK1.rkId });
    const r = await api('PATCH', `/rooms/${room.id}/status`, { token: U.RK1.token, body: { status: 'INACTIVE' } });
    check('RM-03', ok2(r), `Penjaga ruangan mengubah status ruangan → ${r.status}`,
      `Penjaga ruangan mengubah status ruangan → ${r.status} (mobile menampilkan tombolnya untuk ROOM_KEEPER)`);
  });

  // ── DU ────────────────────────────────────────────────────────────────
  await scenario('DU-01/02', async () => {
    const r = await api('POST', '/users', { token: A(), body: { employeeId: `X-${Date.now()}`, name: 'Supir Tanpa SIM', email: `nosim.${Date.now()}@kce-test.local`, password: PASSWORD, roleId: ROLE.DRIVER, departmentId: 1 } });
    check('DU-01', r.status === 400, `Buat supir tanpa SIM/telepon → ${r.status} "${r.msg}"`);
    const d = await newDriver('DRVDU2');
    const av = await api('GET', `/drivers/available?startDate=${encodeURIComponent(wib(5, 9))}&endDate=${encodeURIComponent(wib(5, 10))}`, { token: A() });
    check('DU-02', (av.data ?? []).some((x) => x.driverId === d.driverId), `Supir baru muncul di picker`);
  });
  await scenario('DU-04/05', async () => {
    const d = await newDriver('DRVDU4');
    const avail = async () => ((await api('GET', `/drivers/available?startDate=${encodeURIComponent(wib(5, 9))}&endDate=${encodeURIComponent(wib(5, 10))}`, { token: A() })).data ?? []).some((x) => x.driverId === d.driverId);
    await api('PATCH', `/users/${d.id}/toggle-active`, { token: A() });
    const off = await avail();
    const l = await api('POST', '/auth/login', { body: { email: d.email, password: PASSWORD } });
    check('DU-04', !off && l.status === 403, `Akun supir dinonaktifkan: di picker=${off}, login=${l.status}`);
    await api('PATCH', `/users/${d.id}/toggle-active`, { token: A() });
    check('DU-05', await avail(), `Diaktifkan kembali → muncul lagi di picker`);
  });
  await scenario('DU-06', async () => {
    const r = await api('DELETE', `/users/${U.EMPA.id}`, { token: A() });
    check('DU-06', r.status === 409, `Hapus pengguna berriwayat booking → ${r.status} "${r.msg}"`);
  });
  await scenario('DU-07', async () => {
    const u = await createUser('EMPTODRV', ROLE.EMPLOYEE);
    const base = { name: u.name, email: u.email, employeeId: `E-${Date.now()}`, roleId: ROLE.DRIVER, departmentId: 1 };
    const r1 = await api('PUT', `/users/${u.id}`, { token: A(), body: base });
    const r2 = await api('PUT', `/users/${u.id}`, { token: A(), body: { ...base, licenseNumber: 'SIM-X', phoneNumber: '0812' } });
    const drv = sqlInt(`select id from drivers where "userId"=${u.id}`);
    check('DU-07', r1.status === 400 && ok2(r2) && drv, `Ubah ke DRIVER tanpa SIM → ${r1.status}; dengan SIM → ${r2.status}, data supir dibuat=${!!drv}`);
  });
  await scenario('DU-08', async () => {
    const d = await newDriver('DRVDU8'); const v = await newVehicle();
    await api('POST', `/drivers/${d.driverId}/assign`, { token: A(), body: { vehicleId: v.id } });
    const h1 = heldVehicle(d.driverId);
    await api('PATCH', `/drivers/${d.driverId}/release`, { token: A() });
    const h2 = heldVehicle(d.driverId);
    check('DU-08', h1 === v.id && h2 === null, `Tugaskan manual → memegang ${h1}; lepas → ${h2}`);
  });
  await scenario('DU-03', async () => {
    const d = await newDriver('DRVDU3'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, wib(6, 9), wib(6, 10), { driverId: d.driverId });
    await api('PATCH', `/drivers/${d.driverId}/toggle-active`, { token: A() });
    const g = (await getBooking(b.id)).data;
    record('DU-03', 'INFO', `Supir dinonaktifkan padahal punya booking APPROVED → booking tetap ditugaskan ke supir nonaktif (${g.assignedDriver?.name}), tanpa peringatan`);
  });

  // ── FL ────────────────────────────────────────────────────────────────
  await scenario('FL-01/02/03', async () => {
    const d = await newDriver('DRVFL1'); const v = await newVehicle({ odometer: 10000 });
    const f = await fuel(d.token, v, 10000, 10100);
    const odo = sqlInt(`select "currentOdometer" from vehicles where id=${v.id}`);
    check('FL-01', f.status === 201 && odo === 10100 && Number(f.data?.totalCost) === 100000 && f.changed === 'fuel',
      `Tercatat, odometer ${odo}, biaya ${f.data?.totalCost} (10 L × harga master), X-Data-Changed=${f.changed}`);
    const f2 = await fuel(d.token, v, 9000, 9100);
    check('FL-02', f2.status === 400, `Odometer sebelum < tercatat → ${f2.status}`);
    const f3 = await fuel(d.token, v, 10100, 10100);
    check('FL-03', f3.status === 400, `Odometer sesudah ≤ sebelum → ${f3.status}`);
  });
  await scenario('FL-04/06', async () => {
    const d = await newDriver('DRVFL4'); const v = await newVehicle({ odometer: 10000, energy: 'BBM' });
    const f = await fuel(d.token, v, 10000, 10050, { fuelTypeId: 4, liter: 0, kwh: 20 });
    record('FL-04', 'INFO', `Isi "Listrik PLN" untuk kendaraan BBM lewat API → ${f.status} (UI mengunci; API tidak memvalidasi)`);
    const f2 = await fuel(d.token, v, 10050, 10200);
    await api('DELETE', `/fuel-expenses/${f2.data.id}`, { token: A() });
    const odo = sqlInt(`select "currentOdometer" from vehicles where id=${v.id}`);
    record('FL-06', 'INFO', `Catatan BBM dihapus → odometer kendaraan tetap ${odo} (tidak mundur)`);
  });

  // ── SY (header X-Data-Changed; WebSocket diuji di browser) ────────────
  await scenario('SY-HDR', async () => {
    const v = await newVehicle(); const room = await newRoom();
    const cases = [
      ['PATCH status kendaraan', await api('PATCH', `/vehicles/${v.id}/status`, { token: A(), body: { status: 'INACTIVE' } }), 'vehicle'],
      ['PATCH status ruangan', await api('PATCH', `/rooms/${room.id}/status`, { token: A(), body: { status: 'INACTIVE' } }), 'room'],
      ['PATCH toggle user', await api('PATCH', `/users/${U.RK2.id}/toggle-active`, { token: A() }), 'user,driver,roomKeeper'],
      ['GET daftar kendaraan', await api('GET', '/vehicles', { token: A() }), null],
      ['PATCH baca semua notifikasi', await api('PATCH', '/users/me/notifications/read-all', { token: A() }), null],
      ['POST login gagal', await api('POST', '/auth/login', { body: { email: U.EMPA.email, password: 'salah12345' } }), null],
    ];
    await api('PATCH', `/users/${U.RK2.id}/toggle-active`, { token: A() }); // kembalikan
    const bad = cases.filter(([, r, want]) => (r.changed ?? null) !== want).map(([n, r]) => `${n}=${r.changed}`);
    check('SY-HDR', bad.length === 0, `Header X-Data-Changed tepat untuk ${cases.length} jenis request`, `Header salah: ${bad.join('; ')}`);
  });

  // ── TZ ────────────────────────────────────────────────────────────────
  await scenario('TZ-03', async () => {
    const d = await newDriver('DRVTZ'); const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(52, 0, 30), wib(52, 2, 0), { driverId: d.driverId });
    const same = Date.parse(b.data.startDate) === Date.parse(wib(52, 0, 30));
    const dayWib = sql(`select to_char("startDate" at time zone 'Asia/Jakarta','YYYY-MM-DD HH24:MI') from bookings where id=${b.data.id}`);
    check('TZ-03', same && dayWib.endsWith('00:30'), `Booking 00:30 WIB tersimpan ${dayWib} WIB, dikembalikan sebagai ${b.data.startDate}`);
  });
  await scenario('TZ-04', async () => {
    const n = (await api('GET', '/users/me/notifications?limit=1', { token: A() })).data?.[0];
    const diffMin = Math.abs(Date.now() - Date.parse(n.createdAt)) / 60000;
    check('TZ-04', diffMin < 30, `Waktu notifikasi terbaru selisih ${diffMin.toFixed(1)} menit dari sekarang (tidak bergeser 7 jam)`);
  });
  record('TZ-05..07', 'SKIP', 'Batas periode laporan WIB dicakup unit test backend (TestPeriodBoundsUseWIBCalendar); tampilan diuji di browser');

  // ── DL ────────────────────────────────────────────────────────────────
  await scenario('DL-01', async () => {
    const s = (await api('GET', '/dashboard/summary', { token: A() })).json ?? {};
    const db = {
      available_vehicles: sqlInt(`select count(*) from vehicles v join resources r on r.id=v."resourceId" where r.status='AVAILABLE'`),
      available_rooms: sqlInt(`select count(*) from rooms m join resources r on r.id=m."resourceId" where r.status='AVAILABLE'`),
      available_drivers: sqlInt(`select count(*) from drivers d join users u on u.id=d."userId" where d."isActive" and u."isActive" and not exists (select 1 from driver_assignments a where a."driverId"=d.id and a."releasedAt" is null)`),
    };
    const diff = Object.entries(db).filter(([k, v]) => s[k] !== v).map(([k, v]) => `${k}: dashboard ${s[k]} vs nyata ${v}`);
    check('DL-01', diff.length === 0, `Angka dashboard sama dengan data nyata (${JSON.stringify(s)})`, `Beda: ${diff.join('; ')}`);
  });
  await scenario('DL-05', async () => {
    const rows = sql(`select count(*) filter (where "ipAddress" is null), count(*) from audit_logs where "userId" is not null and "createdAt" > now() - interval '2 hours'`).split('|').map(Number);
    check('DL-05', rows[1] > 0 && rows[0] === 0, `Audit log aksi manusia: ${rows[1]} baris, tanpa IP: ${rows[0]}`);
  });
  await scenario('AP-15', async () => {
    const approved = sqlInt(`select count(*) from bookings where "approvedAt" is not null`);
    const logs = sqlInt(`select count(*) from approval_logs where action='APPROVED'`);
    check('AP-15', logs > 0, `Persetujuan tercatat di log persetujuan (${logs}/${approved})`,
      `${approved} booking disetujui, tapi log persetujuan APPROVE = ${logs} (insert "APPROVE" ditolak enum approval_action)`);
  });
}
