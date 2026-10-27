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
  const input=event.target as HTMLInputElement;file=input.files?.[0];input.value='';preview=null;bytes=undefined;error='';notice='';if(!file)return;
  busy=true;
  try{if(file.size>9*1024*1024)throw Error('Le fichier dépasse 9 Mio.');bytes=await file.arrayBuffer();preview=await upload('preview');}
  catch(e){error=e instanceof Error?e.message:'Fichier illisible.';}finally{busy=false;}
 }
 async function install() {
  if(!preview||busy)return;busy=true;error='';
  try{await upload('import',{'X-Mosaune-Package-Checksum':preview.package.checksum,'X-Mosaune-Registry-Revision':String(preview.revision)});notice='Paquet installé. Retrouvez sa carte sur le serveur pour l’activer et le choisir depuis sa fiche.';if(preview.package.manifest.contributions[0].role==='media.catalogue.v1')window.dispatchEvent(new Event('mosaune:catalogue-changed'));preview=null;bytes=undefined;installed();}
  catch(e){error=e instanceof Error?e.message:'Installation indisponible.';preview=null;bytes=undefined;}finally{busy=false;}
 }
</script>
<section class="extension-import">
 <h3>Importer un module d’analyse</h3>
 <p>Installez une présentation de fiche, un module de catalogue, un reconnaisseur de noms ou un lecteur des informations intégrées aux fichiers musicaux.</p>
 <label class="extension-file">Paquet à vérifier<input type="file" accept=".zip,application/zip" disabled={busy} onchange={inspect}/></label>
 <p>Import local : l’identité de l’éditeur n’est pas vérifiée. Les accès nécessaires sont affichés après vérification, avant votre confirmation.</p>
 {#if error}<p class="form-error" role="alert">{error} Sélectionnez de nouveau le fichier pour refaire la vérification.</p>{/if}
 {#if notice}<p class="extension-notice" role="status">{notice}</p>{/if}
 {#if busy}<p role="status">Vérification en cours…</p>{/if}
 {#if preview}
  <article class="extension-detail"><div class="extension-card-body">
   <h4>{preview.package.manifest.name} · {preview.package.manifest.module.version}</h4>
   <p>{preview.package.manifest.description}</p>
   <p>Éditeur déclaré : {preview.package.manifest.publisher} · Source locale non vérifiée</p>
   {#if preview.package.manifest.contributions[0].role==='media.detail-sections.v1'}<p>Rubriques de fiche · Textes, informations et références aux portraits/vidéos déjà connus. Aucun accès aux fichiers, chemins, réseau ou clés. Aucun code HTML ou JavaScript dans l’interface.</p>{:else if preview.package.manifest.contributions[0].role==='media.detail-list.v1'}<p>Présentation de fiche · Libellés et groupes à partir des informations autorisées uniquement, sans fichiers, chemins, réseau ni clés. Aucun code HTML ou JavaScript dans l’interface.</p>{:else if preview.package.manifest.contributions[0].role==='media.catalogue.v1'}
   <p>Organisation du catalogue · Métadonnées visibles uniquement, sans chemins, fichiers, réseau ou clés. Ce module choisit les regroupements et leur ordre d’affichage.</p>
   {:else if preview.package.manifest.contributions[0].role==='media.audio-tags.v1'}
   <p>Informations musicales · Accès demandé : lecture de blocs limités du seul fichier analysé. Aucun accès aux autres fichiers, au réseau ou aux clés.</p>
   {:else}<p>Accès demandé : noms relatifs uniquement, aucun accès au contenu des fichiers, au réseau ou aux clés.</p>
   <p>Reconnaissance : {kinds[preview.package.manifest.contributions[0].target]} · Mosaune {preview.package.manifest.minCore} minimum</p>{/if}
   <p>Une nouvelle installation est désactivée. Une mise à jour conserve votre activation et votre choix ; la version précédente reste restaurable.</p>
   <button class="primary" disabled={busy} onclick={install}>Confirmer l’installation de ce fichier</button>
  </div></article>
 {/if}
</section>
