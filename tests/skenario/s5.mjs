// Batch 5: VD (vendor), KP (kepemilikan kendaraan), IS (laporan kendala supir), KS (kop surat / pengaturan dokumen)
import { api, ok2, sqlInt, check, scenario, inMin, fakePhoto, wib } from './lib.mjs';
import { U, newVehicle, newDriver, newRoom, book, bookApproved, startB, completeB, hasNotif, maintPlan, resourceStatus } from './fixtures.mjs';

const A = () => U.ADM.token;
const vehicleBody = (v, extra = {}) => ({ name: `Mobil ${v.plate}`, plateNumber: v.plate, brand: 'Toyota', model: 'Uji', year: 2024,
  currentOdometer: 10000, categoryId: 1, capacity: 6, ...extra });
const pngLogo = () => new Blob([Buffer.from(
  '89504e470d0a1a0a0000000d4948445200000001000000010806000000' +
  '1f15c4890000000d49444154789c6360000002000154a24f5d0000000049454e44ae426082', 'hex')], { type: 'image/png' });

export async function runBatch5() {
  // ── VD ────────────────────────────────────────────────────────────────
  await scenario('VD-01..04', async () => {
    const name = `Bengkel VD ${Date.now()}`;
    const c = await api('POST', '/vendors', { token: A(), body: { name, type: 'WORKSHOP', phone: '0811' } });
    const dup = await api('POST', '/vendors', { token: A(), body: { name: name.toUpperCase(), type: 'OWNER' } });
    const badType = await api('POST', '/vendors', { token: A(), body: { name: `${name} X`, type: 'SPBU' } });
    check('VD-01', c.status === 201 && (c.changed ?? '').includes('vendor') && dup.status === 409 && badType.status === 400,
      `Tambah vendor ${c.status} (X-Data-Changed=${c.changed}); nama kembar (beda huruf) ${dup.status}; jenis tak dikenal ${badType.status}`);
    const emp = await api('GET', '/vendors', { token: U.EMPA.token });
    check('VD-02', emp.status === 403, `Karyawan membuka master vendor → ${emp.status}`);
    const v = await newVehicle();
    await maintPlan(v.id, inMin(60 * 24 * 50), 1, { vendorId: c.data.id });
    const del = await api('DELETE', `/vendors/${c.data.id}`, { token: A() });
    const off = await api('PATCH', `/vendors/${c.data.id}/toggle-active`, { token: A() });
    const v2 = await newVehicle();
    const useOff = await maintPlan(v2.id, inMin(60 * 24 * 51), 1, { vendorId: c.data.id });
    check('VD-03', del.status === 409 && ok2(off) && off.data.isActive === false && useOff.status === 400,
      `Hapus vendor terpakai ${del.status}; nonaktifkan ${off.status}; dipakai saat nonaktif ${useOff.status} "${useOff.msg}"`);
    const unused = await api('POST', '/vendors', { token: A(), body: { name: `${name} Hapus`, type: 'BOTH' } });
    const delOk = await api('DELETE', `/vendors/${unused.data.id}`, { token: A() });
    check('VD-04', ok2(delOk), `Hapus vendor yang belum dipakai → ${delOk.status}`);
  });

  // ── KP ────────────────────────────────────────────────────────────────
  await scenario('KP-01..03', async () => {
    const v = await newVehicle();
    const workshopOnly = await api('POST', '/vendors', { token: A(), body: { name: `Bengkel KP ${Date.now()}`, type: 'WORKSHOP' } });
    const owner = await api('POST', '/vendors', { token: A(), body: { name: `Rental KP ${Date.now()}`, type: 'OWNER' } });
    const noVendor = await api('PUT', `/vehicles/${v.id}`, { token: A(), body: vehicleBody(v, { ownership: 'VENDOR' }) });
    const wrongType = await api('PUT', `/vehicles/${v.id}`, { token: A(), body: vehicleBody(v, { ownership: 'VENDOR', ownerVendorId: workshopOnly.data.id }) });
    check('KP-01', noVendor.status === 400 && wrongType.status === 400,
      `Sewa tanpa vendor ${noVendor.status}; vendor bengkel-saja sebagai pemilik ${wrongType.status}`);
    const ok = await api('PUT', `/vehicles/${v.id}`, { token: A(), body: vehicleBody(v, { ownership: 'VENDOR', ownerVendorId: owner.data.id, rentalContractNo: 'KTR-9' }) });
    const list = await api('GET', `/vehicles?search=${encodeURIComponent(v.plate)}`, { token: A() });
    const row = (list.data ?? []).find((x) => x.id === v.id);
    check('KP-02', ok2(ok) && ok.data.ownerVendor?.id === owner.data.id && ok.data.rentalContractNo === 'KTR-9' && row?.ownership === 'VENDOR',
      `Kendaraan sewa tersimpan (detail ${ok.data?.ownership}/${ok.data?.ownerVendor?.name}, daftar ${row?.ownership})`);
    const keep = await api('PUT', `/vehicles/${v.id}`, { token: A(), body: vehicleBody(v) });
    const back = await api('PUT', `/vehicles/${v.id}`, { token: A(), body: vehicleBody(v, { ownership: 'COMPANY' }) });
    check('KP-03', keep.data?.ownership === 'VENDOR' && back.data?.ownership === 'COMPANY' && back.data?.ownerVendor === null,
      `Edit tanpa field kepemilikan tidak mengubahnya (${keep.data?.ownership}); kembali milik perusahaan (${back.data?.ownership})`);
  });

  // ── IS ────────────────────────────────────────────────────────────────
  await scenario('IS-01..06', async () => {
    const d = await newDriver('DRVIS1'); const v = await newVehicle();
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(180), { driverId: d.driverId });
    await startB(b.id, d.token);
    const mine = await api('GET', '/vehicle-issues/my-vehicles', { token: d.token });
    const r = await api('POST', '/vehicle-issues', { token: d.token,
      form: { vehicleId: v.id, description: 'Ban kiri depan bocor di tol', location: 'KM 23 Tol Cikampek', canContinue: 'false', 'photos[]': fakePhoto() } });
    const nA = await hasNotif(A(), 'VEHICLE_ISSUE', null);
    check('IS-01', (mine.data ?? []).some((x) => x.vehicleId === v.id && x.onTrip && x.bookingId === b.id) && r.status === 201 && r.data.bookingId === b.id
      && r.data.canContinue === false && r.data.photos.length === 1 && (r.changed ?? '').includes('maintenance') && nA,
      `Supir melapor kendaraan tripnya → ${r.status}, tertaut booking #${r.data?.bookingId}, foto ${r.data?.photos?.length}, notif admin=${nA}`);
    const other = await newVehicle();
    const notMine = await api('POST', '/vehicle-issues', { token: d.token, form: { vehicleId: other.id, description: 'x', canContinue: 'true' } });
    const emp = await api('POST', '/vehicle-issues', { token: U.EMPA.token, form: { vehicleId: v.id, description: 'x', canContinue: 'true' } });
    check('IS-02', notMine.status === 403 && emp.status === 403,
      `Supir melapor kendaraan lain ${notMine.status}; karyawan melapor ${emp.status}`);
    const listDrv = await api('GET', '/vehicle-issues', { token: d.token });
    check('IS-03', (listDrv.data ?? []).length === 1 && listDrv.data[0].id === r.data.id, `Supir hanya melihat laporannya sendiri (${(listDrv.data ?? []).length})`);
    const conv = await api('POST', `/vehicle-issues/${r.data.id}/convert`, { token: A() });
    const mid = conv.data?.maintenanceId;
    const m = mid ? await api('GET', `/maintenance/${mid}`, { token: A() }) : {};
    const nD = await hasNotif(d.token, 'VEHICLE_ISSUE_UPDATE', null);
    check('IS-04', ok2(conv) && conv.data.status === 'CONVERTED' && m.data?.status === 'DRAFT' && m.data?.sourceIssueId === r.data.id && nD,
      `Ditindaklanjuti → ${conv.data?.status}, draf maintenance #${mid} (${m.data?.status}); notif supir=${nD}`);
    const twice = await api('POST', `/vehicle-issues/${r.data.id}/convert`, { token: A() });
    const r2 = await api('POST', '/vehicle-issues', { token: d.token, form: { vehicleId: v.id, description: 'AC mati', canContinue: 'true' } });
    const conv2 = await api('POST', `/vehicle-issues/${r2.data.id}/convert`, { token: A() });
    const openCount = sqlInt(`select count(*) from maintenance_records where "vehicleId"=${v.id} and status not in ('COMPLETED','CANCELLED')`);
    check('IS-05', twice.status === 409 && conv2.data?.maintenanceId === mid && openCount === 1,
      `Tindak lanjut ulang ${twice.status}; laporan kedua ditautkan ke maintenance yang sama (#${conv2.data?.maintenanceId}), proses terbuka ${openCount}`);
    const r3 = await api('POST', '/vehicle-issues', { token: d.token, form: { vehicleId: v.id, description: 'Lampu kabin redup', canContinue: 'true' } });
    const noNote = await api('POST', `/vehicle-issues/${r3.data.id}/dismiss`, { token: A(), body: {} });
    const dis = await api('POST', `/vehicle-issues/${r3.data.id}/dismiss`, { token: A(), body: { note: 'Sudah diganti sendiri' } });
    check('IS-06', noNote.status === 400 && ok2(dis) && dis.data.status === 'DISMISSED',
      `Abaikan tanpa catatan ${noNote.status}; dengan catatan → ${dis.data?.status}`);
  });

  // ── DS ────────────────────────────────────────────────────────────────
  await scenario('KS-01..02', async () => {
    const put = await api('PUT', '/document-settings', { token: A(), body: {
      companyName: 'PT Uji Skenario', companyAddress: 'Jl. Uji 99', companyPhone: '021-1', companyEmail: 'uji@kce-test.local',
      signerName: 'Budi Uji', signerTitle: 'Kepala Umum', letterCode: 'KCE-MNT' } });
    const bad = await api('PUT', '/document-settings', { token: A(), body: { companyName: 'X', letterCode: 'KCE/MNT' } });
    const emp = await api('GET', '/document-settings', { token: U.EMPA.token });
    check('KS-01', ok2(put) && put.data.signerName === 'Budi Uji' && bad.status === 400 && emp.status === 403,
      `Simpan kop & penandatangan ${put.status}; kode surat mengandung '/' ${bad.status}; karyawan ${emp.status}`);
    const logo = await api('POST', '/document-settings/logo', { token: A(), form: { logo: pngLogo() } });
    const badLogo = await api('POST', '/document-settings/logo', { token: A(), form: { logo: fakePhoto() } });
    const rm = await api('DELETE', '/document-settings/logo', { token: A() });
    check('KS-02', ok2(logo) && !!logo.data.logoUrl && ok2(rm) && rm.data.logoUrl === null,
      `Unggah logo ${logo.status} (${logo.data?.logoUrl}); hapus logo ${rm.status}; jpg-bernama-foto ${badLogo.status}`);
  });

  // ── Kiriman ganda (tombol ditekan beruntun / request terkirim ulang) ──
  await scenario('BC-22', async () => {
    const r = await newRoom();
    const same = () => book(U.EMPA.token, r.resourceId, wib(3, 9), wib(3, 10), { purpose: 'Rapat ganda' });
    // Serentak: dua request identik di saat yang sama → hanya satu booking.
    const [a, b] = await Promise.all([same(), same()]);
    // Berurutan: kiriman ulang setelah berhasil juga ditolak.
    const c = await same();
    const rows = sqlInt(`select count(*) from bookings where "resourceId"=${r.resourceId} and status='PENDING'`);
    // Jam berbeda / orang lain tetap boleh (BC-10).
    const other = await book(U.EMPA.token, r.resourceId, wib(3, 11), wib(3, 12));
    const otherUser = await book(U.EMPB.token, r.resourceId, wib(3, 9), wib(3, 10));
    const st = [a.status, b.status].sort();
    check('BC-22', st[0] === 201 && st[1] === 409 && c.status === 409 && rows === 1
      && other.status === 201 && otherUser.status === 201,
      `Serentak ${a.status}/${b.status}, ulang ${c.status} ("${c.msg}"), tersimpan ${rows}; ` +
      `jam lain ${other.status}, user lain ${otherUser.status}`);
  });
  await scenario('FL-12', async () => {
    const d = await newDriver('DRVFL12'); const v = await newVehicle({ odometer: 20000 });
    const fill = () => api('POST', '/fuel-expenses', { token: d.token,
      form: { vehicleId: v.id, fuelTypeId: 1, liter: 10, odometer: 20050, proofPhoto: fakePhoto() } });
    const [a, b] = await Promise.all([fill(), fill()]);
    const rows = sqlInt(`select count(*) from fuel_expenses where "vehicleId"=${v.id} and "voidedAt" is null`);
    const st = [a.status, b.status].sort();
    const rejected = a.status === 409 ? a : b;
    check('FL-12', st[0] === 201 && st[1] === 409 && rows === 1,
      `Dua pengisian identik serentak → ${a.status}/${b.status}, tersimpan ${rows} ("${rejected.msg}")`);
  });

  // ── Voucher ↔ catatan perjalanan, batal voucher terpakai, odometer akhir ──
  const station = await api('POST', '/fuel-stations', { token: A(), body: { name: `SPBU Uji ${Date.now()}` } });
  const voucherVehicle = async (code) => {
    const d = await newDriver(code); const v = await newVehicle({ odometer: 10000 });
    await api('PUT', `/fuel-balances/${v.id}/profile`, { token: A(), body: { kmPerLiter: 10, tankCapacityLiter: 45 } });
    return { d, v };
  };
  const issue = (v, d, odometer) => api('POST', '/fuel-vouchers', { token: A(),
    body: { vehicleId: v.id, fuelTypeId: 1, stationId: station.data.id, driverId: d.driverId, odometer } });
  const useV = (id, token, odometer) => api('PATCH', `/fuel-vouchers/${id}/use`, { token,
    form: { odometer, receiptPhoto: fakePhoto() } });
  const tripFuel = async (bookingId) =>
    (await api('GET', `/fuel-expenses?bookingId=${bookingId}`, { token: A() })).data ?? [];

  await scenario('VC-13', async () => {
    // Trip berjalan, voucher diterbitkan TANPA memilih booking.
    const { d, v } = await voucherVehicle('DRVVC13');
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(180), { driverId: d.driverId });
    await startB(b.id, d.token);
    const iv = await issue(v, d, 10200);
    const u = await useV(iv.data?.id, d.token, 10210);
    const fuel = await tripFuel(b.id);
    check('VC-13', ok2(iv) && iv.data.bookingId === b.id && ok2(u) && fuel.some((f) => f.source === 'VOUCHER'),
      `Terbit ${iv.status} (booking #${iv.data?.bookingId} = trip #${b.id}), diisi ${u.status}; ` +
      `catatan perjalanan memuat pengisian voucher: ${fuel.some((f) => f.source === 'VOUCHER')}`);
  });
  await scenario('VC-14', async () => {
    // Voucher terbit SEBELUM trip dimulai → ditautkan ke trip saat diisi.
    const { d, v } = await voucherVehicle('DRVVC14');
    const iv = await issue(v, d, 10200);
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(180), { driverId: d.driverId });
    await startB(b.id, d.token);
    const u = await useV(iv.data?.id, d.token, 10210);
    const fuel = await tripFuel(b.id);
    check('VC-14', ok2(iv) && iv.data.bookingId === null && ok2(u) && u.data.bookingId === b.id
      && fuel.some((f) => f.source === 'VOUCHER'),
      `Terbit tanpa trip (booking ${iv.data?.bookingId}), diisi saat trip #${b.id} → voucher booking #${u.data?.bookingId}, ` +
      `masuk catatan perjalanan: ${fuel.some((f) => f.source === 'VOUCHER')}`);

    // VC-07: voucher yang sudah diisi tidak bisa dibatalkan; yang belum diisi bisa.
    const cUsed = await api('PATCH', `/fuel-vouchers/${iv.data.id}/cancel`, { token: A(), body: { reason: 'uji' } });
    const other = await voucherVehicle('DRVVC07');
    const iv2 = await issue(other.v, other.d, 10200);
    const cIssued = await api('PATCH', `/fuel-vouchers/${iv2.data?.id}/cancel`, { token: A(), body: { reason: 'uji' } });
    check('VC-07', cUsed.status === 409 && ok2(cIssued) && cIssued.data.status === 'CANCELLED',
      `Batal voucher TERPAKAI → ${cUsed.status} ("${cUsed.msg}"); batal voucher belum diisi → ${cIssued.status}`);

    // RR-09: odometer akhir laporan pengembalian ≥ odometer kendaraan saat ini
    // (sudah maju ke 10210 lewat voucher di tengah trip) & memperbarui kendaraan.
    const rr = (odometer) => api('POST', `/bookings/${b.id}/return-report`, { token: d.token,
      form: { note: 'Kembali', location: '-6.2,106.8', odometer, 'photos[]': fakePhoto() } });
    const low = await rr(10205);
    const okR = await rr(10300);
    const odo = sqlInt(`select "currentOdometer" from vehicles where id=${v.id}`);
    check('RR-09', low.status === 400 && ok2(okR) && odo === 10300,
      `Odometer akhir < odometer kendaraan → ${low.status} ("${low.msg}"); 10300 → ${okR.status}, kendaraan ${odo} km`);
  });

  // ── Lokasi penjemputan/tujuan, status RETURNED, mulai lebih awal (000018) ──
  await scenario('BC-23', async () => {
    const v = await newVehicle(); const r = await newRoom();
    const bv = await book(U.EMPA.token, v.resourceId, wib(4, 9), wib(4, 12),
      { pickupLocation: 'Lobi Gedung A', destination: 'Bandara Soekarno-Hatta' });
    const br = await book(U.EMPA.token, r.resourceId, wib(4, 9), wib(4, 10),
      { pickupLocation: 'abaikan', destination: 'abaikan' });
    const dv = await api('GET', `/bookings/${bv.data?.id}`, { token: U.EMPA.token });
    check('BC-23', bv.status === 201 && dv.data?.pickupLocation === 'Lobi Gedung A'
      && dv.data?.destination === 'Bandara Soekarno-Hatta' && br.status === 201 && br.data?.pickupLocation === null,
      `Kendaraan: jemput "${dv.data?.pickupLocation}", tujuan "${dv.data?.destination}"; ruangan → lokasi ${br.data?.pickupLocation}`);
  });
  await scenario('RR-10', async () => {
    const d = await newDriver('DRVRR10'); const v = await newVehicle({ odometer: 10000 });
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(5), inMin(180), { driverId: d.driverId });
    await startB(b.id, d.token);
    const rr = await api('POST', `/bookings/${b.id}/return-report`, { token: d.token,
      form: { note: 'Kembali', location: '-6.2,106.8', odometer: 10100, 'photos[]': fakePhoto() } });
    const g = await api('GET', `/bookings/${b.id}`, { token: A() });
    const vehicleFree = resourceStatus(v.resourceId) === 'AVAILABLE';
    const driverFree = sqlInt(`select count(*) from driver_assignments where "driverId"=${d.driverId} and "releasedAt" is null`) === 0;
    const nAdmin = await hasNotif(A(), 'RETURN_REPORT', b.id);
    const nOwner = await hasNotif(U.EMPA.token, 'RETURN_REPORT', b.id);
    const c = await completeB(b.id);
    check('RR-10', ok2(rr) && g.data?.status === 'RETURNED' && !!g.data?.returnedAt && vehicleFree && driverFree
      && nAdmin && nOwner && ok2(c) && c.data?.status === 'COMPLETED',
      `Laporan → ${g.data?.status}; kendaraan bebas=${vehicleFree}, supir bebas=${driverFree}; ` +
      `notif admin=${nAdmin}, pemohon=${nOwner}; admin selesaikan → ${c.data?.status}`);
  });
  await scenario('ST-15', async () => {
    const setEarly = (key, value) => api('PUT', `/master-settings/${key}`, { token: A(), body: { value: String(value) } });
    const d = await newDriver('DRVST13'); const v = await newVehicle();
    // Non-SPD 15 menit: mulai 60 menit sebelum jadwal ditolak.
    await setEarly('booking_start_early_minutes_non_spd', 15);
    const b = await bookApproved(U.EMPA.token, v.resourceId, inMin(60), inMin(180), { driverId: d.driverId });
    const early = await startB(b.id, d.token);
    // Dilonggarkan jadi 90 menit → boleh.
    await setEarly('booking_start_early_minutes_non_spd', 90);
    const ok = await startB(b.id, d.token);
    const bad = await setEarly('booking_start_early_minutes_non_spd', -5);
    await setEarly('booking_start_early_minutes_non_spd', 15);
    const unit = (await api('GET', '/master-settings/booking_start_early_minutes_non_spd', { token: A() })).data?.unit;
    check('ST-15', early.status === 400 && ok2(ok) && bad.status === 400 && unit === 'menit',
      `Non-SPD 15 mnt, mulai 60 mnt lebih awal → ${early.status} ("${early.msg}"); diubah 90 mnt → ${ok.status}; ` +
      `nilai -5 → ${bad.status}; satuan tetap "${unit}"`);
  });
}
