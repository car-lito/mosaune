// Exercise the shipped executable with an isolated catalogue and no browser.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { randomBytes } from 'node:crypto';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import net from 'node:net';
import os from 'node:os';
import path from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

const version = JSON.parse(await readFile(new URL('../package.json', import.meta.url))).version;
const platform = process.platform === 'win32' ? 'windows' : process.platform;
const arch = process.arch === 'x64' ? 'amd64' : process.arch;
const binary = path.resolve(process.argv[2] || `build/mosaune-${version}-${platform}-${arch}${platform === 'windows' ? '.exe' : ''}`);
const temporaryRoot = path.resolve(os.tmpdir());
const directory = await mkdtemp(path.join(temporaryRoot, 'mosaune-minimal-'));
const password = randomBytes(24).toString('hex');
let child, completion, origin, output, startupError;
let auth = {};

async function request(endpoint, body, expected = 200) {
  const response = await fetch(origin + endpoint, {
    method: body === undefined ? 'GET' : 'POST',
    headers: { ...auth, Origin: origin, 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(30000),
  });
  assert.equal(response.status, expected, `${endpoint}: unexpected status`);
  return response;
}

async function start(restart = false) {
  const probe = net.createServer();
  await new Promise((resolve, reject) => { probe.once('error', reject); probe.listen(0, '127.0.0.1', resolve); });
  const port = probe.address().port;
  await new Promise(resolve => probe.close(resolve));
  origin = `http://127.0.0.1:${port}`;
  output = '';
  startupError = undefined;
  auth = {};
  child = spawn(binary, ['-data-dir', directory, '-listen', '127.0.0.1', '-port', String(port),
    '-public-url=', '-tls-cert=', '-tls-key=', '-tls-ca=',
    ...(restart ? ['-providers', 'none', '-initial-media', 'films,series,music'] : ['-initial-home=false'])], {
    windowsHide: true,
    env: { ...process.env, MOSAUNE_PROVIDERS: restart ? 'tmdb' : 'none', MOSAUNE_INITIAL_MEDIA: 'none' },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  completion = new Promise(resolve => {
    child.once('error', error => { startupError = error; });
    child.once('close', resolve);
  });
  // Startup logs contain the one-time setup code: keep them private.
  for (const stream of [child.stdout, child.stderr]) stream.on('data', value => { output = (output + value).slice(-32768); });
  for (let attempt = 0; attempt < 200; attempt++) {
    if (startupError) throw startupError;
    if (child.exitCode !== null) throw Error('Minimal server exited before becoming ready');
    try {
      const response = await fetch(origin + '/healthz', { signal: AbortSignal.timeout(1000) });
      if (response.ok) return;
    } catch { /* Retry only during bounded startup. */ }
    await delay(100);
  }
  throw Error('Minimal server startup timed out');
}

async function stop() {
  if (!child) return;
  if (child.exitCode === null) child.kill();
  let timer;
  try {
    await Promise.race([completion, new Promise((_, reject) => {
      timer = setTimeout(() => { child.kill('SIGKILL'); reject(Error('Minimal server shutdown timed out')); }, 10000);
    })]);
  } finally { clearTimeout(timer); }
  child = undefined;
}

async function enter(setup) {
  const state = await (await request('/api/v1/auth/state')).json();
  auth = { 'X-Mosaune-Token': state.token };
  const body = { name: 'MinimalOwner', password };
  if (setup) {
    body.code = output.match(/Code unique[^\r\n]* : ([^\r\n]+)/)?.[1];
    assert(body.code, 'Missing first-run setup code');
  }
  const response = await request(`/api/v1/auth/${setup ? 'setup' : 'login'}`, body);
  const cookie = response.headers.get('set-cookie');
  assert(cookie, 'Missing authenticated session');
  auth = { Cookie: cookie.split(';')[0], 'X-Mosaune-Token': (await response.json()).token };
}

async function composition(musicInstalled) {
  assert.deepEqual(await (await request('/api/v1/metadata/providers')).json(), []);
  assert.deepEqual(await (await request('/api/v1/metadata/provider-management')).json(), []);
  const { bundles } = await (await request('/api/v1/media-bundles')).json();
  assert.equal(bundles.length, 3);
  for (const bundle of bundles) {
    assert.equal(Boolean(bundle.installedVersion), musicInstalled && bundle.manifest.kind === 'music', bundle.manifest.kind);
  }
  for (const endpoint of ['/api/v1/status', '/api/v1/library', '/api/v1/extension-repositories', '/api/v1/acquisition/settings']) await request(endpoint);
  return bundles;
}

try {
  await start();
  await request('/api/v1/metadata/providers', undefined, 401);
  await enter(true);
  const music = (await composition(false)).find(bundle => bundle.manifest.kind === 'music');
  const plan = await (await request('/api/v1/media-bundles/music/preview', { operation: 'prepare', revision: music.revision })).json();
  await request('/api/v1/media-bundles/music/apply', { operation: 'prepare', revision: plan.revision, checksum: plan.checksum });
  await composition(true);
  await stop();
  // Explicit flags override the environment; initial defaults must not seed an existing catalogue.
  await start(true);
  await enter(false);
  await composition(true);
  console.log('Minimal executable: setup, isolated music installation, settings and restart passed.');
} finally {
  await stop();
  const resolved = path.resolve(directory);
  assert.equal(path.dirname(resolved), temporaryRoot, 'Refusing cleanup outside the temporary root');
  assert(path.basename(resolved).startsWith('mosaune-minimal-'));
  await rm(resolved, { recursive: true, force: true });
}
