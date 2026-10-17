import { spawnSync } from 'node:child_process';
import { readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

// Accompagner les exécutables des textes de licence de leurs dépendances.
// Les chemins de la machine de construction ne sont jamais inclus dans le résultat.
export function buildNotices(root, go, env) {
  const capture = (args, directory = root) => {
    const result = spawnSync(go, args, { cwd: directory, env, encoding: 'utf8', windowsHide: true });
    if (result.status !== 0) throw new Error(result.stderr || 'Impossible de lire les dépendances Go.');
    return result.stdout.trim();
  };
  const sections = [
    'Mosaune — notices des composants tiers',
    'La licence propre à Mosaune reste à définir. Les textes ci-dessous concernent les composants tiers, pas une licence générale du projet.',
  ];
  function append(name, directory) {
    const files = readdirSync(directory, { withFileTypes: true }).filter(file => file.isFile() && /^(LICENSE|LICENCE|COPYING|NOTICE)([._-]|$)/i.test(file.name)).map(file => file.name).sort();
    if (!files.length) throw new Error(`Texte de licence introuvable pour ${name}.`);
    for (const file of files) sections.push(`${name} — ${file}\n\n${readFileSync(join(directory, file), 'utf8').trim()}`);
  }
  append(`Go ${capture(['env', 'GOVERSION'])}`, capture(['env', 'GOROOT']));
  const seen = new Set();
  for (const moduleDirectory of [root, join(root, 'extension-kit')]) {
    for (const row of capture(['list', '-m', '-f', '{{.Path}}|{{.Version}}|{{.Dir}}', 'all'], moduleDirectory).split(/\r?\n/)) {
      const [name, version, directory] = row.split('|');
      const identity = `${name} ${version}`;
      if (name === 'mosaune' || name === 'mosaune/extensionkit' || !directory || seen.has(identity)) continue;
      append(identity, directory);
      seen.add(identity);
    }
  }
  for (const name of ['svelte', 'esm-env']) {
    const directory = join(root, 'node_modules', name);
    const { version } = JSON.parse(readFileSync(join(directory, 'package.json'), 'utf8'));
    append(`${name} ${version}`, directory);
  }
  writeFileSync(join(root, 'build', 'THIRD-PARTY-NOTICES.txt'), sections.join('\n\n============================================================\n\n') + '\n');
}
