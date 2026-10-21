// Run only against a disposable image/container and freshly created volumes.
// Never reads developer media, credentials, or existing Mosaune databases.
import { spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, chmodSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import https from 'node:https';
import net from 'node:net';
import { checkServerIdentity } from 'node:tls';

const image = process.argv[2];
if (!image) throw new Error('Usage: node scripts/test-container.mjs IMAGE');
const id = 'mosaune-ci-' + randomUUID();
const directory = mkdtempSync(join(tmpdir(), id));
const certDirectory = join(directory, 'tls');
const mediaDirectory = join(directory, 'media');
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
  writeFileSync(join(mediaDirectory, 'sample.mp3'), '0123456789');
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
      req.end(body === undefined ? undefined : JSON.stringify(body));
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
  const scan = await request('/api/v1/libraries', { method: 'POST', timeout: 30000, headers: auth, body: { name: 'CI only', kind: 'music', path: '/media' } });
  assert(scan.status === 201, `Read-only media scan failed: HTTP ${scan.status}, ${scan.text}`);
  const library = JSON.parse((await request('/api/v1/library', { headers: auth })).text);
  assert(library.items.length === 1, 'Sample media was not catalogued');
  const streamed = await request(library.items[0].url, { headers: { ...auth, Range: 'bytes=2-5' } });
  assert(streamed.status === 206 && streamed.text === '2345', 'HTTPS range streaming failed');
  docker(['stop', '--time', '15', id]);
  assert(JSON.parse(docker(['inspect', id]))[0].State.ExitCode === 0, 'SIGTERM did not stop cleanly');
  docker(['start', id]); await ready();
  const after = JSON.parse((await request('/api/v1/auth/state')).text);
  assert(!after.setup, 'Account lost after restart');
  const login = await request('/api/v1/auth/login', { method: 'POST', headers: { Origin: origin, 'X-Mosaune-Token': after.token }, body: { name: 'ContainerTest', password } });
  assert(login.status === 200, 'Login after restart failed');
  const persisted = JSON.parse((await request('/api/v1/library', { headers: { Cookie: login.headers['set-cookie'][0].split(';')[0] } })).text);
  assert(persisted.items.length === 1, 'Catalogue lost after restart');
  assert(readFileSync(join(mediaDirectory, 'sample.mp3'), 'utf8') === '0123456789', 'Source media modified');
  docker(['exec', id, '/mosaune', '-healthcheck']);
  docker(['stop', '--time', '15', id]);
  console.log(JSON.stringify({ image, https: 'verified', cookies: 'secure', media: 'read-only + range verified', restart: 'account and catalogue preserved', sigterm: 'exit 0' }));
} finally {
  if (created) command('docker', ['rm', '-f', id], true);
  if (volumeCreated) command('docker', ['volume', 'rm', id], true);
  // This directory was exclusively created by mkdtemp in this process.
  rmSync(directory, { recursive: true, force: true });
}
