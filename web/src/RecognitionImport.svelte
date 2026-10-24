<script lang="ts">
 export let token:string;
 export let installed:()=>void;
 let file:File|undefined;
 let bytes:ArrayBuffer|undefined;
 let preview:any=null;
 let busy=false;
 let error='';
 let notice='';
 const kinds:Record<string,string>={films:'Films',series:'Séries',music:'Musique'};
 async function upload(path:string,headers:Record<string,string>={}) {
  const response=await fetch('/api/v1/recognition-packages/'+path,{method:'POST',headers:{'Content-Type':'application/zip','X-Mosaune-Token':token,...headers},body:bytes});
  const data=await response.json();if(!response.ok)throw Error(data.error||'Import indisponible.');return data;
 }
 async function inspect(event:Event) {
  file=(event.target as HTMLInputElement).files?.[0];preview=null;bytes=undefined;error='';notice='';if(!file)return;
  busy=true;
  try{if(file.size>9*1024*1024)throw Error('Le fichier dépasse 9 Mio.');bytes=await file.arrayBuffer();preview=await upload('preview');}
  catch(e){error=e instanceof Error?e.message:'Fichier illisible.';}finally{busy=false;}
 }
 async function install() {
  if(!preview||busy)return;busy=true;error='';
  try{await upload('import',{'X-Mosaune-Package-Checksum':preview.package.checksum,'X-Mosaune-Registry-Revision':String(preview.revision)});notice='Paquet installé. Retrouvez sa carte sur le serveur pour l’activer et le choisir pour vos scans.';preview=null;bytes=undefined;installed();}
  catch(e){error=e instanceof Error?e.message:'Installation indisponible.';preview=null;bytes=undefined;}finally{busy=false;}
 }
</script>
<section class="extension-import">
 <h3>Importer un module de reconnaissance</h3>
 <p>Un créateur peut fournir son propre module pour reconnaître les films, les séries ou la musique à partir des noms de fichiers.</p>
 <label class="extension-file">Paquet à vérifier<input type="file" accept=".zip,application/zip" disabled={busy} onchange={inspect}/></label>
 <p>Import local : l’identité de l’éditeur n’est pas vérifiée. Le module ne reçoit ni vos fichiers, ni vos clés, ni un accès au réseau ; il reçoit les noms relatifs pendant les scans.</p>
 {#if error}<p class="form-error" role="alert">{error} Sélectionnez de nouveau le fichier pour refaire la vérification.</p>{/if}
 {#if notice}<p class="extension-notice" role="status">{notice}</p>{/if}
 {#if busy}<p role="status">Vérification en cours…</p>{/if}
 {#if preview}
  <article class="extension-detail"><div class="extension-card-body">
   <h4>{preview.package.manifest.name} · {preview.package.manifest.module.version}</h4>
   <p>{preview.package.manifest.description}</p>
   <p>Éditeur déclaré : {preview.package.manifest.publisher} · Source locale non vérifiée</p>
   <p>Reconnaissance : {kinds[preview.package.manifest.contributions[0].target]} · Mosaune {preview.package.manifest.minCore} minimum</p>
   <p>Une nouvelle installation est désactivée. Une mise à jour conserve votre activation et votre choix ; la version précédente reste restaurable.</p>
   <button class="primary" disabled={busy} onclick={install}>Confirmer l’installation de ce fichier</button>
  </div></article>
 {/if}
</section>
