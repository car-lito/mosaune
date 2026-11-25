// Explicit local launch; never runs as part of build or CI.
import {readFileSync,existsSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import {dirname,resolve} from 'node:path';
import {spawn} from 'node:child_process';
const root=resolve(dirname(fileURLToPath(import.meta.url)),'..');
const flags=process.argv.slice(2);
if(flags.some(v=>v!=='--safe-mode'))throw Error('Usage: node scripts/run-server.mjs [--safe-mode]');
const {version}=JSON.parse(readFileSync(resolve(root,'package.json'),'utf8'));
const platform=process.platform==='win32'?'windows':process.platform;
const arch=process.arch==='x64'?'amd64':process.arch;
const binary=resolve(root,`build/mosaune-${version}-${platform}-${arch}${platform==='windows'?'.exe':''}`);
if(!existsSync(binary))throw Error('Construisez d’abord avec node scripts/project.mjs build.');
const child=spawn(binary,['-data-dir',resolve(root,'data'),...(flags.includes('--safe-mode')?['-safe-mode']:[])],{cwd:root,stdio:'inherit',windowsHide:true});
child.on('error',e=>{console.error(e.message);process.exitCode=1});
child.on('exit',code=>{process.exitCode=code??1});
