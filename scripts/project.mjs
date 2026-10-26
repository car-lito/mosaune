// Entrée commune aux postes de développement et à GitHub Actions.
// Les commandes sont bornées : aucun serveur personnel n'est lancé ou arrêté.
import { spawnSync } from 'node:child_process';
import { mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve, dirname, relative, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';
import { buildNotices } from './build-notices.mjs';
import { buildRecognitionRepository } from './build-recognition-repository.mjs';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const go = process.env.MOSAUNE_GO || 'go';
const env = { ...process.env, CGO_ENABLED: '0', GOTELEMETRY: 'off', GOOS: '', GOARCH: '' };
const mode = process.argv[2];
if (!['check', 'build', 'ci'].includes(mode)) {
  console.error('Usage : node scripts/project.mjs check|build|ci');
  process.exit(2);
}
function run(command, args, directory = root, extra = {}) {
  console.log(`\n> ${relative(root, directory) || '.'} : ${command === go ? 'go' : 'node'} ${args.join(' ')}`);
  const result = spawnSync(command, args, { cwd: directory, env: { ...env, ...extra }, stdio: 'inherit', windowsHide: true });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`La commande a échoué (${result.status ?? result.signal}).`);
}
const kit = join(root, 'extension-kit');
function wasm(output, source) {
  run(go, ['build', '-trimpath', '-ldflags=-s -w', '-o', output, source], kit, { GOOS: 'wasip1', GOARCH: 'wasm' });
}
try {
  wasm('providers/tmdb/provider.wasm', './providers/tmdb/source');
  wasm('modules/audiotags/tags.wasm', './modules/audiotags/source');
  for (const kind of ['films', 'series', 'music', 'catalogue-films', 'catalogue-series', 'catalogue-music', 'detail-series']) wasm(`modules/${kind}/recognizer.wasm`, `./modules/${kind}/source`);
  wasm('examples/recognition/module.wasm', './examples/recognition/source');
  wasm('examples/audiotags/module.wasm', './examples/audiotags/source');
  wasm('examples/detaillist/module.wasm', './examples/detaillist/source');
  wasm('examples/catalogue/module.wasm', './examples/catalogue/source');
  mkdirSync(join(root, 'build'), { recursive: true });
  run(go, ['run', './cmd/package-module', 'examples/detaillist/extension.json', 'examples/detaillist/module.wasm', '../build/example-detaillist.mosaune.zip'], kit);
  run(go, ['run', './cmd/package-module', 'examples/recognition/extension.json', 'examples/recognition/module.wasm', '../build/example-recognition.mosaune.zip'], kit);
  run(go, ['run', './cmd/package-module', 'examples/catalogue/extension.json', 'examples/catalogue/module.wasm', '../build/example-catalogue.mosaune.zip'], kit);
  buildRecognitionRepository(root, go, env);
  buildRecognitionRepository(root, go, env, "audiotags");
  // Le serveur embarque l'interface : elle est nécessaire même pour les tests Go.
  run(process.execPath, ['node_modules/vite/bin/vite.js', 'build']);
  if (mode !== 'build') {
    wasm('examples/demo/provider.wasm', './examples/demo/source');
    wasm('runtime/testdata/probe.wasm', './runtime/testdata/probe');
    run(process.execPath, ['node_modules/svelte-check/bin/svelte-check', '--tsconfig', './tsconfig.json']);
    const tests = readdirSync(join(root, 'web/src')).filter(name => name.endsWith('.test.mjs')).sort().map(name => `web/src/${name}`);
    run(process.execPath, ['--test', ...tests]);
    for (const directory of [kit, root]) {
      // The server suite creates many durable SQLite catalogues; Windows disk flushes vary.
      // This is a suite allowance, not an extension execution-time limit.
      run(go, ['test', '-timeout', directory === root ? '12m' : '5m', './...'], directory);
      run(go, ['vet', './...'], directory);
    }
  }
  if (mode !== 'check') {
    const { version } = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
    const serverVersion = readFileSync(join(root, 'main.go'), 'utf8').match(/const version = "([^"]+)"/)?.[1];
    if (version !== serverVersion) throw new Error('La version du serveur et celle du paquet ne correspondent pas.');
    mkdirSync(join(root, 'build'), { recursive: true });
    const checksums = [];
    for (const [os, arch, suffix] of [['windows', 'amd64', '.exe'], ['linux', 'amd64', ''], ['linux', 'arm64', '']]) {
      const name = `mosaune-${version}-${os}-${arch}${suffix}`;
      run(go, ['build', '-trimpath', '-ldflags=-s -w', '-o', `build/${name}`, '.'], root, { GOOS: os, GOARCH: arch });
      const hash = createHash('sha256').update(readFileSync(join(root, 'build', name))).digest('hex');
      checksums.push(`${hash}  ${name}`);
    }
    writeFileSync(join(root, 'build', `mosaune-${version}-SHA256SUMS.txt`), `${checksums.join('\n')}\n`);
    buildNotices(root, go, env);
  }
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
