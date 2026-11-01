<script lang="ts">
 import FamilyArchivePlan from './FamilyArchivePlan.svelte';
 import type {FamilyPlan as Plan} from './family-archive';
 import {extensionsChanged} from './extensions';
 export let token:string;
 export let installed:(kind:string)=>Promise<void>;
 let input:HTMLInputElement,file:File|null=null,plan:Plan|null=null,busy=false,error='';
 async function send(action:string){if(!file)throw Error('Choisissez une archive.');const r=await fetch('/api/v1/media-bundle-archives/'+action,{method:'POST',headers:{'Content-Type':'application/zip','X-Mosaune-Token':token,'X-Mosaune-Plan-Checksum':plan?.checksum??''},body:file});const out=await r.json();if(!r.ok)throw Error(out.error||'Archive refusée.');return out as Plan;}
 async function inspect(){plan=null;error='';file=input.files?.[0]??null;if(!file)return;if(file.size>32*1024*1024){error='L’archive doit faire moins de 32 Mio.';file=null;return}busy=true;try{plan=await send('preview')}catch(e){error=e instanceof Error?e.message:'Vérification impossible.'}finally{busy=false}}
 async function apply(){if(!plan||busy)return;busy=true;error='';try{const result=await send('import');plan=null;file=null;input.value='';window.dispatchEvent(new Event('mosaune:catalogue-changed'));window.dispatchEvent(new Event('mosaune:detail-changed'));extensionsChanged();await installed(result.manifest.kind)}catch(e){error=e instanceof Error?e.message:'Installation non appliquée.';plan=null}finally{busy=false}}
</script>
<section class="extension-import" aria-busy={busy}>
 <h3>Installer une famille depuis un fichier</h3>
 <p>Un seul paquet pour ses programmes et son écran. Mosaune conserve les remplaçants choisis et réutilise les composants partagés déjà présents.</p>
 <label>Archive de famille (.zip)<input bind:this={input} type="file" accept=".zip,application/zip" disabled={busy} onchange={inspect}/></label>
 {#if error}<p class="form-error" role="alert">{error}</p>{/if}
 {#if plan}<div class="extension-preview">
 <h4>{plan.manifest.name}</h4><p>{plan.previousVersion?plan.previousVersion+' → ':''}{plan.manifest.version} · {plan.manifest.publisher} · éditeur déclaré, non vérifié</p>
 <FamilyArchivePlan {plan}/>
 <div class="extension-actions"><button class="primary" disabled={busy} onclick={apply}>Installer ce paquet de famille</button><button class="text-action" disabled={busy} onclick={()=>{plan=null;file=null;input.value=''}}>Annuler</button></div>
 </div>{/if}
 <details><summary>Exemples pour les contributeurs</summary><p>Ces exemples réutilisent les fonctions actuelles avec des identifiants distincts. Ils servent à tester l’installation et une mise à jour ; ils n’ajoutent pas de lecteur.</p>
 {#each [{id:'films',name:'Films'},{id:'series',name:'Séries'},{id:'music',name:'Musique'}] as family}<p>{family.name} : <a href={'/api/v1/media-bundle-archives/example/'+family.id} download>Version 3.0</a> · <a href={'/api/v1/media-bundle-archives/example/'+family.id+'?version=3.1.0'} download>Version 3.1</a></p>{/each}
 </details>
</section>
