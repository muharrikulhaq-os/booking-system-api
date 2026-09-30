// Batch 3: ST, CP, TO, MG, RR, RT
import { api, ok2, sql, sqlInt, record, check, scenario, wib, inMin, fakePhoto } from './lib.mjs';
import {
  U, newVehicle, newRoom, newDriver, book, getBooking, approve, startB, completeB, cancelB,
  hasNotif, onlyDrivers, bookApproved, heldVehicle, resourceStatus, setDates, sweep,
} from './fixtures.mjs';

const activity = async (id) => (await api('GET', `/bookings/${id}/activity`, { token: U.ADM.token })).data ?? [];
const merge = (primaryId, body) => api('POST', `/bookings/${primaryId}/merge`, { token: U.ADM.token, body });
const returnReport = (id, token, odometer) => api('POST', `/bookings/${id}/return-report`, {
  token, form: { note: 'Kendaraan kembali baik', location: '-6.2088,106.8456', odometer, 'photos[]': fakePhoto() } });
const rateDriver = (id, token, rating = 5) => api('POST', `/bookings/${id}/rate-driver`, { token, body: { rating, review: 'Supir ramah' } });
const rateRoom = (id, token, rating = 4) => api('POST', `/bookings/${id}/rate-room`, { token, body: { rating, review: 'Ruang bersih' } });
const maint = (vehicleId, start, end) => api('POST', '/maintenance', { token: U.ADM.token,
  body: { vehicleId, type: 'REPAIR', status: 'pending', description: 'Uji', location: 'Bengkel', startDate: start, endDate: end } });

/** Booking kendaraan ONGOING milik EMPA dengan supir d (sekarang dalam jendela mulai). */
async function ongoingVehicle(d, v, extra = {}, odometerStart) {
  const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(180), { driverId: d.driverId, ...extra });
  const s = await startB(b.id, d.token, { odometerStart, startLocation: '-6.2,106.8', startPhoto: fakePhoto() });
  if (!ok2(s)) throw new Error(`start gagal ${s.status} ${s.msg}`);
  return b.id;
}

