<script lang="ts">
 export let token:string;export let installed:()=>Promise<void>;
 let input:HTMLInputElement;let file:File|null=null;let busy=false;let error='';
 type Preview={package:{manifest:{name:string;description:string;minCore:string;module:{version:string}};checksum:string};previousVersion:string;previousChecksum:string;signer:string;services:{label:string;description:string}[]};
 let preview:Preview|null=null;
 async function upload(action:string){if(!file)throw Error('Choisissez un paquet.');const r=await fetch('/api/v1/provider-packages/'+action,{method:'POST',headers:{'Content-Type':'application/zip','X-Mosaune-Token':token,'X-Mosaune-Package-Checksum':preview?.package.checksum??'','X-Mosaune-Previous-Checksum':preview?.previousChecksum??''},body:file});const data=await r.json();if(!r.ok)throw Error(data.error||'Paquet non accepté.');return data;}
 async function inspect(){file=input.files?.[0]??null;preview=null;error='';if(!file)return;if(file.size>9*1024*1024){error='Le paquet doit faire moins de 9 Mio.';file=null;return}busy=true;try{preview=await upload('preview')}catch(e){error=e instanceof Error?e.message:'Vérification impossible.'}finally{busy=false}}
 async function install(){if(busy||!preview)return;busy=true;error='';try{await upload('install');preview=null;file=null;input.value='';await installed()}catch(e){error=e instanceof Error?e.message:'Installation impossible.'}finally{busy=false}}
</script>
<section class="extension-import provider-package-import" aria-busy={busy}>
 <span class="eyebrow">LES FOURNISSEURS</span><h3>Installer un fournisseur</h3>
 <p>Le premier paquet TMDB s’installe sans remplacer Mosaune. Une mise à jour est testée avant de remplacer la version actuelle.</p>
 <div class="extension-actions"><a class="text-action" href="/api/v1/provider-packages/example?catalogue=1" download>Télécharger TMDB 0.5.0</a><a class="text-action" href="/api/v1/provider-packages/example" download>Télécharger TMDB 0.4.0</a><a class="text-action" href="/api/v1/provider-packages/example?update=1" download>Télécharger la mise à jour 0.4.1</a></div>
 <p>La 0.5.0 prend aussi en charge la découverte, les profils et les affiches alternatives. Les anciennes versions restent disponibles pour tester la restauration.</p>
 <label class="extension-file">Paquet fournisseur à vérifier<input bind:this={input} type="file" accept=".zip,application/zip" disabled={busy} onchange={inspect}/></label>
 <small>Ce pilote accepte les paquets TMDB signés par la clé de développement Mosaune. Les autres fournisseurs et clés communautaires ne sont pas encore pris en charge.</small>
 {#if preview}{@const m=preview.package.manifest}<div class="extension-preview"><span class="extension-tag">Signature et compatibilité vérifiées</span><h3>{m.name} · {preview.previousVersion?preview.previousVersion+' → ':''}{m.module.version}</h3><p>{m.description}</p><dl><dt>Signature reconnue</dt><dd>{preview.signer}</dd><dt>Compatibilité</dt><dd>Mosaune {m.minCore} ou plus</dd><dt>Dépendance</dt><dd>Adaptateur TMDB de Mosaune · films et séries</dd><dt>Empreinte</dt><dd class="extension-hash">{preview.package.checksum}</dd></dl>
 {#each preview.services as service}<strong>{service.label}</strong><p>{service.description}</p>{/each}
 <p>{preview.previousVersion?'Si le nouveau module ne démarre pas, la version actuelle reste en place. Après réussite, elle reste disponible pour restauration. Une mise à jour conserve l’état actif ou désactivé du module.':'Le paquet sera installé désactivé. Vous pourrez ensuite l’activer depuis sa carte TMDB.'}</p>
 <div class="extension-actions"><button class="primary" disabled={busy} onclick={install}>{busy?'Vérification du module…':preview.previousVersion?'Installer cette mise à jour':'Installer ce fournisseur'}</button><button class="text-action" disabled={busy} onclick={()=>{preview=null;file=null;input.value=''}}>Annuler</button></div></div>{/if}
 {#if error}<p class="form-error" role="alert">{error}</p>{/if}
</section>
