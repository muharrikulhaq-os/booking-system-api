// Batch 4: MT, VH, RM, DU, FL, SY (header), TZ, DL
import { api, apiRaw, ok2, sql, sqlInt, record, check, scenario, wib, inMin, fakePhoto, PASSWORD } from './lib.mjs';
import {
  U, ROLE, createUser, newVehicle, newRoom, newDriver, book, getBooking, approve, startB, completeB,
  bookApproved, heldVehicle, resourceStatus, setVehicleStatus, workshop, maintPlan, maintActive,
} from './fixtures.mjs';

const A = () => U.ADM.token;
const mApi = (method, path, body) => api(method, `/maintenance${path}`, { token: A(), body });
const fuel = (token, v, before, after, extra = {}) => api('POST', '/fuel-expenses', {
  token, form: { vehicleId: v.id, fuelTypeId: 1, liter: 10, odometerBefore: before, odometerAfter: after, proofPhoto: fakePhoto(), ...extra } });
const vehicleBody = (v, odometer) => ({ name: `Mobil ${v.plate}`, plateNumber: v.plate, brand: 'Toyota', model: 'Uji', year: 2024, currentOdometer: odometer, categoryId: 1, capacity: 6 });
const odo = (v) => sqlInt(`select "currentOdometer" from vehicles where id=${v.id}`);