export async function runBatch3() {
  // ── ST ────────────────────────────────────────────────────────────────
  const dST = await newDriver('DRVST1'); const vST = await newVehicle({ odometer: 10000 });
  let ongoingST;
  await scenario('ST-01', async () => {
    const b = await bookApproved(U.EMPA.token, vST.resourceId, inMin(10), inMin(180), { driverId: dST.driverId });
    const s = await startB(b.id, dST.token, { odometerStart: 10050, startLocation: '-6.2,106.8', startPhoto: fakePhoto() });
    const g = (await getBooking(b.id)).data;
    const n = await hasNotif(U.EMPA.token, 'BOOKING_STARTED', b.id);
    check('ST-01', ok2(s) && g.status === 'ONGOING' && resourceStatus(vST.resourceId) === 'IN_USE' && g.odometerStart === 10050 && n,
      `ONGOING, kendaraan IN_USE, odometer awal ${g.odometerStart}, notif pemohon=${n}`);
    ongoingST = b.id;
  });
  await scenario('ST-02', async () => {
    const d = await newDriver('DRVST2'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(60), inMin(120), { driverId: d.driverId });
    const s = await startB(b.id, d.token);
    check('ST-02', s.status === 400, `Mulai 60 menit sebelum jadwal → ${s.status} "${s.msg}"`);
  });
  await scenario('ST-03', async () => {
    const d = await newDriver('DRVST3'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(10), inMin(60), { driverId: d.driverId });
    setDates(b.id, inMin(-180), inMin(-60));
    const s = await startB(b.id, d.token);
    check('ST-03', s.status === 400, `Mulai setelah jam selesai → ${s.status} "${s.msg}"`);
  });
  await scenario('ST-04/05', async () => {
    const d = await newDriver('DRVST4'); const dOther = await newDriver('DRVST5'); const v = await newVehicle({ odometer: 10000 });
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(120), { driverId: d.driverId });
    const s4 = await startB(b.id, d.token, { odometerStart: 9000 });
    check('ST-04', s4.status === 400, `Odometer awal < tercatat → ${s4.status} "${s4.msg}"`);
    const s5 = await startB(b.id, dOther.token);
    check('ST-05', s5.status === 403, `Supir lain memulai → ${s5.status}`);
  });
  const rKept = await newRoom({ keeperId: U.RK1.rkId });
  let roomOngoing;
  await scenario('ST-06/08', async () => {
    const b = await bookApproved(U.EMPA.token, rKept.resourceId, inMin(5), inMin(90));
    const s6 = await startB(b.id, dST.token);
    check('ST-06', s6.status === 400 || s6.status === 403, `Supir memulai booking ruangan → ${s6.status}`);
    const s8 = await startB(b.id, U.EMPA.token);
    check('ST-08', ok2(s8) && resourceStatus(rKept.resourceId) === 'IN_USE', `Karyawan memulai ruangannya sendiri → ${s8.status}, ruangan ${resourceStatus(rKept.resourceId)}`);
    roomOngoing = b.id;
  });
  let noDriverCompleted;
  await scenario('ST-07', async () => {
    const v = await newVehicle();
    const b = await onlyDrivers([], () => book(U.EMPA.token, v.resourceId, inMin(5), inMin(90)));
    await approve(b.data.id);
    const s = await startB(b.data.id);
    record('ST-07', 'INFO', `Admin memulai booking kendaraan TANPA supir → ${s.status} (UI menyembunyikan tombol; perlu keputusan apakah API juga menolak)`);
    if (ok2(s)) { await completeB(b.data.id); noDriverCompleted = b.data.id; }
  });
  await scenario('ST-09', async () => {
    const d = await newDriver('DRVST9'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(90), { driverId: d.driverId });
    const s = await startB(b.id, U.EMPA.token);
    check('ST-09', s.status === 400, `Karyawan memulai booking kendaraannya → ${s.status}`);
  });
  await scenario('ST-10/11', async () => {
    const r1 = await newRoom({ keeperId: U.RK1.rkId });
    const b1 = await bookApproved(U.EMPA.token, r1.resourceId, inMin(5), inMin(60));
    const s10 = await startB(b1.id, U.RK1.token);
    check('ST-10', ok2(s10), `Penjaga ruangan memulai ruangannya → ${s10.status}`);
    const r2 = await newRoom({ keeperId: U.RK1.rkId });
    const b2 = await bookApproved(U.EMPA.token, r2.resourceId, inMin(5), inMin(60));
    const s11 = await startB(b2.id, U.RK2.token);
    record('ST-11', 'INFO', `Penjaga ruangan LAIN (bukan penjaga ruangan itu) memulai → ${s11.status}`);
  });
  await scenario('ST-12', async () => {
    const d = await newDriver('DRVST12'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(120), { driverId: d.driverId });
    await maint(v.id, inMin(-10), inMin(600));
    const s = await startB(b.id, d.token);
    check('ST-12', s.status === 409, `Maintenance dibuat setelah disetujui → mulai ${s.status} "${s.msg}"`);
  });

  // ── CP ────────────────────────────────────────────────────────────────
  let completedVehicle;
  await scenario('CP-01', async () => {
    const c = await completeB(ongoingST);
    const g = (await getBooking(ongoingST)).data;
    const nDone = await hasNotif(U.EMPA.token, 'BOOKING_COMPLETED', ongoingST);
    const nRate = await hasNotif(U.EMPA.token, 'RATE_DRIVER_PROMPT', ongoingST);
    check('CP-01', ok2(c) && g.status === 'COMPLETED' && resourceStatus(vST.resourceId) === 'AVAILABLE' && heldVehicle(dST.driverId) === null && !g.overtime && nDone && nRate,
      `COMPLETED, kendaraan ${resourceStatus(vST.resourceId)}, supir dilepas=${heldVehicle(dST.driverId) === null}, notif selesai=${nDone}, ajakan rating=${nRate}, overtime=${JSON.stringify(g.overtime)}`);
    completedVehicle = ongoingST;
  });
  await scenario('CP-02/03', async () => {
    for (const [id, type] of [['CP-02', 'NON_SPD'], ['CP-03', 'SPD']]) {
      const d = await newDriver(`DRV${id.replace('-', '')}`); const v = await newVehicle();
      const bid = await ongoingVehicle(d, v, { bookingType: type });
      setDates(bid, inMin(-240), inMin(-90));
      await completeB(bid);
      const ot = (await getBooking(bid)).data.overtime;
      if (type === 'NON_SPD') {
        const nD = await hasNotif(d.token, 'OVERTIME_RECORDED', bid);
        check(id, ot && ot.overtimeMinutes >= 89 && ot.overtimeMinutes <= 91 && nD, `Selesai 90 menit terlambat → overtime ${ot?.overtimeMinutes} menit, notif supir=${nD}`);
      } else {
        check(id, !ot, `SPD terlambat → tanpa overtime (${JSON.stringify(ot)})`);
      }
    }
  });
  await scenario('CP-04/TO-04', async () => {
    const d = await newDriver('DRVCP4'); const v = await newVehicle();
    const bid = await ongoingVehicle(d, v);
    setDates(bid, inMin(-240), inMin(-30));
    await sweep();
    const g = (await getBooking(bid)).data;
    check('TO-04', g.status === 'OVERDUE' && resourceStatus(v.resourceId) === 'IN_USE', `Lewat jam selesai → ${g.status}, kendaraan ${resourceStatus(v.resourceId)}`);
    const c = await completeB(bid);
    const g2 = (await getBooking(bid)).data;
    check('CP-04', ok2(c) && g2.status === 'COMPLETED' && !!g2.overtime, `Selesaikan OVERDUE → ${g2.status}, overtime ${g2.overtime?.overtimeMinutes} menit`);
  });
  await scenario('CP-05/06', async () => {
    const d = await newDriver('DRVCP5'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, wib(41, 9), wib(41, 10), { driverId: d.driverId });
    const c5 = await completeB(b.id);
    check('CP-05', c5.status === 409, `Selesaikan booking APPROVED → ${c5.status}`);
    const bid = await ongoingVehicle(d, await newVehicle());
    const c6 = await completeB(bid, d.token);
    check('CP-06', c6.status === 403, `Supir menyelesaikan → ${c6.status}`);
  });
  await scenario('CP-07', async () => {
    const c = await completeB(roomOngoing, U.EMPA.token);
    check('CP-07', ok2(c) && resourceStatus(rKept.resourceId) === 'AVAILABLE', `Karyawan menyelesaikan ruangannya → ${c.status}, ruangan ${resourceStatus(rKept.resourceId)}`);
  });
  await scenario('CP-08', async () => {
    const d = await newDriver('DRVCP8'); const v = await newVehicle({ fixedDriverId: d.driverId });
    const b1 = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(60));
    await bookApproved(U.EMPB.token, v.resourceId, wib(42, 9), wib(42, 10));
    await startB(b1.id, d.token); await completeB(b1.id);
    check('CP-08', heldVehicle(d.driverId) === v.id, `Supir masih punya booking aktif lain → tetap memegang kendaraan`);
  });

  // ── RR (+ CP-10) ──────────────────────────────────────────────────────
  await scenario('RR-01/03/08', async () => {
    const d = await newDriver('DRVRR1'); const v = await newVehicle({ odometer: 10000 });
    const bid = await ongoingVehicle(d, v, {}, 10020);
    const r = await returnReport(bid, d.token, 10100);
    const odo = sqlInt(`select "currentOdometer" from vehicles where id=${v.id}`);
    const n = await hasNotif(U.ADM.token, 'RETURN_REPORT', bid);
    check('RR-01', ok2(r) && odo === 10100 && n, `Laporan terkirim, odometer kendaraan ${odo}, notif admin=${n}`);
    const r3 = await returnReport(bid, d.token, 10110);
    check('RR-03', r3.status === 409, `Laporan kedua → ${r3.status}`);
    const g = await api('GET', `/bookings/${bid}/return-report`, { token: U.EMPA.token });
    check('RR-08', ok2(g) && (g.data?.photos?.length ?? 0) >= 1, `Pemohon melihat laporan: ${g.status}, foto ${g.data?.photos?.length}`);
  });
  await scenario('RR-02/04/06/07', async () => {
    const d = await newDriver('DRVRR2'); const v = await newVehicle({ odometer: 10000 });
    const bid = await ongoingVehicle(d, v, {}, 10050);
    const r4 = await returnReport(bid, d.token, 10000);
    check('RR-04', r4.status === 400, `Odometer akhir < awal trip → ${r4.status} "${r4.msg}"`);
    const r6 = await returnReport(bid, U.ADM.token, 10100);
    check('RR-06', r6.status === 403, `Admin mengirim laporan → ${r6.status}`);
    setDates(bid, inMin(-240), inMin(-30)); await sweep();
    const r2 = await returnReport(bid, d.token, 10100);
    check('RR-02', ok2(r2), `Laporan saat OVERDUE → ${r2.status}`);
    const r7 = await returnReport(roomOngoing, d.token, 1);
    check('RR-07', r7.status === 400 || r7.status === 403, `Laporan untuk booking ruangan → ${r7.status}`);
  });
  await scenario('RR-05/CP-10', async () => {
    const d = await newDriver('DRVRR5'); const v = await newVehicle({ odometer: 10000 });
    const bid = await ongoingVehicle(d, v, {}, 10000);
    await returnReport(bid, d.token, 20500);
    const autoM = sqlInt(`select count(*) from maintenance_records where "vehicleId"=${v.id}`);
    check('RR-05', autoM === 0 && resourceStatus(v.resourceId) === 'IN_USE',
      `Odometer akhir +10.500 km → tidak ada maintenance otomatis (${autoM}), kendaraan tetap ${resourceStatus(v.resourceId)}`);
    // Maintenance diajukan admin saat trip masih berjalan (mis. mobil dilaporkan rusak).
    await maint(v.id, inMin(-5), inMin(600));
    await completeB(bid);
    check('CP-10', resourceStatus(v.resourceId) === 'MAINTENANCE', `Setelah booking selesai, kendaraan tetap MAINTENANCE`,
      `Maintenance terbuka, tapi setelah booking selesai status kendaraan jadi ${resourceStatus(v.resourceId)}`);
  });

  // ── TO ────────────────────────────────────────────────────────────────
  await scenario('TO-01/05', async () => {
    const d = await newDriver('DRVTO1'); const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(43, 9), wib(43, 10), { driverId: d.driverId });
    setDates(b.data.id, inMin(-300), inMin(-240));
    const before = (await getBooking(b.data.id)).data.status;
    const dash = await api('GET', '/dashboard/summary', { token: U.ADM.token });
    check('TO-05', before === 'IGNORED', `Tanpa membuka daftar, status sudah IGNORED`,
      `Jam selesai sudah lewat, tapi detail booking masih ${before} sampai ada yang membuka DAFTAR booking (dashboard ${dash.status})`);
    await sweep();
    const g = (await getBooking(b.data.id)).data;
    const act = (await activity(b.data.id)).find((a) => a.action === 'IGNORED');
    check('TO-01', g.status === 'IGNORED' && act && !act.actor, `Setelah daftar dibuka → ${g.status}, timeline aksi sistem=${!!act}`);
  });
  await scenario('TO-02/03/06', async () => {
    const d = await newDriver('DRVTO2'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, wib(44, 9), wib(44, 10), { driverId: d.driverId });
    const heldBefore = heldVehicle(d.driverId);
    setDates(b.id, inMin(-300), inMin(-240)); await sweep();
    const g = (await getBooking(b.id)).data;
    check('TO-02', g.status === 'EXPIRED', `APPROVED tak pernah dimulai → ${g.status}`);
    const held = heldVehicle(d.driverId);
    check('TO-03', held === null, `Supir dilepas setelah EXPIRED`, `Setelah EXPIRED supir MASIH memegang kendaraan (sebelum=${heldBefore}, sesudah=${held})`);
    const v2 = await newVehicle(); const other = await newDriver('DRVTO3');
    const b2 = await book(U.EMPB.token, v.resourceId, wib(45, 9), wib(45, 10), { driverId: other.driverId });
    const a2 = await approve(b2.data.id);
    record('TO-03b', ok2(a2) ? 'PASS' : 'FAIL', `Kendaraan bekas booking EXPIRED dipakai supir lain → approve ${a2.status} "${a2.msg}"`);
    void v2;
    const a = await approve(b.id); const s = await startB(b.id); const c = await cancelB(b.id, U.EMPA.token);
    check('TO-06', a.status >= 400 && s.status >= 400 && c.status >= 400, `Booking EXPIRED: approve ${a.status}, mulai ${s.status}, batal ${c.status}`);
  });

  // ── MG ────────────────────────────────────────────────────────────────
  await scenario('MG-01', async () => {
    const dA = await newDriver('DRVMG1A'); const dB = await newDriver('DRVMG1B'); const va = await newVehicle(); const vb = await newVehicle();
    const P = await bookApproved(U.EMPA.token, va.resourceId, wib(46, 9), wib(46, 12), { driverId: dA.driverId, passengerCount: 3 });
    const T = await book(U.EMPB.token, vb.resourceId, wib(46, 10), wib(46, 13), { driverId: dB.driverId, passengerCount: 2 });
    const m = await merge(P.id, { targetBookingId: T.data.id, reason: 'Searah' });
    const a = await approve(T.data.id, U.ADM.token, 'telah dilakukan merge');
    const gT = (await getBooking(T.data.id)).data; const gP = (await getBooking(P.id)).data;
    const vbFree = sqlInt(`select count(*) from bookings where "resourceId"=${vb.resourceId} and status in ('PENDING','APPROVED','ONGOING')`) === 0;
    const t = (x) => Date.parse(x);
    const sameWindow = t(gT.startDate) === t(gP.startDate) && t(gT.endDate) === t(gP.endDate) && t(gP.startDate) === t(wib(46, 9)) && t(gP.endDate) === t(wib(46, 13));
    const n = await hasNotif(U.EMPB.token, 'BOOKING_MERGED', T.data.id);
    check('MG-01', ok2(m) && ok2(a) && gT.resource.id === va.resourceId && gT.assignedDriver?.id === dA.driverId && gT.status === 'APPROVED' && sameWindow && vbFree && n,
      `Gabung: ikut kendaraan & supir utama, jendela 09–13 (${sameWindow}), kendaraan lama bebas=${vbFree}, notif=${n}`);
    U.mg = { P: P.id, T: T.data.id };
  });
  await scenario('MG-02..08', async () => {
    const dA = await newDriver('DRVMG2A'); const va = await newVehicle({ capacity: 4 });
    const P = await bookApproved(U.EMPA.token, va.resourceId, wib(47, 9), wib(47, 11), { driverId: dA.driverId, passengerCount: 3 });
    const T = await book(U.EMPB.token, (await newVehicle()).resourceId, wib(47, 10), wib(47, 12), { driverId: (await newDriver('DRVMG2B')).driverId, passengerCount: 3 });
    const m2 = await merge(P.id, { targetBookingId: T.data.id, startDate: wib(47, 8), endDate: wib(47, 14) });
    const gP = (await getBooking(P.id)).data;
    check('MG-02', ok2(m2) && Date.parse(gP.startDate) === Date.parse(wib(47, 8)) && Date.parse(gP.endDate) === Date.parse(wib(47, 14)), `Jendela gabungan kustom 08–14 diterapkan`);
    const a8 = await approve(T.data.id);
    check('MG-08', ok2(a8) && !!a8.json?.warning, `Total penumpang 6 di kapasitas 4 → peringatan "${a8.json?.warning ?? '-'}"`);
    const m3 = await merge(P.id, { targetBookingId: P.id });
    check('MG-03', m3.status === 400, `Gabung dengan diri sendiri → ${m3.status}`);
    const m4 = await merge(P.id, { targetBookingId: T.data.id });
    check('MG-04', m4.status === 409, `Gabung ulang pasangan yang sama → ${m4.status}`);
    const r1 = await newRoom(); const r2 = await newRoom();
    const RP = await bookApproved(U.EMPA.token, r1.resourceId, wib(47, 9), wib(47, 10));
    const RT = await book(U.EMPB.token, r2.resourceId, wib(47, 9), wib(47, 10));
    const m5 = await merge(RP.id, { targetBookingId: RT.data.id });
    check('MG-05', m5.status === 400, `Gabung booking ruangan → ${m5.status}`);
    const dO = await newDriver('DRVMG6'); const vo = await newVehicle();
    const O = await ongoingVehicle(dO, vo);
    const T6 = await book(U.EMPB.token, (await newVehicle()).resourceId, inMin(10), inMin(60), { driverId: (await newDriver('DRVMG6B')).driverId });
    const m6 = await merge(O, { targetBookingId: T6.data.id });
    check('MG-06', m6.status === 409, `Gabung ke booking ONGOING → ${m6.status}`);
    const dC = await newDriver('DRVMG7'); const vc = await newVehicle();
    const PC = await bookApproved(U.EMPA.token, vc.resourceId, wib(48, 9), wib(48, 12), { driverId: dC.driverId });
    await bookApproved(U.EMPB.token, vc.resourceId, wib(48, 14), wib(48, 16), { driverId: dC.driverId });
    const TC = await book(U.EMPB.token, (await newVehicle()).resourceId, wib(48, 13), wib(48, 15), { driverId: (await newDriver('DRVMG7B')).driverId });
    const m7 = await merge(PC.id, { targetBookingId: TC.data.id });
    check('MG-07', m7.status === 409, `Jendela gabungan menabrak booking lain di kendaraan yang sama → ${m7.status}`);
  });
  await scenario('MG-09', async () => {
    const dA = await newDriver('DRVMG9A'); const dNew = await newDriver('DRVMG9N'); const va = await newVehicle();
    const P = await bookApproved(U.EMPA.token, va.resourceId, wib(49, 9), wib(49, 11), { driverId: dA.driverId });
    const T = await book(U.EMPB.token, (await newVehicle()).resourceId, wib(49, 9), wib(49, 11), { driverId: (await newDriver('DRVMG9B')).driverId });
    const m = await merge(P.id, { targetBookingId: T.data.id, driverId: dNew.driverId });
    const gP = (await getBooking(P.id)).data; const gT = (await getBooking(T.data.id)).data;
    check('MG-09', ok2(m) && gP.assignedDriver?.id === dNew.driverId && gP.assignedVehicle?.id === va.id && gT.assignedDriver?.id === dNew.driverId,
      `Gabung + ganti supir: supir baru di kedua booking, kendaraan tetap`,
      `Gabung + ganti supir: supir utama=${gP.assignedDriver?.name}, kendaraan utama=${gP.assignedVehicle?.id ?? 'HILANG (null)'}, supir sekunder=${gT.assignedDriver?.name}`);
  });
  await scenario('MG-10/11', async () => {
    const dA = await newDriver('DRVMG10'); const va = await newVehicle();
    const P = await bookApproved(U.EMPA.token, va.resourceId, inMin(5), inMin(120), { driverId: dA.driverId });
    const T = await book(U.EMPB.token, (await newVehicle()).resourceId, inMin(10), inMin(100), { driverId: (await newDriver('DRVMG10B')).driverId });
    await merge(P.id, { targetBookingId: T.data.id }); await approve(T.data.id);
    await startB(P.id, dA.token);
    const sT = (await getBooking(T.data.id)).data.status;
    await completeB(P.id);
    const cT = (await getBooking(T.data.id)).data.status;
    check('MG-10', sT === 'ONGOING' && cT === 'COMPLETED' && heldVehicle(dA.driverId) === null,
      `Mulai utama → sekunder ${sT}; selesai utama → sekunder ${cT}; supir dilepas=${heldVehicle(dA.driverId) === null}`);
    const r = await rateDriver(T.data.id, U.EMPB.token);
    check('MG-11', r.status === 400, `Rating dari booking sekunder → ${r.status} "${r.msg}"`);
  });

  // ── RT ────────────────────────────────────────────────────────────────
  await scenario('RT-01/02/03', async () => {
    const r = await rateDriver(completedVehicle, U.EMPA.token, 5);
    const n = await hasNotif(dST.token, 'DRIVER_RATED', completedVehicle);
    const sum = await api('GET', `/bookings/drivers/${dST.driverId}/ratings`, { token: U.ADM.token });
    check('RT-01', ok2(r) && n && Number(sum.data?.averageRating ?? sum.data?.average) === 5, `Rating tersimpan, notif supir=${n}, rata-rata=${sum.data?.averageRating ?? sum.data?.average}`);
    const r2 = await rateDriver(completedVehicle, U.EMPA.token, 4);
    check('RT-02', r2.status === 409, `Rating kedua → ${r2.status}`);
    const r3a = await rateDriver(completedVehicle, U.EMPB.token);
    const pend = await bookApproved(U.EMPA.token, vST.resourceId, wib(50, 9), wib(50, 10), { driverId: dST.driverId });
    const r3b = await rateDriver(pend.id, U.EMPA.token);
    check('RT-03', r3a.status === 403 && r3b.status === 400, `Bukan pemilik → ${r3a.status}; belum selesai → ${r3b.status}`);
  });
  await scenario('RT-04', async () => {
    if (!noDriverCompleted) { record('RT-04', 'SKIP', 'Butuh booking tanpa supir yang selesai (ST-07 ditolak)'); return; }
    const r = await rateDriver(noDriverCompleted, U.EMPA.token);
    check('RT-04', r.status === 400, `Rating supir pada booking tanpa supir → ${r.status}`);
  });
  await scenario('RT-05/06', async () => {
    const rNo = await newRoom();
    const b = await bookApproved(U.EMPA.token, rNo.resourceId, inMin(5), inMin(60));
    await startB(b.id, U.EMPA.token); await completeB(b.id, U.EMPA.token);
    const r5 = await rateRoom(b.id, U.EMPA.token);
    check('RT-05', ok2(r5) && r5.data.roomKeeperId == null, `Rating ruangan tanpa penjaga → ${r5.status}`);
    const r6 = await rateRoom(roomOngoing, U.EMPA.token, 5);
    const list = await api('GET', `/bookings/room-keepers/${U.RK1.rkId}/ratings`, { token: U.ADM.token });
    const found = (list.data?.ratings ?? []).some((x) => x.ratedBy?.id === U.EMPA.id && x.rating === 5);
    check('RT-06', ok2(r6) && found, `Rating ruangan masuk ringkasan penjaga (${found})`);
  });
}
