// Batch 1: AU, RL, BC, SP
import { api, ok2, sql, sqlInt, record, check, scenario, wib, inMin, PASSWORD } from './lib.mjs';
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
import {
  U, ROLE, login, createUser, newVehicle, newRoom, newDriver, setVehicleStatus, book, getBooking,
  approve, cancelB, hasNotif, onlyDrivers, bookApproved, heldVehicle, maintPlan,
} from './fixtures.mjs';

export async function runBatch1() {
  const A = () => U.ADM.token;

  // ── AU ────────────────────────────────────────────────────────────────
  await scenario('AU-01', async () => {
    const roles = [];
    for (const k of ['ADM', 'EMPA', 'RK1']) {
      const me = await api('GET', '/auth/me', { token: U[k].token });
      roles.push(`${k}=${me.data?.role?.name}`);
    }
    const d = await newDriver('DRVLOGIN');
    const me = await api('GET', '/auth/me', { token: d.token });
    roles.push(`DRV=${me.data?.role?.name}`);
    check('AU-01', roles.join() === 'ADM=ADMIN,EMPA=EMPLOYEE,RK1=ROOM_KEEPER,DRV=DRIVER', `Login semua role OK (${roles.join(', ')})`);
  });
  await scenario('AU-02', async () => {
    const r = await api('POST', '/auth/login', { body: { email: U.EMPA.email, password: 'SalahSekali123' } });
    check('AU-02', r.status === 401, `Password salah → ${r.status} "${r.msg}"`);
  });
  await scenario('AU-03', async () => {
    const r = await api('POST', '/auth/login', { body: { email: 'tidakada@kce-test.local', password: 'Apapun12345' } });
    check('AU-03', r.status >= 400 && r.status < 500, `Email tidak terdaftar → ${r.status} "${r.msg}"`);
  });
  await scenario('AU-04/05', async () => {
    const u = await createUser('EMPOFF', ROLE.EMPLOYEE);
    await sleep(1100);
    const l = await api('POST', '/auth/login', { body: { email: u.email, password: PASSWORD } });
    const refresh = l.data.refreshToken;
    await api('PATCH', `/users/${u.id}/toggle-active`, { token: A() });
    const l2 = await api('POST', '/auth/login', { body: { email: u.email, password: PASSWORD } });
    check('AU-04', l2.status >= 400, `Login akun nonaktif → ${l2.status} "${l2.msg}"`);
    const rf = await api('POST', '/auth/refresh', { body: { refreshToken: refresh } });
    check('AU-05', rf.status >= 400, `Refresh token akun yang baru dinonaktifkan → ${rf.status} "${rf.msg}"`);
    const me = await api('GET', '/auth/me', { token: l.data.accessToken });
    check('AU-05b', me.status === 401, `Access token lama akun nonaktif langsung ditolak → ${me.status}`);
  });
  await scenario('AU-08', async () => {
    const u = await createUser('EMPLOGOUT', ROLE.EMPLOYEE);
    await sleep(1100);
    const l = await api('POST', '/auth/login', { body: { email: u.email, password: PASSWORD } });
    await api('POST', '/auth/logout', { token: l.data.accessToken, body: { refreshToken: l.data.refreshToken } });
    const rf = await api('POST', '/auth/refresh', { body: { refreshToken: l.data.refreshToken } });
    check('AU-08', rf.status >= 400, `Refresh setelah logout → ${rf.status}`);
  });
  await scenario('AU-12', async () => {
    const r = await api('PATCH', '/auth/change-password', { token: U.EMPB.token, body: { currentPassword: 'BukanPassword1', newPassword: 'PasswordBaru123' } });
    check('AU-12', r.status >= 400, `Ganti password dengan password lama salah → ${r.status} "${r.msg}"`);
  });
  await scenario('AU-18', async () => {
    const u = await createUser('EMPDOUBLE', ROLE.EMPLOYEE);
    await sleep(1100);
    const [a, b] = await Promise.all([1, 2].map(() => api('POST', '/auth/login', { body: { email: u.email, password: PASSWORD } })));
    check('AU-18', ok2(a) && ok2(b), `Dua login bersamaan (klik ganda / dua perangkat) → ${a.status}, ${b.status}`,
      `Dua login di detik yang sama → ${a.status}, ${b.status}: refresh token kembar melanggar unique constraint`);
  });
  record('AU-10', 'SKIP', 'OTP lewat email — SMTP sengaja dimatikan di lingkungan test');
  record('AU-11', 'SKIP', 'OTP lewat email — SMTP sengaja dimatikan di lingkungan test');

  // ── RL ────────────────────────────────────────────────────────────────
  const d1 = await newDriver('DRVRL1'); const d2 = await newDriver('DRVRL2');
  const vRL = await newVehicle({ fixedDriverId: d1.driverId });
  const bRL = await book(U.EMPA.token, vRL.resourceId, wib(1, 8), wib(1, 9));
  await scenario('RL-01', async () => {
    const r1 = await book(d2.token, vRL.resourceId, wib(1, 10), wib(1, 11));
    const r2 = await book(U.RK1.token, vRL.resourceId, wib(1, 10), wib(1, 11));
    check('RL-01', r1.status === 403 && r2.status === 403, `DRIVER buat booking → ${r1.status}, ROOM_KEEPER → ${r2.status}`);
  });
  await scenario('RL-02', async () => {
    const r = await getBooking(bRL.data.id, U.EMPB.token);
    check('RL-02', r.status === 403 || r.status === 404, `EMPB buka booking EMPA → ${r.status}`);
  });
  await scenario('RL-03', async () => {
    const r = await getBooking(bRL.data.id, d2.token);
    check('RL-03', r.status === 403, `Supir lain buka booking yang bukan tugasnya → ${r.status}`);
  });
  await scenario('RL-04', async () => {
    const r = await api('GET', '/bookings?limit=100', { token: U.EMPA.token });
    const bad = (r.data ?? []).filter((b) => b.user.id !== U.EMPA.id);
    check('RL-04', r.status === 200 && bad.length === 0, `Daftar EMPA: ${r.data?.length} booking, milik orang lain: ${bad.length}`);
  });
  await scenario('RL-05', async () => {
    const r = await api('GET', '/bookings?limit=100', { token: d1.token });
    const bad = (r.data ?? []).filter((b) => b.assignedDriver?.id !== d1.driverId);
    check('RL-05', r.status === 200 && r.data.length >= 1 && bad.length === 0, `Daftar supir: ${r.data?.length} tugas, bukan miliknya: ${bad.length}`);
  });
  await scenario('RL-06', async () => {
    const r = await api('GET', `/bookings?resourceId=${vRL.resourceId}&limit=100`, { token: U.EMPB.token });
    check('RL-06', (r.data ?? []).some((b) => b.id === bRL.data.id), `Kalender resource dibuka EMPB memuat booking EMPA: ${(r.data ?? []).some((b) => b.id === bRL.data.id)}`);
  });
  await scenario('RL-07', async () => {
    const b = await book(U.EMPA.token, vRL.resourceId, wib(1, 13), wib(1, 14));
    const r = await cancelB(b.data.id, d2.token);
    const r2 = await cancelB((await book(U.EMPA.token, vRL.resourceId, wib(1, 15), wib(1, 16))).data.id, U.RK1.token);
    check('RL-07', r.status === 403 && r2.status === 403,
      `Supir/penjaga membatalkan booking PENDING orang lain → supir ${r.status}, penjaga ${r2.status}`,
      `Supir (${r.status}) & penjaga (${r2.status}) BISA membatalkan booking PENDING milik EMPA`);
  });
  await scenario('RL-08', async () => {
    const r = await api('POST', '/fuel-expenses', { token: U.EMPA.token, form: { vehicleId: vRL.id, fuelTypeId: 1, liter: 5, odometerBefore: 10000, odometerAfter: 10010 } });
    check('RL-08', r.status === 403, `EMPLOYEE catat BBM → ${r.status}`, `EMPLOYEE lolos otorisasi catat BBM (${r.status} "${r.msg}")`);
  });
  await scenario('RL-09', async () => {
    const r1 = await api('GET', '/reports/overview', { token: U.EMPA.token });
    const r2 = await api('GET', '/maintenance', { token: U.EMPA.token });
    const r3 = await api('GET', '/users', { token: d1.token });
    check('RL-09', [r1, r2, r3].every((r) => r.status === 403), `Laporan ${r1.status}, maintenance ${r2.status}, pengguna(oleh supir) ${r3.status}`);
  });

  // ── BC ────────────────────────────────────────────────────────────────
  await scenario('BC-01', async () => {
    const d = await newDriver('DRVBC1'); const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(1, 9), wib(1, 12), { driverId: d.driverId, passengerCount: 3 });
    const notif = await hasNotif(U.ADM.token, 'BOOKING_CREATED', b.data?.id);
    check('BC-01', b.status === 201 && b.data.status === 'PENDING' && notif && b.changed === 'booking',
      `PENDING, notif admin=${notif}, X-Data-Changed=${b.changed}`);
  });
  await scenario('BC-02', async () => {
    const r = await newRoom();
    const b = await book(U.EMPA.token, r.resourceId, wib(1, 9), wib(1, 10));
    check('BC-02', b.status === 201 && b.data.status === 'PENDING', `Booking ruangan → ${b.status} ${b.data?.status}`);
  });
  await scenario('BC-03', async () => {
    const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(1, 12), wib(1, 9));
    check('BC-03', b.status === 400, `Jam selesai < mulai → ${b.status} "${b.msg}"`);
  });
  await scenario('BC-04', async () => {
    const d = await newDriver('DRVBC4'); const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(-1, 9), wib(-1, 12), { driverId: d.driverId });
    check('BC-04', b.status >= 400, `Tanggal masa lalu ditolak (${b.status})`, `Booking KEMARIN diterima (${b.status} ${b.data?.status})`);
    if (b.status === 201) await cancelB(b.data.id, U.EMPA.token);
  });
  await scenario('BC-05', async () => {
    const v = await newVehicle({ status: 'MAINTENANCE' });
    const b = await book(U.EMPA.token, v.resourceId, wib(1, 9), wib(1, 10));
    check('BC-05', b.status === 409, `Kendaraan MAINTENANCE → ${b.status} "${b.msg}"`);
  });
  await scenario('BC-06/07', async () => {
    const d = await newDriver('DRVBC6'); const v = await newVehicle();
    const m = await maintPlan(v.id, wib(3, 8), 2); // diajukan: H+3 08.00 s.d. H+5 08.00
    const statusAfter = sql(`select status from resources where id=${v.resourceId}`);
    const b7 = await book(U.EMPA.token, v.resourceId, wib(5, 9), wib(5, 10), { driverId: d.driverId });
    check('BC-07', b7.status === 201, `Booking H+5 (di luar jadwal maintenance H+3..H+4) diterima`,
      `Booking H+5 DITOLAK ${b7.status} "${b7.msg}" — kendaraan sudah berstatus ${statusAfter} sejak maintenance terjadwal dibuat`);
    await setVehicleStatus(v, 'AVAILABLE'); // isolasi: uji pengecekan jadwal saja
    const b6 = await book(U.EMPA.token, v.resourceId, wib(3, 9), wib(3, 10), { driverId: d.driverId });
    check('BC-06', b6.status === 409, `Booking di tanggal maintenance → ${b6.status} "${b6.msg}"`);
    void m;
  });
  await scenario('BC-08', async () => {
    const v = await newVehicle({ status: 'INACTIVE' });
    const b = await book(U.EMPA.token, v.resourceId, wib(1, 9), wib(1, 10));
    check('BC-08', b.status >= 400, `Kendaraan INACTIVE ditolak (${b.status})`, `Kendaraan INACTIVE BISA dibooking (${b.status} ${b.data?.status})`);
  });
  await scenario('BC-09', async () => {
    const d = await newDriver('DRVBC9'); const v = await newVehicle();
    sql(`update resources set status='IN_USE' where id=${v.resourceId}`); // IN_USE tidak bisa diset lewat API (bagus)
    const b = await book(U.EMPA.token, v.resourceId, wib(1, 9), wib(1, 10), { driverId: d.driverId });
    check('BC-09', b.status === 201, `Kendaraan IN_USE sekarang, booking besok → ${b.status}`);
  });
  await scenario('BC-10', async () => {
    const d = await newDriver('DRVBC10'); const v = await newVehicle();
    const a = await book(U.EMPA.token, v.resourceId, wib(1, 9), wib(1, 12), { driverId: d.driverId });
    const b = await book(U.EMPB.token, v.resourceId, wib(1, 9), wib(1, 12), { driverId: d.driverId });
    check('BC-10', a.status === 201 && b.status === 201, `Dua booking PENDING kendaraan & jam sama → ${a.status}, ${b.status}`);
  });
  await scenario('BC-11', async () => {
    const r = await newRoom();
    const a = await book(U.EMPA.token, r.resourceId, wib(2, 9), wib(2, 11));
    const b = await book(U.EMPB.token, r.resourceId, wib(2, 10), wib(2, 12));
    const pa = await approve(a.data.id); const pb = await approve(b.data.id);
    check('BC-11', ok2(pa) && pb.status === 409, `Booking ruangan kedua yang bentrok ditolak saat approve (${pb.status})`,
      `DUA rapat bentrok di ruangan yang sama sama-sama DISETUJUI (${pa.status}, ${pb.status})`);
  });
  await scenario('BC-12', async () => {
    const d = await newDriver('DRVBC12'); const v = await newVehicle({ capacity: 6 });
    const b = await book(U.EMPA.token, v.resourceId, wib(2, 13), wib(2, 14), { driverId: d.driverId, passengerCount: 10 });
    const a = await approve(b.data.id);
    check('BC-12', b.status === 201 && ok2(a) && !!a.json?.warning, `10 penumpang di kapasitas 6: dibuat, approve + peringatan "${a.json?.warning}"`);
  });
  await scenario('BC-13', async () => {
    const dx = await newDriver('DRVX');
    await api('PATCH', `/drivers/${dx.driverId}/toggle-active`, { token: A() });
    const v = await newVehicle();
    const b = await book(U.EMPA.token, v.resourceId, wib(1, 9), wib(1, 10), { driverId: dx.driverId });
    check('BC-13', b.status === 400, `Pilih supir nonaktif → ${b.status} "${b.msg}"`);
  });
  await scenario('BC-14', async () => {
    const d = await newDriver('DRVBC14'); const v = await newVehicle();
    await bookApproved(U.EMPA.token, v.resourceId, wib(6, 8), wib(6, 10), { driverId: d.driverId, bookingType: 'SPD' });
    const b = await book(U.EMPB.token, v.resourceId, wib(6, 15), wib(6, 17), { driverId: d.driverId });
    check('BC-14', b.status === 409, `Kendaraan SPD pagi, booking sore hari yang sama → ${b.status} "${b.msg}"`);
  });
  await scenario('BC-15', async () => {
    const d = await newDriver('DRVBC15'); const d2b = await newDriver('DRVBC15B'); const v = await newVehicle();
    await bookApproved(U.EMPA.token, v.resourceId, wib(7, 8), wib(7, 10), { driverId: d.driverId });
    const b = await book(U.EMPB.token, v.resourceId, wib(7, 15), wib(7, 17), { driverId: d2b.driverId, bookingType: 'SPD' });
    check('BC-15', b.status === 409, `SPD baru di hari yang sudah terisi NON_SPD → ${b.status} "${b.msg}"`);
  });
  await scenario('BC-16', async () => {
    const d = await newDriver('DRVBC16'); const d2b = await newDriver('DRVBC16B'); const v = await newVehicle();
    await bookApproved(U.EMPA.token, v.resourceId, wib(8, 20), wib(9, 6), { driverId: d.driverId, bookingType: 'SPD' });
    const b = await book(U.EMPB.token, v.resourceId, wib(9, 15), wib(9, 17), { driverId: d2b.driverId });
    check('BC-16', b.status === 409, `SPD lintas malam memblokir hari berikutnya → ${b.status} "${b.msg}"`);
  });
  await scenario('BC-17', async () => {
    const d = await newDriver('DRVBC17'); const va = await newVehicle(); const vb = await newVehicle();
    await bookApproved(U.EMPA.token, va.resourceId, wib(10, 8), wib(10, 12), { driverId: d.driverId });
    const b = await book(U.EMPB.token, vb.resourceId, wib(10, 9), wib(10, 11), { driverId: d.driverId });
    check('BC-17', b.status === 201 && b.data.assignedVehicle?.id === va.id,
      `Pilih supir yang memegang kendaraan lain → kendaraan booking = kendaraan supir (${b.data?.assignedVehicle?.id === va.id})`);
  });

  // ── SP ────────────────────────────────────────────────────────────────
  await scenario('SP-01/02', async () => {
    const df = await newDriver('DRVSPF'); const dOther = await newDriver('DRVSPO');
    const v = await newVehicle({ fixedDriverId: df.driverId });
    const b1 = await book(U.EMPA.token, v.resourceId, wib(11, 8), wib(11, 9));
    check('SP-01', b1.data?.assignedDriver?.id === df.driverId, `Tanpa pilih supir → supir tetap (${b1.data?.assignedDriver?.name})`);
    const b2 = await book(U.EMPA.token, v.resourceId, wib(11, 10), wib(11, 11), { driverId: dOther.driverId });
    check('SP-02', b2.data?.assignedDriver?.id === df.driverId, `Pilih supir lain → tetap supir tetap (${b2.data?.assignedDriver?.name})`);
  });
  await scenario('SP-03', async () => {
    const dfree = await newDriver('DRVFREE'); const v = await newVehicle();
    const b = await onlyDrivers([dfree.driverId], () => book(U.EMPA.token, v.resourceId, wib(12, 8), wib(12, 9)));
    check('SP-03', b.data?.assignedDriver?.id === dfree.driverId, `Supir kosong otomatis → ${b.data?.assignedDriver?.name}`);
  });
  await scenario('SP-04', async () => {
    const v = await newVehicle();
    const b = await onlyDrivers([], () => book(U.EMPA.token, v.resourceId, wib(12, 10), wib(12, 11)));
    check('SP-04', b.status === 201 && !b.data.assignedDriver, `Tidak ada supir kosong → PENDING tanpa supir (${b.status}, supir=${b.data?.assignedDriver?.name ?? 'kosong'})`);
    U.bookingNoDriver = b.data?.id;
  });
  await scenario('SP-05', async () => {
    const d = await newDriver('DRVSP5'); const v = await newVehicle();
    await api('PATCH', `/drivers/${d.driverId}/toggle-active`, { token: A() });
    const b = await onlyDrivers([d.driverId], () => book(U.EMPA.token, v.resourceId, wib(12, 12), wib(12, 13)));
    check('SP-05', !b.data?.assignedDriver, `Supir nonaktif (menu Driver) tidak dipilih otomatis (supir=${b.data?.assignedDriver?.name ?? 'kosong'})`);
  });
  await scenario('SP-06', async () => {
    const d = await newDriver('DRVSP6'); const v = await newVehicle();
    await api('PATCH', `/users/${d.id}/toggle-active`, { token: A() });
    const b = await onlyDrivers([d.driverId], () => book(U.EMPA.token, v.resourceId, wib(12, 14), wib(12, 15)));
    const av = await api('GET', `/drivers/available?startDate=${encodeURIComponent(wib(12, 14))}&endDate=${encodeURIComponent(wib(12, 15))}`, { token: A() });
    const inPicker = (av.data ?? []).some((x) => x.driverId === d.driverId);
    check('SP-06', !b.data?.assignedDriver && !inPicker, `Akun supir nonaktif (menu Pengguna): tidak otomatis & tidak di picker`,
      `Akun nonaktif: otomatis=${b.data?.assignedDriver?.name ?? '-'}, di picker=${inPicker}`);
  });
  await scenario('SP-07', async () => {
    const ds = await newDriver('DRVSP7'); const vf = await newVehicle({ fixedDriverId: ds.driverId }); const vg = await newVehicle();
    await bookApproved(U.EMPA.token, vg.resourceId, wib(13, 8), wib(13, 10), { driverId: ds.driverId, bookingType: 'SPD' });
    const b = await book(U.EMPB.token, vf.resourceId, wib(13, 14), wib(13, 16));
    check('SP-07', b.status === 409, `Supir tetap sedang SPD di kendaraan lain → ${b.status} "${b.msg}"`);
  });
  await scenario('SP-08/09/10', async () => {
    const da = await newDriver('DRVSP8A'); const db = await newDriver('DRVSP8B');
    const v1 = await newVehicle({ fixedDriverId: da.driverId }); const v3 = await newVehicle();
    await api('PATCH', `/vehicles/${v1.id}/fixed-driver`, { token: A(), body: { driverId: db.driverId } });
    const ga = await api('GET', `/drivers/${da.driverId}`, { token: A() }); const gb = await api('GET', `/drivers/${db.driverId}`, { token: A() });
    check('SP-08', !ga.data?.fixedVehicle && gb.data?.fixedVehicle?.id === v1.id, `Pindah supir tetap dari menu Kendaraan tersinkron di menu Driver`);
    await api('PATCH', `/drivers/${db.driverId}/fixed-vehicle`, { token: A(), body: { vehicleId: v3.id } });
    const f1 = sqlInt(`select "fixedDriverId" from vehicles where id=${v1.id}`); const f3 = sqlInt(`select "fixedDriverId" from vehicles where id=${v3.id}`);
    check('SP-09', f1 === null && f3 === db.driverId, `Ubah kendaraan tetap dari menu Driver: kendaraan lama kosong, baru terisi`);
    await api('PATCH', `/vehicles/${v3.id}/fixed-driver`, { token: A(), body: { driverId: null } });
    const f3b = sqlInt(`select "fixedDriverId" from vehicles where id=${v3.id}`);
    check('SP-10', f3b === null, `Hapus supir tetap → kosong`);
  });
  await scenario('SP-11', async () => {
    const d = await newDriver('DRVSP11'); const v = await newVehicle({ capacity: 6 });
    await bookApproved(U.EMPA.token, v.resourceId, wib(14, 8), wib(14, 12), { driverId: d.driverId, passengerCount: 4 });
    const av = await api('GET', `/drivers/available?startDate=${encodeURIComponent(wib(14, 9))}&endDate=${encodeURIComponent(wib(14, 10))}`, { token: A() });
    const row = (av.data ?? []).find((x) => x.driverId === d.driverId);
    check('SP-11', row && row.plateNumber === v.plate && row.remainingSeats === 2 && row.overlappingPurpose !== '' && 'isSpdActive' in row,
      `Picker: plat=${row?.plateNumber}, sisa kursi=${row?.remainingSeats}, tujuan bentrok="${row?.overlappingPurpose}"`);
  });
  await scenario('SP-12', async () => {
    const d = await newDriver('DRVSP12'); const va = await newVehicle(); const vb = await newVehicle();
    const a = await onlyDrivers([d.driverId], () => book(U.EMPA.token, va.resourceId, wib(15, 9), wib(15, 12)));
    await bookApproved(U.EMPB.token, vb.resourceId, wib(15, 10), wib(15, 11), { driverId: d.driverId });
    const r = await approve(a.data.id);
    check('SP-12', r.status === 409, `Supir yang sudah bertugas di jam bentrok tidak bisa disetujui lagi (${r.status})`,
      `Supir yang SAMA disetujui untuk 2 perjalanan bentrok di kendaraan berbeda (${r.status})`);
  });
  await scenario('SP-13', async () => {
    const dFixed = await newDriver('DRVSP13F'); const dFree = await newDriver('DRVSP13K');
    await newVehicle({ fixedDriverId: dFixed.driverId });
    const v = await newVehicle();
    const b = await onlyDrivers([dFixed.driverId, dFree.driverId], () => book(U.EMPA.token, v.resourceId, wib(16, 9), wib(16, 10)));
    const got = b.data?.assignedDriver?.id;
    check('SP-13', got === dFree.driverId, 'Supir tanpa kendaraan tetap diutamakan untuk pemilihan otomatis',
      `Pemilihan otomatis mengambil supir tetap kendaraan LAIN (${b.data?.assignedDriver?.name}) padahal ada supir bebas`);
  });
}
