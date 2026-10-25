<script lang="ts">
 import './repository-store.css';
 import {onMount,tick} from 'svelte';
 import {accountRequest} from './accounts';
 import {extensionsChanged} from './extensions';
 import {catalogCandidates,compareVersions as compare} from './repository-catalog';
 import type {Entry,Repository,Choice} from './repository-catalog';
 export let token:string;
 export let safeMode=false;
 export let installed:(kind:string,id:string)=>Promise<void>;
 type Installed={id:string;version:string;kind:string};
 type Preview={revision?:number;package:{checksum:string;manifest:{requires?:string[];contributions?:{role:string;target:string}[]}};previousVersion:string;previousChecksum:string;signer?:string;services?:{label:string;description:string}[]};
 let repos:Repository[]=[];let versions:Installed[]=[];let core='0.0.0';
 let busy=false;let loading=true;let error='';let notice='';let search='';let source='';let family='';let url='';let ref='';let removing='';
 $: official=repos.find(r=>r.official);
 let choice:Choice|null=null;let preview:Preview|null=null;let archive:Blob|null=null;let heading:HTMLElement;
 function current(e:Entry){return versions.find(v=>v.kind===e.kind&&v.id===e.id)?.version??''}
 function status(e:Entry){if(compare(e.minCore,core)>0)return 'Mosaune '+e.minCore+' requis';const v=current(e);return !v?'Disponible':compare(e.version,v)>0?'Mise à jour disponible':compare(e.version,v)===0?'Installé':'Version plus récente installée'}
 $: visible=catalogCandidates(repos,search,source,family);
 $: releases=choice?choice.repo.index.packages.filter(e=>e.id===choice?.entry.id&&e.kind===choice?.entry.kind).sort((a,b)=>compare(b.version,a.version)):[];
 $: canInspect=choice&&compare(choice.entry.minCore,core)<=0&&(!current(choice.entry)||compare(choice.entry.version,current(choice.entry))>0)&&!(safeMode&&choice.entry.kind!=='theme');
 async function load(){const data=await accountRequest(token,'extension-repositories');repos=data.repositories;versions=data.installed;core=data.coreVersion;if(source&&!repos.some(r=>r.id===source))source=''}
 onMount(()=>{void perform(load).finally(()=>loading=false)});
 async function perform(action:()=>Promise<void>){if(busy)return;busy=true;error='';notice='';try{await action()}catch(e){error=e instanceof Error?e.message:'Opération indisponible.'}finally{busy=false}}
 async function open(c:Choice|null){choice=c;archive=null;preview=null;error='';notice='';await tick();heading?.focus();heading?.closest('.settings-content')?.scrollTo({top:0})}
 async function add(){await perform(async()=>{await accountRequest(token,'extension-repositories','POST',{url,ref});url='';ref='';await load();notice='Dépôt ajouté. Aucun paquet n’a été installé automatiquement.'})}
 async function refreshRepo(repo:Repository){await perform(async()=>{await accountRequest(token,'extension-repositories/'+repo.id+'/refresh','POST',{});await load();notice='Catalogue actualisé. Les mises à jour restent à votre choix.'})}
 async function removeRepo(repo:Repository){await perform(async()=>{await accountRequest(token,'extension-repositories/'+repo.id,'DELETE',{});removing='';await load();notice='Dépôt retiré. Ses extensions installées restent disponibles.'})}
 async function packageRequest(action:string){if(!choice||!archive)throw Error('Vérifiez le paquet.');const endpoint=choice.entry.kind==='provider'?'provider-packages':(choice.entry.kind==='recognition'||choice.entry.kind==='audio-tags')?'recognition-packages':'extensions';if(action==='install'&&(choice.entry.kind==='recognition'||choice.entry.kind==='audio-tags'))action='import';const r=await fetch('/api/v1/'+endpoint+'/'+action,{method:'POST',headers:{'Content-Type':'application/zip','X-Mosaune-Token':token,'X-Mosaune-Package-Checksum':preview?.package.checksum??'','X-Mosaune-Previous-Checksum':preview?.previousChecksum??'','X-Mosaune-Registry-Revision':String(preview?.revision??'')},body:archive});const data=await r.json();if(!r.ok)throw Error(data.error||'Paquet non accepté.');return data}
 async function inspect(){await perform(async()=>{if(!choice)return;archive=null;preview=null;const e=choice.entry;const r=await fetch('/api/v1/extension-repositories/'+choice.repo.id+'/package',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:JSON.stringify({id:e.id,version:e.version,sha256:e.sha256})});if(!r.ok){const data=await r.json();throw Error(data.error||'Téléchargement impossible.')}archive=await r.blob();preview=await packageRequest('preview')})}
 async function install(){await perform(async()=>{if(!choice||!preview)return;const selected=choice;try{await packageRequest('install')}catch(e){archive=null;preview=null;await load();throw e}archive=null;preview=null;extensionsChanged();await installed(selected.entry.kind,selected.entry.id)})}
