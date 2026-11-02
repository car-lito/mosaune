import { readFileSync } from 'node:fs';
import { resolve, dirname, posix } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');

// Compile the same assets Go embeds, in local builds and Docker. Adding an
// embedded module cannot silently omit it from a second compilation list.
export function buildBundledWasm(compile) {
  compile('providers/tmdb/provider.wasm', './providers/tmdb/source');
  const source = readFileSync(resolve(root, 'extension-kit/modules/bundled.go'), 'utf8');
  const assets = [...source.matchAll(/^\/\/go:embed (.+)$/gm)]
    .flatMap(match => match[1].trim().split(/\s+/)).filter(asset => asset.endsWith('.wasm'));
  if (!assets.length) throw new Error('Aucun programme intégré déclaré.');
  for (const asset of new Set(assets)) {
    if (!/^[a-z0-9-]+\/[a-z0-9-]+\.wasm$/.test(asset)) throw new Error('Chemin de programme invalide.');
    compile(`modules/${asset}`, `./modules/${posix.dirname(asset)}/source`);
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  buildBundledWasm((output, source) => {
    const result = spawnSync(process.env.MOSAUNE_GO || 'go', ['build', '-trimpath', '-buildvcs=false', '-ldflags=-s -w', '-o', output, source], {
      cwd: resolve(root, 'extension-kit'), stdio: 'inherit', windowsHide: true,
      env: {...process.env, CGO_ENABLED: '0', GOOS: 'wasip1', GOARCH: 'wasm'},
    });
    if (result.error) throw result.error;
    if (result.status !== 0) throw new Error(`Compilation échouée : ${source}`);
  });
}
