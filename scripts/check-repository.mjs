// Repository hygiene only: never opens local catalogues or prints matched secrets.
import {execFileSync} from 'node:child_process';
import {existsSync,readFileSync,statSync} from 'node:fs';
import {resolve,dirname,relative,sep} from 'node:path';
import {fileURLToPath} from 'node:url';
const root=resolve(dirname(fileURLToPath(import.meta.url)),'..');
const paths=[...new Set(execFileSync('git',['ls-files','-z','--cached','--others','--exclude-standard'],{cwd:root,encoding:'utf8'}).split('\0').filter(Boolean))].filter(p=>existsSync(resolve(root,p))&&statSync(resolve(root,p)).isFile());
const rootFiles=new Set(['README.md','AGENTS.md','Dockerfile','go.mod','go.sum','package.json','package-lock.json','tsconfig.json','vite.config.ts','.gitignore','.gitattributes','.dockerignore']);
const roots=new Set(['.github','server','web','resources','extension-kit','scripts','deploy','docs','licenses']);
const errors=[];let links=0;
for(const file of paths){
 if(!file.includes('/')&&!rootFiles.has(file))errors.push(`${file}: fichier racine non prévu`);
 if(file.includes('/')&&!roots.has(file.split('/')[0]))errors.push(`${file}: dossier racine non prévu`);
 if(/(^|\/)(data|media|node_modules|build)\/|\.(db(?:-.*)?|private|key|pem|log)$|(^|\/)\.env(?:\..*)?$/.test(file)&&!file.endsWith('/.env.example'))errors.push(`${file}: donnée privée ou sortie à ne pas suivre`);
 const bytes=readFileSync(resolve(root,file));
 if(bytes.includes(0))continue;
 const text=bytes.toString('utf8');
 if(/-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----/.test(text)||/(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,}|AKIA[A-Z0-9]{16}|sk-[A-Za-z0-9]{32,})/.test(text))errors.push(`${file}: motif de secret détecté (valeur masquée)`);
 if(!file.endsWith('.md'))continue;
 for(const m of text.matchAll(/!?\[[^\]]*\]\(([^)]+)\)/g)){
  const link=m[1];if(/^(?:[a-z][a-z0-9+.-]*:|#)/i.test(link))continue;
  const raw=link.split('#')[0].split(' "')[0].replace(/^<|>$/g,'');if(!raw)continue;
  let target;try{target=resolve(dirname(resolve(root,file)),decodeURIComponent(raw))}catch{errors.push(`${file}: lien invalide`);continue}
  if(target!==root&&!target.startsWith(root+sep)){errors.push(`${file}: lien hors dépôt`);continue}
  links++;if(!existsSync(target))errors.push(`${file}: lien local absent ${relative(root,target).split(sep).join('/')}`);
 }
}
if(errors.length){console.error(errors.join('\n'));process.exitCode=1}else console.log(`Repository: ${paths.length} files, ${links} local links, layout and publication guards passed.`);