</script>

<section class="repository-store" aria-busy={busy}>
 {#if choice}<button class="back-link" disabled={busy} onclick={()=>open(null)}>← Retour au catalogue</button>{/if}
 <span class="eyebrow">À VOTRE CHOIX</span><h3 bind:this={heading} tabindex="-1">{choice?choice.entry.name:'Le catalogue des extensions'}</h3>
 {#if error}<p class="form-error" role="alert">{error}</p>{/if}
 {#if notice}<p class="extension-notice" role="status">{notice}</p>{/if}
 {#if choice}
  <article class="extension-preview">
   <span class="extension-tag">{choice.entry.kind==='theme'?'Interface · thème':choice.entry.kind==='audio-tags'?'Métadonnées · fichiers locaux':choice.entry.kind==='recognition'?'Bibliothèque · reconnaissance':'Métadonnées · fournisseur'}</span>
   <p>{choice.entry.description}</p>
   <p>Source : {choice.repo.index.name} · {choice.repo.bundled?'fourni avec cette version de Mosaune':choice.repo.url}</p>
   <label>Version du paquet<select disabled={busy} value={choice.entry.version} onchange={e=>{const selected=releases.find(p=>p.version===e.currentTarget.value);if(choice&&selected){choice={repo:choice.repo,entry:selected};preview=null;archive=null;error=''}}}>{#each releases as release}<option value={release.version}>{release.version}</option>{/each}</select></label>
   <dl><dt>Éditeur déclaré</dt><dd>{choice.entry.publisher}</dd><dt>Compatibilité annoncée</dt><dd>Mosaune {choice.entry.minCore} ou plus</dd><dt>Sur votre serveur</dt><dd>{current(choice.entry)||'Pas encore installé'}</dd></dl>
   {#if !preview}
    <p>{(choice.entry.kind==='recognition'||choice.entry.kind==='audio-tags')?'Module d’analyse : l’identité de l’éditeur n’est pas vérifiée. Le dépôt et son empreinte ne constituent pas une signature. Le programme sera exécuté uniquement après confirmation, dans le moteur limité de Mosaune.':choice.entry.kind==='provider'?'Le fichier doit porter la signature TMDB reconnue par Mosaune. Ajouter un dépôt ne lui accorde aucun accès.':'Thème déclaratif : aucun programme, accès aux fichiers, aux comptes ou au réseau. Le nom d’éditeur ne constitue pas une identité vérifiée.'}</p>
    <button class="secondary" disabled={busy||!canInspect} onclick={inspect}>{busy?'Téléchargement et vérification…':'Vérifier ce paquet'}</button>
    {#if !canInspect}<p>{safeMode&&choice.entry.kind!=='theme'?'Le mode de secours suspend les installations de programmes.':status(choice.entry)}</p>{/if}
   {:else}
    <span class="extension-tag">{choice.entry.kind==='provider'?'Signature et compatibilité vérifiées':'Format et compatibilité vérifiés'}</span>
    <dl><dt>Accès du paquet</dt><dd>{choice.entry.kind==='theme'?'Apparence uniquement. Aucun accès à vos fichiers ou comptes.':(choice.entry.kind==='recognition'||choice.entry.kind==='audio-tags')?(choice.entry.kind==='audio-tags'?'Blocs limités du seul fichier musical analysé. Aucun accès aux autres fichiers, au réseau ou aux clés. Éditeur non vérifié.':'Noms relatifs pendant les scans uniquement. Aucun accès aux fichiers, au réseau ou aux clés. Éditeur non vérifié.'):preview.signer}</dd><dt>Empreinte du fichier vérifié</dt><dd class="extension-hash">{choice.entry.sha256}</dd></dl>
    {#if (choice.entry.kind==='recognition'||choice.entry.kind==='audio-tags')}<p>{choice.entry.kind==='audio-tags'?'Informations musicales':'Reconnaissance'} : {({films:'Films',series:'Séries',music:'Musique'} as Record<string,string>)[preview.package.manifest.contributions?.[0]?.target??'']}</p>{/if}
    {#each preview.services??[] as service}<strong>{service.label}</strong><p>{service.description}</p>{/each}
    <p>{(choice.entry.kind==='recognition'||choice.entry.kind==='audio-tags')?'Une nouvelle installation reste désactivée. Activez le module puis choisissez-le pour vos scans. Une mise à jour conserve votre choix et votre activation ; la version précédente reste restaurable.':choice.entry.kind==='theme'?'Le thème sera installé désactivé. Vous pourrez l’activer et le choisir pour votre compte.':preview.previousVersion?'La version actuelle restera disponible pour restauration. Son état actif ou désactivé sera conservé.':'Le fournisseur sera installé désactivé. Vous pourrez l’activer depuis sa fiche TMDB.'}</p>
    <div class="extension-actions"><button class="primary" disabled={busy} onclick={install}>{busy?'Installation…':(preview.previousVersion||current(choice.entry))?'Installer cette mise à jour':'Installer ce paquet'}</button><button class="text-action" disabled={busy} onclick={()=>{preview=null;archive=null}}>Annuler</button></div>
   {/if}
  </article>
 {:else}
  <p>Choisissez un paquet, vérifiez ses accès, puis installez-le. Les exemples inclus fonctionnent sans dépôt GitHub.</p>
  {#if official}<article class="extension-preview" aria-label="Dépôt officiel">
   <span class="extension-tag">Source officielle</span><h4>{official.index.name}</h4>
   <p><a href={official.url} target="_blank" rel="noreferrer">car-lito/mosaune-extensions ↗</a></p>
   <p>{official.updatedAt?'Catalogue conservé sur votre serveur. Actualisez-le pour rechercher de nouvelles versions.':'Le dépôt est prêt. Chargez son catalogue pour voir les paquets publiés sur GitHub.'}</p>
   {#if official.updatedAt}<small>Dernière actualisation : {new Date(official.updatedAt*1000).toLocaleString()}</small>{/if}
   <div class="extension-actions"><button class="secondary" disabled={busy} onclick={()=>refreshRepo(official!)}>{official.updatedAt?'Actualiser le catalogue officiel':'Charger le catalogue officiel'}</button></div>
  </article>{/if}
  <div class="extension-toolbar">
   <label>Rechercher dans le catalogue<input type="search" bind:value={search} placeholder="Nom, éditeur, description…"/></label>
   <label>Source<select bind:value={source}><option value="">Toutes les sources</option>{#each repos as repo}<option value={repo.id}>{repo.index.name}{repo.bundled?'':' · '+repo.url.replace('https://github.com/','')}</option>{/each}</select></label>
   <label>Type de paquet<select bind:value={family}><option value="">Tous les types</option><option value="theme">Thèmes</option><option value="provider">Fournisseurs</option><option value="recognition">Reconnaissance des médias</option><option value="audio-tags">Informations musicales</option></select></label>
  </div>
  {#if loading}<p role="status">Chargement du catalogue…</p>{/if}
  <div class="extension-grid">{#each visible as c (c.repo.id+':'+c.entry.kind+':'+c.entry.id)}<article class="extension-card extension-summary"><button class="extension-card-open" disabled={busy} onclick={()=>open(c)} aria-label={'Voir '+c.entry.name+' depuis '+c.repo.index.name}><small>{c.entry.kind==='theme'?'Thème':c.entry.kind==='recognition'?'Reconnaissance':c.entry.kind==='audio-tags'?'Informations musicales':'Fournisseur'} · {c.entry.version}</small><h3>{c.entry.name}</h3><p>{c.entry.description}</p><small>{c.repo.index.name}</small><span class="extension-tag">{status(c.entry)}</span></button></article>{/each}</div>
  {#if !loading&&!visible.length}<p>Aucun paquet ne correspond à ces filtres.</p>{/if}
  <details class="extension-import"><summary>Gérer les dépôts GitHub</summary>
   <p>Un dépôt public doit contenir un fichier mosaune-repository.json. Sa liste est conservée ici ; l’actualisation et les installations restent manuelles.</p>
   <a class="text-action" href="/api/v1/extension-repositories/example" download>Télécharger un dépôt d’exemple (ZIP)</a>
   <form onsubmit={e=>{e.preventDefault();void add()}}>
    <label>Adresse du dépôt GitHub<input type="url" bind:value={url} placeholder="https://github.com/auteur/depot" required disabled={busy}/></label>
    <label>Branche ou référence (facultatif)<input bind:value={ref} placeholder="Vide : branche par défaut" maxlength="128" disabled={busy}/></label>
    <button type="submit" class="secondary" disabled={busy}>{busy?'Vérification…':'Ajouter le dépôt'}</button>
   </form>
   {#each repos.filter(r=>!r.bundled&&!r.official) as repo}<article class="extension-preview"><h4>{repo.index.name}</h4><p>{repo.url} · {repo.ref||'Branche par défaut'}</p><small>Dernière actualisation : {new Date(repo.updatedAt*1000).toLocaleString()}</small><div class="extension-actions"><button class="secondary" disabled={busy} onclick={()=>refreshRepo(repo)}>Actualiser ce dépôt</button><button class="text-action" disabled={busy} onclick={()=>removing=repo.id}>Retirer ce dépôt</button></div>{#if removing===repo.id}<p>Les extensions déjà installées seront conservées.</p><div class="extension-actions"><button class="secondary" disabled={busy} onclick={()=>removeRepo(repo)}>Confirmer le retrait du dépôt</button><button class="text-action" disabled={busy} onclick={()=>removing=''}>Annuler</button></div>{/if}</article>{/each}
   <p>La source officielle reste proposée. Aucun dépôt n’est contacté automatiquement. Les paquets « Inclus avec Mosaune » restent toujours accessibles.</p>
  </details>
 {/if}
</section>
