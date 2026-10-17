import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
export function buildTheme(){
 const theme=JSON.parse(fs.readFileSync(path.join(root,'theme/default.json'),'utf8'));
 const css=['light','dark'].map(mode=>{
  const declarations=Object.keys(theme.tokens).sort().map(key=>{
   let value=theme.tokens[key];
   if(value.startsWith('$')){const [kind,name]=value.slice(1).split('.');if(kind==='palette')value=theme.palette[name][mode];else if(kind==='fonts')value=theme.fonts[name].map(f=>['serif','sans-serif','monospace','system-ui','cursive','fantasy'].includes(f)?f:JSON.stringify(f)).join(',');else if(kind==='radii')value=theme.radii[name];else throw Error('Unknown theme reference '+value)}
   return '--m-'+key.replaceAll('.','-')+':'+value+';';
  }).join('');return (mode==='light'?':root':':root[data-theme="dark"]')+'{'+declarations+'}\n';
 }).join('');
 const target=path.join(root,'web/src/theme-defaults.css');if(!fs.existsSync(target)||fs.readFileSync(target,'utf8')!==css)fs.writeFileSync(target,css);
}
buildTheme();
