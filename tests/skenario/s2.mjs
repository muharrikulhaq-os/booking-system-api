// Batch 2: AP, SB, AS, CN
import { api, ok2, sql, record, check, scenario, wib, inMin } from './lib.mjs';
import {
  U, newVehicle, newRoom, newDriver, book, getBooking, approve, startB, completeB, cancelB,
  hasNotif, onlyDrivers, bookApproved, heldVehicle, resourceStatus,
} from './fixtures.mjs';

const activity = async (id) => (await api('GET', `/bookings/${id}/activity`, { token: U.ADM.token })).data ?? [];
const assign = (id, driverId, vehicleId) => api('POST', `/bookings/${id}/assign-vehicle`, { token: U.ADM.token, body: { driverId, vehicleId } });
const substitute = (id, resourceId) => api('PATCH', `/bookings/${id}/substitute-resource`, { token: U.ADM.token, body: { resourceId, note: 'Uji alihkan' } });
const maint = (vehicleId, start, end) => api('POST', '/maintenance', { token: U.ADM.token,
  body: { vehicleId, type: 'REPAIR', status: 'pending', description: 'Uji', location: 'Bengkel', startDate: start, endDate: end } });

export async function runBatch2() {
  // ── AP ────────────────────────────────────────────────────────────────
  await scenario('AP-01/02', async () => {
    const d = await newDriver('DRVAP1'); const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(20, 9), wib(20, 12), { driverId: d.driverId });
    const a = await approve(b.data.id);
    const g = (await getBooking(b.data.id)).data;
    const held = heldVehicle(d.driverId);
    const nEmp = await hasNotif(U.EMPA.token, 'BOOKING_APPROVED', b.data.id);
    const nDrv = await hasNotif(d.token, 'NEW_BOOKING', b.data.id);
    check('AP-01', ok2(a) && g.status === 'APPROVED' && held === v.id && nEmp && nDrv && a.changed === 'booking',
      `APPROVED, supir memegang kendaraan=${held === v.id}, notif pemohon=${nEmp}, notif supir=${nDrv}, X-Data-Changed=${a.changed}`);
    const again = await approve(b.data.id);
    check('AP-02', again.status >= 400, `Approve ulang → ${again.status} "${again.msg}"`);
  });
  await scenario('AP-03', async () => {
    const d = await newDriver('DRVAP3'); const v = await newVehicle();
    await bookApproved(U.EMPA.token, v.resourceId, wib(21, 9), wib(21, 12), { driverId: d.driverId });
    const b2 = await book(U.EMPB.token, v.resourceId, wib(21, 10), wib(21, 11), { driverId: d.driverId });
    const r = await approve(b2.data.id);
    check('AP-03', r.status === 409, `Kendaraan sudah APPROVED di jam bentrok → ${r.status} "${r.msg}"`);
  });
  await scenario('AP-04', async () => {
    const d1 = await newDriver('DRVAP4A'); const d2 = await newDriver('DRVAP4B'); const v = await newVehicle();
    await bookApproved(U.EMPA.token, v.resourceId, wib(22, 8), wib(22, 10), { driverId: d1.driverId });
    const b2 = await book(U.EMPB.token, v.resourceId, wib(23, 8), wib(23, 10), { driverId: d2.driverId });
    const r = await approve(b2.data.id);
    check('AP-04', r.status === 409, `Kendaraan dipegang supir lain (booking hari lain) → ${r.status} "${r.msg}"`);
  });
  await scenario('AP-05', async () => {
    const d = await newDriver('DRVAP5'); const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(24, 9), wib(24, 12), { driverId: d.driverId });
    await maint(v.id, wib(24, 7), wib(24, 17));
    const r = await approve(b.data.id);
    check('AP-05', r.status === 409, `Maintenance dijadwalkan setelah booking PENDING → approve ${r.status} "${r.msg}"`);
  });
  await scenario('AP-07', async () => {
    const room = await newRoom({ keeperId: U.RK1.rkId });
    const b = await book(U.EMPA.token, room.resourceId, wib(20, 13), wib(20, 14));
    await approve(b.data.id);
    check('AP-07', await hasNotif(U.RK1.token, 'ROOM_BOOKED', b.data.id), `Penjaga ruangan dapat notifikasi "Ruangan dipesan"`);
  });
  await scenario('AP-08/09', async () => {
    const d = await newDriver('DRVAP8'); const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(25, 9), wib(25, 10), { driverId: d.driverId });
    const r0 = await api('POST', `/bookings/${b.data.id}/reject`, { token: U.ADM.token, body: {} });
    check('AP-08', r0.status >= 400, `Tolak tanpa catatan → ${r0.status}`);
    const r = await api('POST', `/bookings/${b.data.id}/reject`, { token: U.ADM.token, body: { note: 'Kendaraan dipakai direksi' } });
    const g = (await getBooking(b.data.id)).data;
    const n = await hasNotif(U.EMPA.token, 'BOOKING_REJECTED', b.data.id);
    const act = (await activity(b.data.id)).some((a) => a.action === 'REJECT');
    check('AP-09', ok2(r) && g.status === 'REJECTED' && n && act, `REJECTED, notif pemohon=${n}, timeline REJECT=${act}`);
  });
  await scenario('AP-10/11', async () => {
    const d = await newDriver('DRVAP10'); const v = await newVehicle();
    const own1 = await book(U.ADM.token, v.resourceId, wib(26, 9), wib(26, 10), { driverId: d.driverId });
    const rj = await api('POST', `/bookings/${own1.data.id}/reject`, { token: U.ADM.token, body: { note: 'x' } });
    check('AP-10', rj.status >= 400, `Admin menolak booking sendiri → ${rj.status} "${rj.msg}"`);
    const own2 = await book(U.ADM.token, v.resourceId, wib(26, 11), wib(26, 12), { driverId: d.driverId });
    const ap = await approve(own2.data.id);
    check('AP-11', ap.status >= 400, `Admin menyetujui booking sendiri ditolak (${ap.status})`,
      `Admin BISA menyetujui booking sendiri (${ap.status}) padahal menolaknya dilarang`);
  });
  await scenario('AP-12', async () => {
    const d = await newDriver('DRVAP12'); const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(27, 9), wib(27, 10), { driverId: d.driverId });
    const [x, y] = await Promise.all([approve(b.data.id, U.ADM.token), approve(b.data.id, U.ADM2.token)]);
    const logs = sql(`select count(*) from approval_logs where "bookingId"=${b.data.id}`);
    check('AP-12', [x, y].filter(ok2).length === 1 && logs === '1', `Dua admin approve bersamaan → ${x.status}, ${y.status}; log persetujuan=${logs}`,
      `Dua admin approve bersamaan → ${x.status}, ${y.status}; log persetujuan tercatat ${logs}x`);
  });
  await scenario('AP-14', async () => {
    const v = await newVehicle();
    const b = await onlyDrivers([], () => book(U.EMPA.token, v.resourceId, wib(28, 9), wib(28, 10)));
    const a = await approve(b.data.id);
    const g = (await getBooking(b.data.id)).data;
    check('AP-14', ok2(a) && g.status === 'APPROVED' && !g.assignedDriver, `Booking tanpa supir bisa disetujui (menunggu penugasan)`);
  });

  // ── SB ────────────────────────────────────────────────────────────────
  await scenario('SB-01/02/03', async () => {
    const d = await newDriver('DRVSB1'); const va = await newVehicle(); const vb = await newVehicle();
    const b = await book(U.EMPA.token, va.resourceId, wib(30, 9), wib(30, 12), { driverId: d.driverId });
    const s = await substitute(b.data.id, vb.resourceId);
    const a = await approve(b.data.id);
    const g = (await getBooking(b.data.id)).data;
    const n = await hasNotif(U.EMPA.token, 'BOOKING_SUBSTITUTED', b.data.id);
    check('SB-01', ok2(s) && ok2(a) && g.resource.id === vb.resourceId && n, `Dialihkan & disetujui di kendaraan baru, notif pemohon=${n}`);
    const held = heldVehicle(d.driverId);
    check('SB-02', g.assignedVehicle?.id === vb.id && held === vb.id, `Kendaraan ditugaskan & dipegang supir = kendaraan baru`,
      `Resource = kendaraan BARU, tapi kendaraan ditugaskan = ${g.assignedVehicle?.id === va.id ? 'LAMA' : g.assignedVehicle?.id} & supir memegang ${held === va.id ? 'kendaraan LAMA' : held}`);
    check('SB-03', g.isReassigned && g.originalResource?.id === va.resourceId, `Penanda "Dialihkan dari" tampil`,
      `Penanda dialihkan tidak ada (isReassigned=${g.isReassigned}, originalResource=${JSON.stringify(g.originalResource)})`);
  });
  await scenario('SB-04..09', async () => {
    const d = await newDriver('DRVSB4'); const va = await newVehicle(); const room = await newRoom();
    const b = await book(U.EMPA.token, va.resourceId, wib(31, 9), wib(31, 12), { driverId: d.driverId });
    const r4 = await substitute(b.data.id, va.resourceId);
    check('SB-04', r4.status === 400, `Alihkan ke resource yang sama → ${r4.status}`);
    const r5 = await substitute(b.data.id, room.resourceId);
    check('SB-05', r5.status === 400, `Alihkan kendaraan → ruangan → ${r5.status} "${r5.msg}"`);
    const vm = await newVehicle({ status: 'MAINTENANCE' });
    const r6 = await substitute(b.data.id, vm.resourceId);
    check('SB-06', r6.status === 409, `Alihkan ke kendaraan MAINTENANCE → ${r6.status}`);
    const vc = await newVehicle(); const dc = await newDriver('DRVSB7');
    await book(U.EMPB.token, vc.resourceId, wib(31, 10), wib(31, 11), { driverId: dc.driverId });
    const r7 = await substitute(b.data.id, vc.resourceId);
    check('SB-07', r7.status === 409, `Alihkan ke kendaraan yang punya booking bentrok → ${r7.status}`);
    const ba = await bookApproved(U.EMPA.token, va.resourceId, wib(32, 9), wib(32, 10), { driverId: d.driverId });
    const r8 = await substitute(ba.id, (await newVehicle()).resourceId);
    check('SB-08', r8.status === 409, `Alihkan booking APPROVED → ${r8.status}`);
    const r1 = await newRoom(); const r2 = await newRoom();
    const br = await book(U.EMPA.token, r1.resourceId, wib(31, 9), wib(31, 10));
    const r9 = await substitute(br.data.id, r2.resourceId);
    check('SB-09', ok2(r9) && r9.data.resource.id === r2.resourceId, `Alihkan ruangan → ruangan → ${r9.status}`);
  });

  // ── AS ────────────────────────────────────────────────────────────────
  await scenario('AS-01', async () => {
    const d3 = await newDriver('DRVAS1'); const v = await newVehicle();
    const b = await onlyDrivers([], () => book(U.EMPA.token, v.resourceId, wib(33, 9), wib(33, 10)));
    await approve(b.data.id);
    const r = await assign(b.data.id, d3.driverId, v.id);
    const g = (await getBooking(b.data.id)).data;
    const held = heldVehicle(d3.driverId);
    const n = await hasNotif(d3.token, 'NEW_BOOKING', b.data.id);
    check('AS-01', ok2(r) && g.assignedDriver?.id === d3.driverId && held === v.id && n,
      `Supir ditugaskan, memegang kendaraan, dapat notifikasi`,
      `Supir ditugaskan=${g.assignedDriver?.id === d3.driverId}, notif=${n}, tapi TIDAK tercatat memegang kendaraan (held=${held})`);
  });
  await scenario('AS-02', async () => {
    const d = await newDriver('DRVAS2'); const va = await newVehicle(); const vb = await newVehicle();
    const b = await bookApproved(U.EMPA.token, va.resourceId, wib(34, 9), wib(34, 12), { driverId: d.driverId });
    const r = await assign(b.id, d.driverId, vb.id);
    const g = (await getBooking(b.id)).data;
    check('AS-02', ok2(r) && g.resource.id === vb.resourceId && g.isReassigned && g.originalResource?.id === va.resourceId,
      `Pindah kendaraan: resource baru + "Dialihkan dari" kendaraan lama`);
    const held = heldVehicle(d.driverId);
    record('AS-02b', held === vb.id ? 'PASS' : 'FAIL', `Supir kini memegang ${held === vb.id ? 'kendaraan BARU' : held === va.id ? 'kendaraan LAMA (tidak ikut pindah)' : held}`);
  });
  await scenario('AS-03/04', async () => {
    const d2 = await newDriver('DRVAS3A'); const d3 = await newDriver('DRVAS3B'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(120), { driverId: d2.driverId });
    const r = await assign(b.id, d3.driverId, v.id);
    const h2 = heldVehicle(d2.driverId); const h3 = heldVehicle(d3.driverId);
    check('AS-03', ok2(r) && h3 === v.id && h2 === null, `Pindah supir: supir baru memegang, supir lama kosong`,
      `Setelah pindah supir: supir LAMA memegang=${h2}, supir BARU memegang=${h3}`);
    await startB(b.id); await completeB(b.id);
    const h2b = heldVehicle(d2.driverId);
    check('AS-04', h2b === null, `Setelah selesai, supir lama kosong`, `Setelah booking selesai, supir LAMA MASIH memegang kendaraan ${h2b} (tertahan)`);
  });
  await scenario('AS-05..10', async () => {
    const d = await newDriver('DRVAS5'); const v = await newVehicle(); const room = await newRoom();
    const p = await book(U.EMPA.token, v.resourceId, wib(35, 9), wib(35, 12), { driverId: d.driverId });
    const r5 = await assign(p.data.id, d.driverId, v.id);
    check('AS-05', r5.status === 409, `Tugaskan pada booking PENDING → ${r5.status}`);
    const rb = await bookApproved(U.EMPA.token, room.resourceId, wib(35, 9), wib(35, 10));
    const r6 = await assign(rb.id, d.driverId, v.id);
    check('AS-06', r6.status === 400, `Tugaskan pada booking ruangan → ${r6.status}`);
    const ap = await bookApproved(U.EMPA.token, (await newVehicle()).resourceId, wib(36, 9), wib(36, 12), { driverId: (await newDriver('DRVAS6')).driverId });
    const dx = await newDriver('DRVAS7X');
    await api('PATCH', `/drivers/${dx.driverId}/toggle-active`, { token: U.ADM.token });
    const r7 = await assign(ap.id, dx.driverId, ap.assignedVehicle.id);
    check('AS-07', r7.status === 404, `Tugaskan supir nonaktif → ${r7.status} "${r7.msg}"`);
    const vc = await newVehicle();
    await bookApproved(U.EMPB.token, vc.resourceId, wib(36, 10), wib(36, 11), { driverId: (await newDriver('DRVAS8')).driverId });
    const r8 = await assign(ap.id, ap.assignedDriver.id, vc.id);
    check('AS-08', r8.status === 409, `Tugaskan kendaraan yang bentrok → ${r8.status}`);
    const vs = await newVehicle();
    await bookApproved(U.EMPB.token, vs.resourceId, wib(36, 15), wib(36, 16), { driverId: (await newDriver('DRVAS9')).driverId, bookingType: 'SPD' });
    const r9 = await assign(ap.id, ap.assignedDriver.id, vs.id);
    check('AS-09', r9.status === 409, `Tugaskan kendaraan yang SPD di hari itu → ${r9.status}`);
    const vm = await newVehicle();
    await maint(vm.id, wib(36, 7), wib(36, 18));
    const r10 = await assign(ap.id, ap.assignedDriver.id, vm.id);
    check('AS-10', r10.status === 409, `Tugaskan kendaraan yang dijadwalkan maintenance → ${r10.status}`);
  });
  await scenario('AS-11', async () => {
    const d = await newDriver('DRVAS11'); const va = await newVehicle(); const vb = await newVehicle();
    const b = await bookApproved(U.EMPA.token, va.resourceId, inMin(5), inMin(120), { driverId: d.driverId });
    await assign(b.id, d.driverId, vb.id);
    const s = await startB(b.id);
    check('AS-11', ok2(s) && resourceStatus(vb.resourceId) === 'IN_USE' && resourceStatus(va.resourceId) === 'AVAILABLE',
      `Mulai setelah pindah: baru=${resourceStatus(vb.resourceId)}, lama=${resourceStatus(va.resourceId)}`);
    U.ongoingForCN = b.id;
  });

  // ── CN ────────────────────────────────────────────────────────────────
  await scenario('CN-01/02', async () => {
    const d = await newDriver('DRVCN1'); const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(37, 9), wib(37, 12), { driverId: d.driverId });
    const c = await cancelB(b.data.id, U.EMPA.token);
    const g = (await getBooking(b.data.id)).data;
    const nA = await hasNotif(U.ADM.token, 'BOOKING_CANCELLED', b.data.id);
    const nD = await hasNotif(d.token, 'BOOKING_CANCELLED', b.data.id);
    check('CN-01', ok2(c) && g.status === 'CANCELLED' && nA && nD, `CANCELLED, notif admin=${nA}, notif supir=${nD}`);
    const b2 = await book(U.EMPB.token, v.resourceId, wib(37, 9), wib(37, 12), { driverId: d.driverId });
    const a2 = await approve(b2.data.id);
    check('CN-02', resourceStatus(v.resourceId) === 'AVAILABLE' && ok2(a2) && heldVehicle(d.driverId) === v.id,
      `Setelah batal, kendaraan & supir yang sama langsung dipakai & disetujui untuk booking lain`);
  });
  await scenario('CN-03/04', async () => {
    const d = await newDriver('DRVCN3'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, wib(38, 9), wib(38, 12), { driverId: d.driverId });
    const c1 = await cancelB(b.id, U.EMPA.token);
    const c2 = await cancelB(b.id, U.ADM.token);
    const g = (await getBooking(b.id)).data;
    const held = heldVehicle(d.driverId);
    const nE = await hasNotif(U.EMPA.token, 'BOOKING_CANCELLED', b.id);
    check('CN-03', c1.status === 403, `Karyawan membatalkan booking APPROVED → ${c1.status} "${c1.msg}" (hanya admin)`);
    check('CN-04', ok2(c2) && g.status === 'CANCELLED' && held === null && nE,
      `Admin membatalkan booking APPROVED → ${c2.status} ${g.status}; supir dilepas=${held === null}; notif pemohon=${nE}`);
  });
  await scenario('CN-05', async () => {
    const id = U.ongoingForCN;
    const c = await cancelB(id, U.ADM.token);
    check('CN-05', c.status >= 400, `Batalkan booking ONGOING → ${c.status}`);
  });
  await scenario('CN-06/07', async () => {
    const d = await newDriver('DRVCN6'); const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(39, 9), wib(39, 12), { driverId: d.driverId });
    const c6 = await cancelB(b.data.id, U.EMPB.token);
    check('CN-06', c6.status === 403, `Karyawan lain membatalkan → ${c6.status}`);
    const c7 = await cancelB(b.data.id, U.ADM.token);
    const act = (await activity(b.data.id)).find((a) => a.action === 'CANCEL');
    check('CN-07', ok2(c7) && act?.actor?.startsWith('Admin Uji'), `Admin membatalkan; timeline mencatat pelaku "${act?.actor}"`);
  });
}
