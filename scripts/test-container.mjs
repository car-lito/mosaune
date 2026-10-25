// Run only against a disposable image/container and freshly created volumes.
// Never reads developer media, credentials, or existing Mosaune databases.
import { spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, chmodSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import https from 'node:https';
import net from 'node:net';
import { deflateSync } from 'node:zlib';
import { checkServerIdentity } from 'node:tls';

const image = process.argv[2];
if (!image) throw new Error('Usage: node scripts/test-container.mjs IMAGE');
const id = 'mosaune-ci-' + randomUUID();
const directory = mkdtempSync(join(tmpdir(), id));
const certDirectory = join(directory, 'tls');
const mediaDirectory = join(directory, 'media');
// Synthetic ID3/PNG fixture: exercises the WASI reader and disk artwork staging
// inside the read-only, unprivileged ARM64 container as well as amd64.
function taggedAudioFixture() {
  function chunk(type,data) {
    const body=Buffer.concat([Buffer.from(type),data]);let crc=0xffffffff;
    for(const byte of body){crc^=byte;for(let i=0;i<8;i++)crc=(crc>>>1)^((crc&1)?0xedb88320:0)}
    const head=Buffer.alloc(4),tail=Buffer.alloc(4);head.writeUInt32BE(data.length);tail.writeUInt32BE((crc^0xffffffff)>>>0);return Buffer.concat([head,body,tail]);
  }
  const ihdr=Buffer.alloc(13);ihdr.writeUInt32BE(1,0);ihdr.writeUInt32BE(1,4);ihdr[8]=8;ihdr[9]=2;
  const png=Buffer.concat([Buffer.from('89504e470d0a1a0a','hex'),chunk('IHDR',ihdr),chunk('IDAT',deflateSync(Buffer.from([0,200,80,40]))),chunk('IEND',Buffer.alloc(0))]);
  function frame(id,data){const h=Buffer.alloc(10);h.write(id);h.writeUInt32BE(data.length,4);return Buffer.concat([h,data])}
  const payload=Buffer.concat([frame('TIT2',Buffer.from('\x03Morceau conteneur')),frame('TPE1',Buffer.from('\x03Artiste test')),frame('TALB',Buffer.from('\x03Album test')),frame('APIC',Buffer.concat([Buffer.from('\x00image/png\x00\x03\x00'),png]))]);
  const h=Buffer.from([73,68,51,3,0,0,0,0,0,0]);for(let i=0;i<4;i++)h[9-i]=(payload.length>>>(i*7))&127;return Buffer.concat([h,payload]);
}
const taggedAudio=taggedAudioFixture();
const fixtures = [
  { kind: 'music', name: 'sample.mp3' },
  { kind: 'films', name: 'Iron.Man.2008.mkv' },
  { kind: 'series', name: 'Example.S01E02.Title.mkv' },
];
mkdirSync(certDirectory); mkdirSync(mediaDirectory);
let created = false, volumeCreated = false;
function command(exe, args, allowFailure = false) {
  const result = spawnSync(exe, args, { encoding: 'utf8', timeout: 120000, maxBuffer: 8 * 1024 * 1024 });
  if (result.status !== 0 && !allowFailure) throw new Error(`${exe} ${args[0]} failed: ${result.stderr || result.error?.message || 'nonzero exit'}`);
  return result;
}
const docker = args => command('docker', args).stdout.trim();
function assert(value, message) { if (!value) throw new Error(message); }
const probe = net.createServer();
await new Promise(resolve => probe.listen(0, '127.0.0.1', resolve));
const port = probe.address().port;
await new Promise(resolve => probe.close(resolve));
const origin = `https://127.0.0.1:${port}`;
try {
  command('openssl', ['req', '-x509', '-newkey', 'rsa:2048', '-sha256', '-nodes', '-days', '1', '-subj', '/CN=Mosaune CI temporary', '-addext', 'subjectAltName=IP:127.0.0.1', '-keyout', join(certDirectory, 'server.key'), '-out', join(certDirectory, 'server.crt')]);
  // Test key is disposable and mounted in an isolated container, never a real key.
  chmodSync(certDirectory, 0o755); chmodSync(join(certDirectory, 'server.key'), 0o644);
  for (const fixture of fixtures) {
    mkdirSync(join(mediaDirectory, fixture.kind));
    writeFileSync(join(mediaDirectory, fixture.kind, fixture.name), fixture.kind==='music'?taggedAudio:'0123456789');
  }
  mkdirSync(join(mediaDirectory,'external'));
  writeFileSync(join(mediaDirectory,'external','Contributor.2026.mkv'),'external-fixture');
  docker(['volume', 'create', id]); volumeCreated = true;
  docker(['create', '--name', id, '--read-only', '--cap-drop=ALL', '--security-opt=no-new-privileges:true', '--pids-limit=256', '--tmpfs', '/tmp:size=64m,mode=1777', '-p', `127.0.0.1:${port}:47831`,
    '-e', `MOSAUNE_PUBLIC_URL=${origin}`, '-e', 'MOSAUNE_TLS_CERT=/tls/server.crt', '-e', 'MOSAUNE_TLS_KEY=/tls/server.key', '-e', 'MOSAUNE_TLS_CA=/tls/server.crt',
    '--mount', `type=volume,source=${id},target=/data`, '--mount', `type=bind,source=${certDirectory},target=/tls,readonly`, '--mount', `type=bind,source=${mediaDirectory},target=/media,readonly`, image]);
  created = true;
  const inspect = JSON.parse(docker(['inspect', id]))[0];
  assert(inspect.Config.User === '65532:65532' && inspect.HostConfig.ReadonlyRootfs, 'Container is not unprivileged/read-only');
  assert(inspect.Mounts.find(m => m.Destination === '/media')?.RW === false, 'Media is writable');
  const ca = readFileSync(join(certDirectory, 'server.crt'));
  function request(path, { method = 'GET', body, headers = {}, timeout = 10000 } = {}) {
    const started = Date.now();
    return new Promise((resolve, reject) => {
      // Verify the actual TLS destination even when testing a forged HTTP Host.
      // Node otherwise derives the TLS server name from that injected header.
      const req = https.request(origin + path, { method, ca, headers, timeout,
        servername: '', checkServerIdentity: (_, certificate) => checkServerIdentity('127.0.0.1', certificate),
      }, response => {
        const chunks = []; response.on('data', chunk => chunks.push(chunk));
        response.on('end', () => {
          console.log(`${method} ${path}: HTTP ${response.statusCode}, ${Date.now() - started} ms`);
          resolve({ status: response.statusCode, headers: response.headers, text: Buffer.concat(chunks).toString() });
        });
      });
      req.on('error', reject); req.on('timeout', () => req.destroy(new Error(`${method} ${path} exceeded ${timeout} ms`)));
      req.end(body === undefined ? undefined : Buffer.isBuffer(body) ? body : JSON.stringify(body));
    });
  }
  async function ready() {
    for (let i = 0; i < 120; i++) {
      try { if ((await request('/healthz')).status === 200) return; } catch {}
      const state = JSON.parse(docker(['inspect', id]))[0].State;
      if (!state.Running) throw new Error('Disposable container stopped during startup (logs intentionally not printed)');
      await new Promise(resolve => setTimeout(resolve, 500));
    }
    throw new Error('HTTPS health check timed out');
  }
  docker(['start', id]); await ready();
  docker(['exec', id, '/mosaune', '-healthcheck']);
  assert((await request('/healthz', { headers: { Host: 'attacker.invalid' } })).status === 403, 'Foreign Host accepted');
  assert((await request('/api/v1/auth/state', { headers: { Origin: 'https://attacker.invalid' } })).status === 403, 'Foreign Origin accepted');
  const state = JSON.parse((await request('/api/v1/auth/state')).text);
  const logResult = command('docker', ['logs', id]);
  const logs = logResult.stdout + logResult.stderr;
  const code = logs.match(/configuration \/ récupération : (\S+)/)?.[1];
  assert(state.setup && code, 'Initial setup not available');
  const password = randomUUID();
  const setup = await request('/api/v1/auth/setup', { method: 'POST', headers: { Origin: origin, 'X-Mosaune-Token': state.token }, body: { name: 'ContainerTest', password, code } });
  assert(setup.status === 200, 'Admin setup failed');
  const cookie = setup.headers['set-cookie']?.[0];
  assert(cookie?.includes('Secure') && cookie.includes('HttpOnly') && cookie.includes('SameSite=Strict'), 'Unsafe session cookie');
  const auth = { Cookie: cookie.split(';')[0], Origin: origin, 'X-Mosaune-Token': JSON.parse(setup.text).token };
  // Under ARM emulation the initial WASI compilation can exceed the ordinary
  // HTTP probe budget. This exceeds the host's bounded compilation/call budget
  // so a real host error remains visible instead of being masked by the probe.
  for (const fixture of fixtures) {
    const scan = await request('/api/v1/libraries', { method: 'POST', timeout: 30000, headers: auth, body: { name: `CI ${fixture.kind}`, kind: fixture.kind, path: `/media/${fixture.kind}` } });
    assert(scan.status === 201, `${fixture.kind} read-only scan failed: HTTP ${scan.status}, ${scan.text}`);
  }
  const library = JSON.parse((await request('/api/v1/library', { headers: auth })).text);
  assert(library.items.length === fixtures.length, 'Sample media were not catalogued');
  const movie = library.items.find(item => item.kind === 'films');
  const episode = library.items.find(item => item.kind === 'series');
  const music = library.items.find(item => item.kind === 'music');
  assert(movie?.metadata?.year === 2008 && episode?.metadata?.season === 1 && episode?.metadata?.episode === 2 && music, 'Media recognizers did not return expected metadata');
  const streamed = await request(music.url, { headers: { ...auth, Range: 'bytes=2-5' } });
  assert(streamed.status === 206 && streamed.text === taggedAudio.subarray(2,6).toString(), 'HTTPS range streaming failed');
  assert(music.metadata?.title==='Morceau conteneur'&&music.metadata.artist==='Artiste test'&&music.metadata.album==='Album test'&&music.metadata.imageId,'Audio tag module or artwork staging failed');
  const cover=await request('/api/v1/artwork/'+music.metadata.imageId,{headers:auth});assert(cover.status===200&&cover.headers['content-type']==='image/jpeg','Embedded picture unavailable');
  const tagsState=JSON.parse((await request('/api/v1/audio-tags',{headers:auth})).text);
  assert(tagsState.installed&&tagsState.enabled,'Audio tag extension missing');
  assert((await request('/api/v1/audio-tags',{method:'POST',headers:auth,body:{operation:'disable',revision:tagsState.revision}})).status===200,'Audio tag disable failed');
  const packages = JSON.parse((await request('/api/v1/recognition-packages', { headers: auth })).text);
  const removal = await request('/api/v1/recognition-packages/films', { method: 'POST', headers: auth, body: { operation: 'remove', revision: packages.revision } });
  assert(removal.status === 200, 'Recognition package removal failed');
  docker(['stop', '--time', '15', id]);
  assert(JSON.parse(docker(['inspect', id]))[0].State.ExitCode === 0, 'SIGTERM did not stop cleanly');
  docker(['start', id]); await ready();
  const after = JSON.parse((await request('/api/v1/auth/state')).text);
  assert(!after.setup, 'Account lost after restart');
  const login = await request('/api/v1/auth/login', { method: 'POST', headers: { Origin: origin, 'X-Mosaune-Token': after.token }, body: { name: 'ContainerTest', password } });
  assert(login.status === 200, 'Login after restart failed');
  const persisted = JSON.parse((await request('/api/v1/library', { headers: { Cookie: login.headers['set-cookie'][0].split(';')[0] } })).text);
  assert(persisted.items.length === fixtures.length, 'Catalogue lost after restart');
  const restartedAuth = { Cookie: login.headers['set-cookie'][0].split(';')[0], Origin: origin, 'X-Mosaune-Token': JSON.parse(login.text).token };
  const tagsAfter=JSON.parse((await request('/api/v1/audio-tags',{headers:restartedAuth})).text);assert(tagsAfter.installed&&!tagsAfter.enabled,'Audio tags disabled state lost');
  const cachedCover=await request('/api/v1/artwork/'+music.metadata.imageId,{headers:restartedAuth});assert(cachedCover.status===200,'Cached artwork lost after disabling and restart');
  const removed = JSON.parse((await request('/api/v1/recognition-packages', { headers: restartedAuth })).text);
  assert(!removed.packages.find(p => p.kind === 'films')?.installed, 'Restart restored removed package');
  const install = await request('/api/v1/recognition-packages/films', { method: 'POST', headers: restartedAuth, body: { operation: 'install', revision: removed.revision } });
  assert(install.status === 200, 'Recognition package reinstall failed');
  const installed = JSON.parse((await request('/api/v1/recognition-packages', { headers: restartedAuth })).text);
  assert(installed.packages.find(p => p.kind === 'films')?.installed && !installed.packages.find(p => p.kind === 'films')?.enabled, 'Installation activated package implicitly');
  const enable = await request('/api/v1/recognition-packages/films', { method: 'POST', headers: restartedAuth, body: { operation: 'enable', revision: installed.revision } });
  assert(enable.status === 200, 'Recognition reactivation failed');
  const external = readFileSync('build/example-recognition.mosaune.zip');
  const preview = await request('/api/v1/recognition-packages/preview', { method:'POST',headers:{...restartedAuth,'Content-Type':'application/zip'},body:external });
  assert(preview.status===200,'External package preview failed');
  const candidate=JSON.parse(preview.text);
  const imported=await request('/api/v1/recognition-packages/import',{method:'POST',timeout:30000,headers:{...restartedAuth,'Content-Type':'application/zip','X-Mosaune-Package-Checksum':candidate.package.checksum,'X-Mosaune-Registry-Revision':String(candidate.revision)},body:external});
  assert(imported.status===201,'External package import failed: '+imported.text);
  const externalID=candidate.package.manifest.module.id;
  async function externalAction(operation) {
    const state=JSON.parse((await request('/api/v1/recognition-packages',{headers:restartedAuth})).text);
    const result=await request('/api/v1/recognition-packages/'+externalID,{method:'POST',headers:restartedAuth,body:{operation,revision:state.revision}});
    assert(result.status===200,'External action failed: '+operation);
  }
  await externalAction('enable');await externalAction('select');
  // A new library uses the imported program. Previously enriched metadata is
  // intentionally not overwritten simply because the recognizer changed.
  const externalScan=await request('/api/v1/libraries',{method:'POST',timeout:30000,headers:restartedAuth,body:{name:'External recognition',kind:'films',path:'/media/external'}});
  assert(externalScan.status===201,'External scan failed: '+externalScan.text);
  const recognized=JSON.parse((await request('/api/v1/library',{headers:restartedAuth})).text);
  assert(recognized.items.some(item=>item.metadata?.title?.startsWith('Exemple — ')),'Scan did not execute imported program');
  await externalAction('remove');
  const unavailableScan=await request('/api/v1/libraries',{method:'POST',timeout:30000,headers:restartedAuth,body:{name:'Unavailable recognition',kind:'films',path:'/media/external'}});
  assert(unavailableScan.status!==201,'Removed selection silently fell back to factory module');
  for (const fixture of fixtures) assert(readFileSync(join(mediaDirectory, fixture.kind, fixture.name)).equals(fixture.kind==='music'?taggedAudio:Buffer.from('0123456789')), 'Source media modified');
  assert(readFileSync(join(mediaDirectory,'external','Contributor.2026.mkv'),'utf8')==='external-fixture','External source media modified');
  docker(['exec', id, '/mosaune', '-healthcheck']);
  docker(['stop', '--time', '15', id]);
  console.log(JSON.stringify({ image, https: 'verified', cookies: 'secure', media: 'read-only + range verified', restart: 'account and catalogue preserved', sigterm: 'exit 0' }));
} finally {
  if (created) command('docker', ['rm', '-f', id], true);
  if (volumeCreated) command('docker', ['volume', 'rm', id], true);
  // This directory was exclusively created by mkdtemp in this process.
  rmSync(directory, { recursive: true, force: true });
}
