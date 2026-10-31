<script lang="ts">
 import {onMount} from 'svelte';
 import {accountRequest} from './accounts';
 import RecognitionPackages from './RecognitionPackages.svelte';
 import {extensionsChanged} from './extensions';
 export let token:string;
 export let id:string;
 export let changed:()=>void;
 export let select:(id:string)=>void;
 type Step={slot:string;id:string;name:string;version:string;operation:string;shared:boolean};
 type Plan={manifest:{version:string;components:{slot:string;shared:boolean}[];screen:string};installedVersion:string;screenId:string;screenState:string;state:string;id:string;name:string;operation:string;revision:number;steps:Step[];kept:Step[];blockers:string[];checksum:string};
 let current:Plan|null=null,preview:Plan|null=null;
 let busy=false,error='',notice='';
 const labels:Record<string,string>={install:'Installer',enable:'Activer',select:'Choisir',disable:'Désactiver',remove:'Désinstaller',shared:'Partagé · conservé',ready:'Déjà prêt',unchanged:'Inchangé'};
 async function load(){const result=await accountRequest(token,'media-bundles');current=result.bundles.find((p:Plan)=>p.id===id)??null;}
 onMount(()=>{void load().catch(e=>error=e.message)});
 async function inspect(operation:string){if(busy||!current)return;busy=true;error='';notice='';preview=null;try{await load();preview=await accountRequest(token,'media-bundles/'+id+'/preview','POST',{operation,revision:current.revision});}catch(e){error=e instanceof Error?e.message:'Aperçu indisponible.'}finally{busy=false}}
 async function apply(){if(busy||!preview)return;busy=true;error='';try{await accountRequest(token,'media-bundles/'+id+'/apply','POST',{operation:preview.operation,revision:preview.revision,checksum:preview.checksum});preview=null;notice='Changements appliqués ensemble. Vos fichiers, fiches et progressions sont conservés.';await load();window.dispatchEvent(new Event('mosaune:detail-changed'));window.dispatchEvent(new Event('mosaune:catalogue-changed'));extensionsChanged();changed();}catch(e){error=e instanceof Error?e.message:'Aucune modification appliquée.';preview=null;await load().catch(()=>{});}finally{busy=false}}
</script>
<section class="extension-detail" aria-label="Composants de cette famille" aria-busy={busy}>
 <div class="extension-card-body">
 <h3>Une famille, des composants à votre choix</h3>
 <p>Écran, reconnaissance, catalogue et présentation sont gérés ensemble. Vous gardez vos composants de remplacement et pouvez les régler séparément ci-dessous.</p>
 <p>Les composants partagés restent disponibles pour les autres médias. Le lecteur et certains services sont encore intégrés ; ce regroupement couvre les composants déjà extraits.</p>
 {#if error}<p class="form-error" role="alert">{error}</p>{/if}
 {#if notice}<p class="extension-notice" role="status">{notice}</p>{/if}
 {#if current}
 <p class="folder-help">Regroupement {current.manifest.version}{current.installedVersion?" · installé":" · à installer"}</p>
 <p class="folder-help">{current.blockers.length?'Un choix de composant doit être corrigé.':current.steps.length?'Certains composants sont absents, désactivés ou non sélectionnés.':'Tous les composants de cet ensemble sont prêts.'}</p>
 <div class="extension-actions">
 <button class="secondary" disabled={busy} onclick={()=>inspect('prepare')}>Installer / activer la famille</button>
 <button class="text-action" disabled={busy} onclick={()=>inspect('disable')}>Désactiver la famille</button>
 <button class="text-action" disabled={busy} onclick={()=>inspect('remove')}>Retirer la famille</button>
 </div>
 {:else}<p role="status">Chargement des composants…</p>{/if}
 {#if preview}
 <div class="extension-preview" aria-live="polite">
 <h4>Changements prévus · {preview.name}</h4>
 {#each preview.blockers as blocker}<p class="form-error">{blocker}</p>{/each}
 {#if preview.steps.length}<ul>{#each preview.steps as step}<li><strong>{labels[step.operation]}</strong> · {step.name} · {step.version}{#if step.shared} · partagé avec d’autres médias{/if}</li>{/each}</ul>{:else}<p>Aucun changement nécessaire.</p>{/if}
 {#if preview.kept.length}<details open><summary>Composants conservés</summary><ul>{#each preview.kept as step}<li>{step.name} · {labels[step.operation]}</li>{/each}</ul></details>{/if}
 {#if preview.operation==='remove'}<p>Seuls les programmes choisis indiqués ci-dessus sont désinstallés, y compris un remplaçant externe s’il est choisi. Les alternatives non sélectionnées restent installées. Réimportez un remplaçant externe pour le récupérer ensuite.</p>{/if}
 <p>Tout est appliqué ensemble ou rien ne change. Une modification effectuée ailleurs invalidera cet aperçu.</p>
 <div class="extension-actions"><button class="primary" disabled={busy||preview.blockers.length>0} onclick={apply}>Confirmer les changements</button><button class="text-action" disabled={busy} onclick={()=>preview=null}>Annuler</button></div>
 </div>
 {/if}
 {#if current}<details class="extension-import"><summary>Composants et remplacements · réglages avancés</summary>
 <p>Ces composants sont inclus dans la famille. Les éléments partagés servent aussi aux autres médias. Leurs réglages restent accessibles sans les installer un par un.</p>
 <button class="secondary" disabled={current.screenState==='absent'} onclick={()=>current&&select(current.screenId)}>Configurer l’écran</button>
 <div class="extension-grid"><RecognitionPackages {token} slots={current.manifest.components.map(c=>c.slot)} refreshKey={current.revision} select={id=>select('recognition.'+id)}/></div>
 <RecognitionPackages {token} slots={current.manifest.components.map(c=>c.slot)} refreshKey={current.revision} tab="available" select={id=>select('recognition.'+id)}/>
 </details>{/if}
 </div>
</section>
