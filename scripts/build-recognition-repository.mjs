// Build a standalone, public-repository example without shipping it in the server.
// Every catalogue field comes from the manifest used to build that archive.
import {readFileSync,writeFileSync,mkdirSync,copyFileSync} from 'node:fs';
import {join} from 'node:path';
import {createHash} from 'node:crypto';
import {spawnSync} from 'node:child_process';

export function buildRecognitionRepository(root,go,env){
 const kit=join(root,'extension-kit'),out=join(root,'build/recognition-repository');
 mkdirSync(join(out,'packages'),{recursive:true});
 const source=JSON.parse(readFileSync(join(kit,'examples/recognition/extension.json'),'utf8'));
 const entries=[];
 for(const release of ['1.0.0','1.1.0']){
  const manifest={...source,module:{...source.module,version:release}};
  const manifestFile=join(root,'build/recognition-manifest-'+release+'.json');
  writeFileSync(manifestFile,JSON.stringify(manifest,null,2)+'\n');
  const archivePath='packages/film-names-'+release+'.mosaune.zip';
  const archive=join(out,archivePath);
  const r=spawnSync(go,['run','./cmd/package-module',manifestFile,'examples/recognition/module.wasm',archive],{cwd:kit,env,stdio:'inherit',windowsHide:true});
  if(r.error)throw r.error;if(r.status!==0)throw Error('Recognition repository packaging failed.');
  entries.push({id:manifest.module.id,name:manifest.name,publisher:manifest.publisher,description:manifest.description,version:release,kind:'recognition',minCore:manifest.minCore,path:archivePath,sha256:createHash('sha256').update(readFileSync(archive)).digest('hex')});
 }
 writeFileSync(join(out,'mosaune-repository.json'),JSON.stringify({schema:1,name:'Mosaune — exemple de reconnaissance',packages:entries},null,2)+'\n');
 copyFileSync(join(kit,'examples/recognition/REPOSITORY.md'),join(out,'README.md'));
}
