import { writeFileSync } from 'node:fs';
import { results, sql, RUN } from './lib.mjs';
import { bootstrapAdmin, createUser, ROLE, U } from './fixtures.mjs';

const only = process.argv.slice(2); // mis. "1 2" untuk batch tertentu
console.log(`== Runner skenario KCE (run ${RUN}) ==`);
// Pool supir terkendali: supir dari run sebelumnya/seed dinonaktifkan & dilepas.
sql(`update drivers set "isActive"=false`);
sql(`update driver_assignments set "releasedAt"=now() where "releasedAt" is null`);

await bootstrapAdmin();
await createUser('ADM2', ROLE.ADMIN);
await createUser('EMPA', ROLE.EMPLOYEE);
await createUser('EMPB', ROLE.EMPLOYEE);
await createUser('RK1', ROLE.ROOM_KEEPER);
await createUser('RK2', ROLE.ROOM_KEEPER);

const batches = {
  1: async () => (await import('./s1.mjs')).runBatch1(),
  2: async () => (await import('./s2.mjs')).runBatch2(),
  3: async () => (await import('./s3.mjs')).runBatch3(),
  4: async () => (await import('./s4.mjs')).runBatch4(),
};
for (const [k, fn] of Object.entries(batches)) {
  if (only.length && !only.includes(k)) continue;
  console.log(`\n── Batch ${k} ──`);
  await fn();
}

const count = (s) => results.filter((r) => r.status === s).length;
console.log(`\nPASS ${count('PASS')} · FAIL ${count('FAIL')} · INFO ${count('INFO')} · ERROR ${count('ERROR')} · SKIP ${count('SKIP')}`);
writeFileSync(`results-${RUN}.json`, JSON.stringify(results, null, 2));
void U;