export async function runBatch4() {
  // ── MT (maintenance oleh vendor — docs/RANCANGAN_MAINTENANCE_VENDOR.md) ──
  await scenario('MT-01/02/03', async () => {
    const v = await newVehicle();
    const w = await workshop();
    const d = await mApi('POST', '', { vehicleId: v.id, category: 'REPAIR', description: 'Rem bunyi' });
    const bDraft = await book(U.EMPA.token, v.resourceId, wib(2, 9), wib(2, 10));
    check('MT-01', d.status === 201 && d.data.status === 'DRAFT' && !d.data.requestNo && resourceStatus(v.resourceId) === 'AVAILABLE'
      && bDraft.status === 201 && (d.changed ?? '').includes('maintenance'),
      `Draf → ${d.data?.status}, tanpa nomor surat, kendaraan ${resourceStatus(v.resourceId)}, booking tetap bisa (${bDraft.status}), X-Data-Changed=${d.changed}`);
    const noVendor = await mApi('POST', `/${d.data.id}/submit`);
    await mApi('PUT', `/${d.data.id}`, { vehicleId: v.id, vendorId: w.id, category: 'REPAIR', description: 'Rem bunyi', plannedDate: wib(3, 8), estimatedDays: 2 });
    const s = await mApi('POST', `/${d.data.id}/submit`);
    const inWin = await book(U.EMPA.token, v.resourceId, wib(4, 9), wib(4, 10));
    check('MT-02', noVendor.status === 400 && ok2(s) && s.data.status === 'SUBMITTED' && /^\d{3}\/[A-Z-]+\/[IVX]+\/\d{4}$/.test(s.data.requestNo ?? '')
      && inWin.status === 409,
      `Ajukan tanpa vendor → ${noVendor.status}; diajukan → ${s.data?.status} nomor ${s.data?.requestNo}; booking di jendela rencana → ${inWin.status}`);
    const dup = await mApi('POST', '', { vehicleId: v.id, category: 'ROUTINE', description: 'Servis' });
    check('MT-03', dup.status === 409, `Pengajuan kedua saat masih ada yang berjalan → ${dup.status} "${dup.msg}"`);
  });
  await scenario('MT-04', async () => {
    const d = await newDriver('DRVMT4'); const v = await newVehicle();
    await bookApproved(U.EMPA.token, v.resourceId, wib(60, 9), wib(60, 12), { driverId: d.driverId });
    const m = await maintPlan(v.id, wib(60, 7), 1);
    check('MT-04', m.status === 201 && !!m.json?.warning, `Diajukan bentrok dengan booking disetujui → dibuat + peringatan "${m.json?.warning ?? '-'}"`);
  });
  await scenario('MT-05', async () => {
    const v = await newVehicle();
    const m = await maintPlan(v.id, wib(20, 8), 1);
    const sc = await mApi('POST', `/${m.data.id}/schedule`, { scheduledDate: wib(22, 8), estimatedDays: 1, note: 'Konfirmasi via telepon' });
    const oldDay = await book(U.EMPA.token, v.resourceId, wib(20, 9), wib(20, 10));
    const newDay = await book(U.EMPA.token, v.resourceId, wib(22, 9), wib(22, 10));
    check('MT-05', ok2(sc) && sc.data.status === 'SCHEDULED' && oldDay.status === 201 && newDay.status === 409 && resourceStatus(v.resourceId) === 'AVAILABLE',
      `Jadwal vendor → ${sc.data?.status}; tanggal rencana lama bisa dibooking (${oldDay.status}), tanggal jadwal diblokir (${newDay.status}); kendaraan ${resourceStatus(v.resourceId)}`);
  });
  await scenario('MT-06/07/08', async () => {
    const d = await newDriver('DRVMT6'); const v = await newVehicle({ odometer: 30000 });
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(120), { driverId: d.driverId });
    await startB(b.id, d.token);
    const m = await maintPlan(v.id, inMin(-5), 2);
    const early = await mApi('POST', `/${m.data.id}/handover`, { receiverName: 'Bengkel' });
    check('MT-07', early.status === 409, `Serah terima saat kendaraan masih dipakai trip → ${early.status} "${early.msg}"`);
    await completeB(b.id);
    const lowOdo = await mApi('POST', `/${m.data.id}/handover`, { receiverName: 'Andi', odometer: 29000 });
    const h = await mApi('POST', `/${m.data.id}/handover`, { receiverName: 'Andi', odometer: 30150, fuelLevel: '1/2',
      checklist: { stnk: true, mainKey: true, spareTire: true }, note: 'Baret bumper belakang' });
    const far = await book(U.EMPA.token, v.resourceId, wib(90, 9), wib(90, 10));
    check('MT-06', lowOdo.status === 400 && ok2(h) && h.data.status === 'IN_PROGRESS' && resourceStatus(v.resourceId) === 'MAINTENANCE'
      && odo(v) === 30150 && far.status === 409 && h.data.handover?.checklist?.stnk === true,
      `Odometer mundur → ${lowOdo.status}; diserahkan → ${h.data?.status}, kendaraan ${resourceStatus(v.resourceId)}, odometer ${odo(v)}, booking tanggal jauh ${far.status}`);
    const cancelInProg = await mApi('POST', `/${m.data.id}/cancel`, { reason: 'uji' });
    const badTime = await mApi('POST', `/${m.data.id}/return`, { returnedAt: inMin(-120), workDone: 'x' });
    const noWork = await mApi('POST', `/${m.data.id}/return`, { odometer: 30170 });
    const r = await mApi('POST', `/${m.data.id}/return`, { odometer: 30170, fuelLevel: '1/4', handlerName: 'Andi',
      workDone: 'Ganti kampas rem', partsReplaced: 'Kampas rem 1 set', actualCost: 1250000, costBearer: 'COMPANY',
      checklist: { stnk: true, mainKey: true, spareTire: true } });
    const again = await book(U.EMPA.token, v.resourceId, wib(90, 9), wib(90, 10));
    check('MT-08', cancelInProg.status === 409 && badTime.status === 400 && noWork.status === 422 && ok2(r) && r.data.status === 'COMPLETED'
      && resourceStatus(v.resourceId) === 'AVAILABLE' && odo(v) === 30170 && r.data.actualCost === 1250000 && again.status === 201,
      `Batal saat dikerjakan ${cancelInProg.status}; kembali sebelum serah terima ${badTime.status}; tanpa uraian pekerjaan ${noWork.status}; `
      + `diterima kembali → ${r.data?.status}, kendaraan ${resourceStatus(v.resourceId)}, odometer ${odo(v)}, biaya ${r.data?.actualCost}, bisa dibooking ${again.status}`);
    const pdfs = {};
    for (const k of ['request', 'handover', 'return']) {
      const p = await apiRaw('GET', `/maintenance/${m.data.id}/pdf/${k}?download=1`, { token: A() });
      pdfs[k] = p.status === 200 && p.contentType.includes('application/pdf') && p.bytes.subarray(0, 4).toString() === '%PDF' && p.disposition.includes('attachment');
    }
    check('MT-12', Object.values(pdfs).every(Boolean), `PDF surat/BA serah terima/BA pengembalian: ${JSON.stringify(pdfs)}`);
    const cost = await mApi('PATCH', `/${m.data.id}/cost`, { estimatedCost: 1500000, actualCost: 1300000, costBearer: 'VENDOR' });
    const up = await api('POST', `/maintenance/${m.data.id}/documents`, { token: A(), form: { kind: 'INVOICE', file: fakePhoto() } });
    const docId = up.data?.[0]?.id;
    const del = await api('DELETE', `/maintenance/${m.data.id}/documents/${docId}`, { token: A() });
    const g = await mApi('GET', `/${m.data.id}`);
    check('MT-14', ok2(cost) && cost.data.actualCost === 1300000 && up.status === 201 && ok2(del) && (g.data?.documents ?? []).length === 0,
      `Biaya diubah setelah selesai (${cost.status}, ${cost.data?.actualCost}); unggah invoice ${up.status}; hapus ${del.status}`);
  });
  await scenario('MT-09/10', async () => {
    const v = await newVehicle();
    const m = await maintPlan(v.id, wib(30, 8), 1);
    const noReason = await mApi('POST', `/${m.data.id}/cancel`, {});
    const delSubmitted = await mApi('DELETE', `/${m.data.id}`);
    const c = await mApi('POST', `/${m.data.id}/cancel`, { reason: 'Vendor penuh' });
    const b = await book(U.EMPA.token, v.resourceId, wib(30, 9), wib(30, 10));
    check('MT-09', noReason.status === 400 && ok2(c) && c.data.status === 'CANCELLED' && b.status === 201,
      `Batal tanpa alasan ${noReason.status}; dibatalkan → ${c.data?.status}; tanggal kembali bisa dibooking (${b.status})`);
    const draft = await mApi('POST', '', { vehicleId: v.id, category: 'OTHER', description: 'Draf' });
    const pdfDraft = await apiRaw('GET', `/maintenance/${draft.data.id}/pdf/request`, { token: A() });
    const delDraft = await mApi('DELETE', `/${draft.data.id}`);
    check('MT-10', delSubmitted.status === 409 && ok2(delDraft) && pdfDraft.status === 409,
      `Hapus pengajuan resmi → ${delSubmitted.status}; PDF draf → ${pdfDraft.status}; hapus draf → ${delDraft.status}`);
  });
  await scenario('MT-11/FL-05', async () => {
    const d = await newDriver('DRVMT11'); const v = await newVehicle({ odometer: 10000 });
    const f = await fuel(d.token, v, 10000, 30100);
    const n = sqlInt(`select count(*) from maintenance_records where "vehicleId"=${v.id}`);
    check('MT-11', ok2(f) && n === 0 && resourceStatus(v.resourceId) === 'AVAILABLE',
      `Isi BBM +20.100 km → tidak ada maintenance otomatis (${n}), kendaraan ${resourceStatus(v.resourceId)}`);
    record('FL-05', n === 0 ? 'PASS' : 'FAIL', 'Sama dengan MT-11');
  });
  await scenario('MT-13', async () => {
    const owner = await api('POST', '/vendors', { token: A(), body: { name: `Rental Uji ${Date.now()}`, type: 'OWNER' } });
    const v = await newVehicle();
    const w = await workshop();
    const setOwn = await api('PUT', `/vehicles/${v.id}`, { token: A(), body: { ...vehicleBody(v, 10000), ownership: 'VENDOR', ownerVendorId: owner.data.id, rentalContractNo: 'K-01' } });
    const m = await maintPlan(v.id, wib(40, 8), 1, { vendorId: w.id });
    check('MT-13', ok2(setOwn) && setOwn.data.ownership === 'VENDOR' && m.data?.vendor?.id === owner.data.id,
      `Kendaraan sewa → tujuan surat otomatis vendor pemilik (${m.data?.vendor?.name}), bukan bengkel yang dipilih`);
  });
  await scenario('MT-15', async () => {
    const a = await maintPlan((await newVehicle()).id, wib(41, 8));
    const b = await maintPlan((await newVehicle()).id, wib(41, 8));
    const n = (x) => Number((x.data?.requestNo ?? '').split('/')[0]);
    check('MT-15', n(b) === n(a) + 1, `Nomor surat berurutan: ${a.data?.requestNo} → ${b.data?.requestNo}`);
  });
  await scenario('MT-17', async () => {
    const v = await newVehicle();
    const s = await api('GET', `/vehicles/${v.id}/maintenance-status`, { token: A() });
    check('MT-17', s.status === 404, `Endpoint pengingat sisa km sudah dihapus (${s.status})`);
  });
  await scenario('MT-18', async () => {
    const v = await newVehicle();
    await maintActive(v.id);
    // Jaring pengaman: status kendaraan terlanjur AVAILABLE padahal sedang di vendor.
    sql(`update resources set status='AVAILABLE' where id=${v.resourceId}`);
    const g = await api('GET', `/vehicles/${v.id}`, { token: A() });
    check('MT-18', ok2(g) && resourceStatus(v.resourceId) === 'MAINTENANCE',
      `Kendaraan di vendor tapi tercatat AVAILABLE → ${resourceStatus(v.resourceId)} saat detail dibuka`);
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
  await scenario('VH-11', async () => {
    const v = await newVehicle();
    await maintActive(v.id);
    const r = await api('PATCH', `/vehicles/${v.id}/status`, { token: A(), body: { status: 'AVAILABLE' } });
    const st = resourceStatus(v.resourceId);
    const r2 = await api('PATCH', `/vehicles/${v.id}/status`, { token: A(), body: { status: 'INACTIVE' } });
    check('VH-11', r.status === 409 && st === 'MAINTENANCE' && ok2(r2),
      `Kendaraan di vendor: set AVAILABLE → ${r.status} (status ${st}); set INACTIVE → ${r2.status}`);
  });
  await scenario('VH-04/05', async () => {
    const v = await newVehicle({ odometer: 10000 });
    const r4 = await api('PUT', `/vehicles/${v.id}`, { token: A(), body: vehicleBody(v, 9000) });
    check('VH-04', r4.status === 400, `Odometer diturunkan → ${r4.status} "${r4.msg}"`);
    const r5 = await api('PUT', `/vehicles/${v.id}`, { token: A(), body: vehicleBody(v, 20500) });
    const n = sqlInt(`select count(*) from maintenance_records where "vehicleId"=${v.id}`);
    check('VH-05', ok2(r5) && n === 0, `Odometer dinaikkan +10.500 km → tidak ada maintenance otomatis (${n})`);
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
  await scenario('VH-01', async () => {
    const v = await newVehicle();
    const s = await api('PATCH', `/vehicles/${v.id}/status`, { token: A(), body: { status: 'INACTIVE' } });
    const b = await book(U.EMPA.token, v.resourceId, wib(3, 9), wib(3, 10));
    check('VH-01', ok2(s) && b.status === 409, `Kendaraan diubah INACTIVE (${s.status}) → booking baru ditolak (${b.status})`);
  });

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
    const t = await api('PATCH', `/drivers/${d.driverId}/toggle-active`, { token: A() });
    const u = await api('PATCH', `/users/${d.id}/toggle-active`, { token: A() });
    check('DU-03', t.status === 409 && u.status === 409,
      `Supir punya booking APPROVED → nonaktifkan ditolak (supir ${t.status}, akun ${u.status}) "${t.msg}"`);
  });

  // ── FL ────────────────────────────────────────────────────────────────
  await scenario('FL-01/02/03', async () => {
    const d = await newDriver('DRVFL1'); const v = await newVehicle({ odometer: 10000 });
    const f = await fuel(d.token, v, 10000, 10100);
    const odo = sqlInt(`select "currentOdometer" from vehicles where id=${v.id}`);
    // Saldo BBM: tulis BBM ikut mengubah kendaraan (odometer/saldo) → fuel,vehicle.
    check('FL-01', f.status === 201 && odo === 10100 && Number(f.data?.totalCost) === 100000
      && (f.changed ?? '').split(',').includes('fuel'),
      `Tercatat, odometer ${odo}, biaya ${f.data?.totalCost} (10 L × harga master), X-Data-Changed=${f.changed}`);
    const f2 = await fuel(d.token, v, 9000, 9100);
    check('FL-02', f2.status === 400, `Odometer sebelum < tercatat → ${f2.status}`);
    // FL-03 (SKENARIO_TESTING §19): odometer saat isi kosong → ditolak.
    const f3 = await api('POST', '/fuel-expenses', { token: d.token,
      form: { vehicleId: v.id, fuelTypeId: 1, liter: 10, proofPhoto: fakePhoto() } });
    check('FL-03', f3.status === 400, `Odometer saat isi kosong → ${f3.status} "${f3.msg}"`);
  });
  await scenario('FL-04/06', async () => {
    const d = await newDriver('DRVFL4'); const v = await newVehicle({ odometer: 10000, energy: 'BBM' });
    const f = await fuel(d.token, v, 10000, 10050, { fuelTypeId: 4, liter: 0, kwh: 20 });
    check('FL-04', f.status === 400, `Isi "Listrik PLN" untuk kendaraan BBM lewat API → ${f.status} "${f.msg}"`);
    const f2 = await fuel(d.token, v, 10050, 10200);
    await api('DELETE', `/fuel-expenses/${f2.data.id}`, { token: A() });
    const odo = sqlInt(`select "currentOdometer" from vehicles where id=${v.id}`);
    record('FL-06', 'INFO', `Catatan BBM dihapus → odometer kendaraan tetap ${odo} (tidak mundur) — dikaji ulang bersama fitur voucher BBM`);
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
