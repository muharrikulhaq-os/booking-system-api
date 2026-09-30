// Batch 5: VD (vendor), KP (kepemilikan kendaraan), IS (laporan kendala supir), DS (pengaturan dokumen)
import { api, ok2, sqlInt, check, scenario, inMin, fakePhoto } from './lib.mjs';
import { U, newVehicle, newDriver, bookApproved, startB, hasNotif, maintPlan } from './fixtures.mjs';

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
    check('IS-01', (mine.data ?? []).some((x) => x.vehicleId === v.id && x.onTrip) && r.status === 201 && r.data.bookingId === b.id
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
  await scenario('DS-01..02', async () => {
    const put = await api('PUT', '/document-settings', { token: A(), body: {
      companyName: 'PT Uji Skenario', companyAddress: 'Jl. Uji 99', companyPhone: '021-1', companyEmail: 'uji@kce-test.local',
      signerName: 'Budi Uji', signerTitle: 'Kepala Umum', letterCode: 'KCE-MNT' } });
    const bad = await api('PUT', '/document-settings', { token: A(), body: { companyName: 'X', letterCode: 'KCE/MNT' } });
    const emp = await api('GET', '/document-settings', { token: U.EMPA.token });
    check('DS-01', ok2(put) && put.data.signerName === 'Budi Uji' && bad.status === 400 && emp.status === 403,
      `Simpan kop & penandatangan ${put.status}; kode surat mengandung '/' ${bad.status}; karyawan ${emp.status}`);
    const logo = await api('POST', '/document-settings/logo', { token: A(), form: { logo: pngLogo() } });
    const badLogo = await api('POST', '/document-settings/logo', { token: A(), form: { logo: fakePhoto() } });
    const rm = await api('DELETE', '/document-settings/logo', { token: A() });
    check('DS-02', ok2(logo) && !!logo.data.logoUrl && ok2(rm) && rm.data.logoUrl === null,
      `Unggah logo ${logo.status} (${logo.data?.logoUrl}); hapus logo ${rm.status}; jpg-bernama-foto ${badLogo.status}`);
  });
}
