<script lang="ts">
 import {extensionsChanged} from './extensions';
 export let token:string;
 export let installed:(kind:string)=>Promise<void>;
 type Plan={manifest:{name:string;publisher:string;version:string;kind:string};previousVersion:string;checksum:string;steps:{slot:string;id:string;name:string;version:string;operation:string;select:boolean;previousId:string;shared:boolean}[]};
 let input:HTMLInputElement,file:File|null=null,plan:Plan|null=null,busy=false,error='';
 const labels:Record<string,string>={install:'Installer · en pause',update:'Mettre à jour',reuse:'Réutiliser',keep:'Conserver le composant choisi'};
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
 <ul>{#each plan.steps as step}<li><strong>{labels[step.operation]}</strong> · {step.operation==='keep'?step.previousId:step.name}{step.operation==='keep'?'':' · '+step.version}{step.shared?' · partagé':''}{step.select?' · devient le composant choisi':''}</li>{/each}</ul>
 <p>Les nouveaux composants restent en pause. Une mise à jour conserve les états existants. Un nouvel écran remplace le choix de l’écran fourni et met celui-ci en pause ; ses accès utilisateurs seront à définir dans les réglages avancés.</p>
 <p>Programmes isolés : pas d’accès réseau. Les composants musicaux peuvent lire les données du fichier analysé. Aucun accès général aux fichiers ni aux comptes.</p>
 <p>Tout est installé ensemble ou rien ne change. Activez ensuite la famille depuis sa fiche.</p>
 <div class="extension-actions"><button class="primary" disabled={busy} onclick={apply}>Installer ce paquet de famille</button><button class="text-action" disabled={busy} onclick={()=>{plan=null;file=null;input.value=''}}>Annuler</button></div>
 </div>{/if}
 <details><summary>Exemples pour les contributeurs</summary><p>Ces exemples réutilisent les fonctions actuelles avec des identifiants distincts. Ils servent à tester l’installation et une mise à jour ; ils n’ajoutent pas de lecteur.</p>
 {#each [{id:'films',name:'Films'},{id:'series',name:'Séries'},{id:'music',name:'Musique'}] as family}<p>{family.name} : <a href={'/api/v1/media-bundle-archives/example/'+family.id} download>Version 1.0</a> · <a href={'/api/v1/media-bundle-archives/example/'+family.id+'?version=1.1.0'} download>Version 1.1</a></p>{/each}
 </details>
</section>
