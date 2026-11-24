<script lang="ts">
 import {onMount,tick} from 'svelte';
 import ExtensionCard from './ExtensionCard.svelte';
 import RecognitionPackages from './RecognitionPackages.svelte';
 import MediaBundlePanel from './MediaBundlePanel.svelte';
 import MediaBundleImport from './MediaBundleImport.svelte';
 import RecognitionImport from './RecognitionImport.svelte';
 const recognitionNames:Record<string,string>={films:'Reconnaissance des films',series:'Reconnaissance des séries',music:'Reconnaissance musicale'};
 import ProviderDetails from './ProviderDetails.svelte';
 import {providerStatus,providerRoute,type ProviderManagement} from './provider-management';
 import ProviderPackageImport from './ProviderPackageImport.svelte';
 import ExtensionRepositoryStore from './ExtensionRepositoryStore.svelte';
 import {accountRequest,type Account} from './accounts';
 import {extensionsChanged,type ExtensionPackage,type ExtensionAppearance} from './extensions';
 export let user:Account;export let token:string;
 export let refresh:()=>Promise<void>;export let openSettings:(id:string)=>void;
 let tab:'installed'|'available'='installed';let selected='';let filter='';let heading:HTMLElement;
 let providers:ProviderManagement[]=[];
 async function providerInstalled(id:string){await load();tab='installed';await show('provider.'+id);notice='Paquet fournisseur installé. Vos fiches sont conservées.';}
 async function repositoryInstalled(kind:string,id:string){if(kind==='detail-list'||kind==='detail-sections'||kind==='detail-header'||kind==='detail-experience')window.dispatchEvent(new Event('mosaune:detail-changed'));if(kind==='catalogue')window.dispatchEvent(new Event('mosaune:catalogue-changed'));await load();tab='installed';await show(kind==='family'?'builtin.'+id:kind==='provider'?(providers.some(p=>p.id===id)?'provider.'+id:''):(kind==='acquisition'||kind==='recognition'||kind==='audio-tags'||((kind==='detail-list'||kind==='detail-sections'||kind==='detail-header'||kind==='detail-experience')||kind==='catalogue'))?'recognition.'+id:id);notice='Paquet installé. Retrouvez ses réglages ci-dessous.';}
 $: selectedPackage=packages.find(p=>p.manifest.id===selected);
 $: selectedNative=native.find(p=>p.id===selected);
 $: selectedProvider=providers.find(p=>'provider.'+p.id===selected);
 $: visiblePackages=packages.filter(p=>!groupedScreens.has(p.manifest.id)&&(p.manifest.name+' '+p.manifest.description).toLocaleLowerCase().includes(filter.toLocaleLowerCase()));
 async function show(id:string){if(!id)selectionStack=[];else if(selected&&selected!==id)selectionStack.push(selected);selected=id;editing='';removing='';await tick();heading?.focus();heading?.closest('.settings-content')?.scrollTo({top:0});}
 async function back(){const previous=selected;selected=selectionStack.pop()??'';editing='';removing='';await tick();document.getElementById('extension-card-'+previous)?.focus();}
 async function toggleProvider(p:ProviderManagement){await perform(async()=>{await accountRequest(token,providerRoute(p.id,'state'),'PUT',{enabled:!p.enabled});notice=p.enabled?'Fournisseur désactivé. Vos informations enregistrées restent disponibles.':'Fournisseur activé selon les autorisations et composants disponibles.';});}
 function matches(name:string,query:string){return name.toLocaleLowerCase().includes(query.toLocaleLowerCase())}

 let packages:ExtensionPackage[]=[];let users:Account[]=[];let appearance:ExtensionAppearance={selected:'',effective:'',stamp:'',safeMode:false};
 let recognitionRefresh=0;
 let bundles:{id:string;name:string;state:string;manifest:{version:string;screen:string};archive?:{ids:Record<string,string>;screenId:string}}[]=[];
 let selectionStack:string[]=[];let suppliedComponentIds:string[]=[];
 $: groupedScreens=new Set(bundles.flatMap(b=>[b.manifest.screen,...(b.archive?[b.archive.screenId]:[])]));
 $: bundledComponentIds=bundles.flatMap(b=>Object.values(b.archive?.ids??{}));
 const bundleStatus:Record<string,string>={ready:'Prêt',partial:'À compléter',disabled:'Désactivé',absent:'À installer'};
 let busy=false;let loading=true;let error='';let notice='';let file:File|null=null;let input:HTMLInputElement;
 let preview:{package:ExtensionPackage;previousChecksum:string;previousVersion:string}|null=null;
 let removing='';let editing='';let grants:string[]=[];
 const native=[
 {id:'builtin.films',name:'Films',family:'Médias',description:'Bibliothèques, fiches et lecture des films.',settings:'libraries'},
 {id:'builtin.series',name:'Séries',family:'Médias',description:'Saisons, épisodes et reprise de lecture.',settings:'libraries'},
 {id:'builtin.music',name:'Musique',family:'Médias',description:'Albums, playlists et file d’écoute.',settings:'libraries'},
 {id:'builtin.radarr',name:'Radarr',family:'Connecteurs',description:'Demandes et suivi de vos films.',settings:'downloads'},
 {id:'builtin.sonarr',name:'Sonarr',family:'Connecteurs',description:'Demandes et suivi de vos séries.',settings:'downloads'}];
 async function load(){error='';try{
 const [list,a]=await Promise.all([accountRequest(token,'extensions'),accountRequest(token,'extensions/appearance')]);packages=list.packages;appearance=a;
 if(user.admin){const result=await Promise.all([accountRequest(token,'accounts'),accountRequest(token,'metadata/provider-management'),accountRequest(token,'media-bundles'),accountRequest(token,'recognition-packages')]);users=result[0];providers=result[1];bundles=result[2].bundles;suppliedComponentIds=result[3].packages.filter((p:{factory:boolean})=>p.factory).map((p:{id:string})=>p.id);}
 if(selected.startsWith('provider.')&&!providers.some(p=>'provider.'+p.id===selected))selected='';
 if(selected&&!selected.startsWith('builtin.')&&!selected.startsWith('recognition.')&&!selected.startsWith('provider.')&&selected!=='audio-tags'&&!packages.some(p=>p.manifest.id===selected))selected='';
 }catch(e){error=e instanceof Error?e.message:'Extensions indisponibles.'}finally{loading=false;recognitionRefresh++}}
 onMount(()=>{void load();const focus=()=>{if(!busy)void load()};window.addEventListener('focus',focus);return()=>window.removeEventListener('focus',focus)});
 async function perform(action:()=>Promise<void>){if(busy)return;busy=true;error='';notice='';try{await action();await load();extensionsChanged()}catch(e){error=e instanceof Error?e.message:'Opération indisponible.'}finally{busy=false}}
 async function upload(path:string){if(!file)throw Error('Choisissez un paquet.');const r=await fetch('/api/v1/extensions/'+path,{method:'POST',headers:{'Content-Type':'application/zip','X-Mosaune-Token':token,'X-Mosaune-Package-Checksum':preview?.package.checksum??'','X-Mosaune-Previous-Checksum':preview?.previousChecksum??''},body:file});const result=await r.json();if(!r.ok)throw Error(result.error||'Paquet non accepté.');return result;}
 async function inspect(){file=input.files?.[0]??null;preview=null;notice='';error='';if(!file)return;if(file.size>262144){error='Le paquet doit faire moins de 256 Ko.';file=null;return}await perform(async()=>{preview=await upload('preview')})}
 async function install(){await perform(async()=>{await upload('install');notice='Paquet installé, désactivé par défaut. Activez-le pour rendre ses fonctions disponibles. Les thèmes se choisissent ensuite pour votre compte.';preview=null;file=null;if(input)input.value='';tab='installed';selected='';})}
 async function toggle(p:ExtensionPackage){await perform(async()=>{await accountRequest(token,'extensions/'+p.manifest.id+'/state','PUT',{enabled:!p.enabled,checksum:p.checksum});notice=p.enabled?'Extension désactivée. Vos choix restent conservés.':'Extension activée. Ses fonctions sont disponibles pour les comptes autorisés.';})}
 async function choose(id:string){await perform(async()=>{await accountRequest(token,'account/extension-theme','PUT',{id});notice=id?'Thème choisi. Le style s’applique à toutes les rubriques.':'Apparence Mosaune rétablie pour votre compte.';})}
 async function remove(p:ExtensionPackage){await perform(async()=>{await accountRequest(token,'extensions/'+p.manifest.id,'DELETE',{checksum:p.checksum});removing='';editing='';selected='';notice='Paquet retiré. Les comptes concernés retrouvent l’apparence Mosaune ; votre collection est conservée.';})}
 function editAccess(p:ExtensionPackage){editing=p.manifest.id;grants=[...p.users];removing=''}
 function changeAccess(id:string,checked:boolean){grants=checked?[...grants,id]:grants.filter(x=>x!==id)}
 async function saveAccess(p:ExtensionPackage){await perform(async()=>{await accountRequest(token,'extensions/'+p.manifest.id+'/access','PUT',{users:grants,checksum:p.checksum});editing='';notice='Accès enregistrés. Les administrateurs gardent accès à toutes les extensions.';})}
</script>
<section class="extension-settings" aria-busy={busy}>
 {#if selected}<button class="back-link" onclick={back}>← Retour aux extensions</button>{/if}
 <span class="eyebrow">VOTRE MOSAUNE</span>
 <h2 bind:this={heading} tabindex="-1">{selectedPackage?.manifest.name ?? selectedProvider?.name ?? selectedNative?.name ?? recognitionNames[selected.replace('recognition.','')] ?? (selected==='audio-tags'?'Informations musicales':'Les extensions')}</h2>
 {#if !selected}<p>Vos médias, vos services et votre apparence. Ouvrez une carte pour retrouver tous ses réglages.</p>{/if}
 {#if appearance.safeMode}<div class="extension-warning" role="status"><strong>Mode de secours</strong><p>Les thèmes externes et les modules sont suspendus. Les choix d’activation des fournisseurs restent conservés.</p></div>{/if}
 {#if error}<p class="form-error" role="alert">{error}</p>{/if}{#if notice}<p class="extension-notice" role="status">{notice}</p>{/if}
 {#if selectedPackage}
  {@const p=selectedPackage}{@const m=p.manifest}
  <article class="extension-detail"> {#if m.api!==3}<svg class="extension-palette" viewBox="0 0 360 88" role="img" aria-label={'Palettes claire et sombre de '+m.name}><rect width="180" height="88" fill={p.preview.light.canvas}/><rect x="180" width="180" height="88" fill={p.preview.dark.canvas}/><circle cx="42" cy="44" r="19" fill={p.preview.light.accent}/><rect x="77" y="28" width="74" height="8" rx="4" fill={p.preview.light.ink}/><rect x="77" y="48" width="52" height="5" rx="2" fill={p.preview.light.muted}/><circle cx="222" cy="44" r="19" fill={p.preview.dark.accent}/><rect x="257" y="28" width="74" height="8" rx="4" fill={p.preview.dark.ink}/><rect x="257" y="48" width="52" height="5" rx="2" fill={p.preview.dark.muted}/></svg>{/if}
 <div class="extension-card-body"><div class="extension-card-heading"><h3>{m.name}</h3><span class="extension-tag">{appearance.safeMode?'Suspendu':p.enabled?'Actif':'Désactivé'}</span></div><p>{m.description}</p><small>{m.publisher} · {m.version}<br/>Import local · identité non vérifiée</small>
 <div class="extension-actions">{#if m.pages?.some(p=>p.screen)}<p>Rubriques fournies : {m.pages.filter(p=>p.screen).map(p=>p.label).join(" · ")}. Désactiver ces pages conserve la recherche, les fiches, les lecteurs et vos données.</p>{/if}{#if m.api!==3}<button class="secondary" disabled={busy||!p.enabled||appearance.safeMode||appearance.selected===m.id} onclick={()=>choose(m.id)}>{appearance.selected===m.id?'Choisi pour vous':'Utiliser pour moi'}</button>{/if}{#if user.admin}<button class="text-action" disabled={busy||appearance.safeMode&&!p.enabled} onclick={()=>toggle(p)}>{p.enabled?'Désactiver':'Activer'}</button>{/if}</div>
 {#if user.admin}<div class="extension-admin-actions"><button class="text-action" disabled={busy} onclick={()=>editAccess(p)}>Accès des utilisateurs</button><button class="text-action" disabled={busy} onclick={()=>{removing=m.id;editing=''}}>Désinstaller</button></div>
 {#if editing===m.id}<fieldset class="extension-access"><legend>Qui peut utiliser {m.name} ?</legend><p>Les administrateurs y ont accès. Les autres comptes doivent être autorisés ici.</p>{#each users.filter(a=>!a.admin) as a}<label><input type="checkbox" checked={grants.includes(a.id)} disabled={busy} onchange={e=>changeAccess(a.id,e.currentTarget.checked)}/>{a.name}{a.disabled?' · compte désactivé':''}</label>{/each}{#if !users.some(a=>!a.admin)}<p>Aucun autre compte. Vous pouvez en créer dans Utilisateurs.</p>{/if}<div class="extension-actions"><button class="secondary" disabled={busy} onclick={()=>saveAccess(p)}>Enregistrer les accès</button><button class="text-action" disabled={busy} onclick={()=>editing=''}>Fermer</button></div></fieldset>{/if}
 {#if removing===m.id}<div class="extension-warning"><strong>Retirer {m.name} ?</strong><p>Le paquet, ses pages, ses autorisations et les choix associés seront retirés. Un thème retiré laisse place à l’apparence Mosaune. Vos médias, fiches et playlists sont conservés.</p><div class="extension-actions"><button class="secondary" disabled={busy} onclick={()=>remove(p)}>Confirmer la désinstallation</button><button class="text-action" disabled={busy} onclick={()=>removing=''}>Annuler</button></div></div>{/if}{/if}
 </div></article>
 {:else if selectedProvider && user.admin}
  {#key selectedProvider.id}<ProviderDetails {token} provider={selectedProvider} changed={async()=>{await load();await refresh();extensionsChanged();}} browse={()=>{tab='available';void show('')}}/>{/key}
 {:else if selected==='audio-tags' && user.admin}
  <RecognitionPackages {token} detail="audiotags" refreshKey={recognitionRefresh} select={id=>show(id?'recognition.'+id:'')}/>
 {:else if selected.startsWith('recognition.') && user.admin}
  <RecognitionPackages {token} refreshKey={recognitionRefresh} detail={selected.slice(12)} select={id=>show(id?'recognition.'+id:'')}/>
 {:else if selectedNative}
  <span class="extension-tag">{selectedNative.family}</span><p>{selectedNative.description}</p>
  <button class="secondary" onclick={()=>openSettings(selectedNative?.settings??'account')}>Ouvrir ses réglages</button>
  {#if user.admin&&bundles.some(b=>'builtin.'+b.id===selected)}{#key selected}<MediaBundlePanel {token} id={selected.slice(8)} select={id=>show(id)} changed={()=>{void load();}}/>{/key}
  {:else}<p>Cette fonction reste intégrée au serveur ; son extraction est prévue.</p>{/if}
 {:else}
  <div class="extension-tabs" aria-label="Catalogue des extensions"><button class:selected={tab==='installed'} aria-pressed={tab==='installed'} onclick={()=>tab='installed'}>Sur le serveur</button><button class:selected={tab==='available'} aria-pressed={tab==='available'} onclick={()=>tab='available'}>À télécharger</button></div>
  {#if tab==='available'}
   {#if user.admin}<div class="extension-grid">{#each bundles as b}<ExtensionCard id={'available.'+b.id} name={b.name} family="Famille média" description="Installer ou gérer ensemble ses composants et son écran." status={bundleStatus[b.state]} open={()=>show('builtin.'+b.id)}/>{/each}</div>{/if}
   {#if user.admin}<RecognitionPackages {token} refreshKey={recognitionRefresh} hideFactory excludeIds={bundledComponentIds} {tab} select={kind=>show('recognition.'+kind)}/>{/if}
   {#if user.admin}<ExtensionRepositoryStore {token} groupedComponents={[...groupedScreens,...suppliedComponentIds,...bundledComponentIds]} safeMode={appearance.safeMode} installed={repositoryInstalled}/>{/if}
   {#if user.admin}<details class="extension-import"><summary>Importer un fichier depuis cet appareil</summary><MediaBundleImport {token} installed={async kind=>{await load();tab="installed";await show("builtin."+kind);notice="Archive de famille installée. Vérifiez ses composants puis activez la famille si nécessaire.";}}/><RecognitionImport {token} installed={()=>{recognitionRefresh++;}}/><ProviderPackageImport {token} managers={providers.filter(p=>p.packages)} installed={providerInstalled}/>
   <section class="extension-import"><div><span class="eyebrow">UN PAQUET, UNE NOUVELLE POSSIBILITÉ</span><h3>Installer des pages ou un thème</h3><p>Importez un fichier <strong>.mosaune.zip</strong>. Mosaune vérifie son format et sa compatibilité avant de proposer l’installation.</p></div>
 <label class="extension-file">Paquet à vérifier<input bind:this={input} type="file" accept=".zip,application/zip" disabled={busy} onchange={inspect}/></label>
 <a class="text-action" href="/api/v1/extensions/pages-example" download>Télécharger l’exemple Salon personnel</a>
 <a class="text-action" href="/api/v1/extensions/example" download>Télécharger Ambre 2</a>
 <a class="text-action" href="/api/v1/extensions/theme-template" download>Télécharger le fichier de thème complet (JSON)</a>
 <a class="text-action" href="/api/v1/extensions/theme-contract" download>Télécharger le catalogue des réglages</a>
 <small>Les paquets importés sont des sources locales non vérifiées. Le nom d’éditeur est déclaratif ; l’empreinte vérifie le contenu, pas l’identité de son auteur.</small>
 {#if preview}{@const m=preview.package.manifest}<div class="extension-preview"><span class="extension-tag">{preview.previousVersion?'Mise à jour à vérifier':'Prêt à installer'}</span><h3>{m.name} <small>{preview.previousVersion?preview.previousVersion+' → ':''}{m.version}</small></h3><p>{m.description}</p><dl><dt>Éditeur déclaré</dt><dd>{m.publisher}</dd><dt>Type</dt><dd>{m.api===3?'Interface · pages et menus':'Interface · thème déclaratif'}</dd><dt>Compatibilité</dt><dd>Mosaune {m.minCore} ou plus · API {m.api}</dd><dt>Capacité requise</dt><dd>{m.requires.join(', ')}</dd><dt>Accès demandés</dt><dd>{m.api===3?'Pages déclaratives : collections filtrées par les droits du compte, aucun code exécuté.':'Aucun accès aux médias, fichiers, comptes ou réseau.'}</dd><dt>Empreinte SHA-256</dt><dd class="extension-hash">{preview.package.checksum}</dd></dl>
 <p>Le paquet reste désactivé après installation. {preview.previousVersion?'La mise à jour le désactive jusqu’à votre prochaine activation.':''}</p><div class="extension-actions"><button class="primary" disabled={busy} onclick={install}>{busy?'Installation…':preview.previousVersion?'Installer cette mise à jour':'Installer ce paquet'}</button><button class="text-action" disabled={busy} onclick={()=>{preview=null;file=null;input.value=''}}>Annuler</button></div></div>{/if}
 </section></details>{:else}<p>Votre administrateur peut installer de nouveaux paquets et vous y donner accès.</p>{/if}
  {:else}
   <div class="extension-toolbar"><label>Rechercher une extension<input type="search" bind:value={filter} placeholder="Nom ou description…"/></label><button class="text-action" disabled={busy} onclick={()=>void load()}>Actualiser</button></div>
   {#if loading}<p role="status">Chargement des extensions…</p>{/if}
   <div class="extension-grid">
    {#if user.admin}<RecognitionPackages {token} refreshKey={recognitionRefresh} hideFactory excludeIds={bundledComponentIds} {filter} select={kind=>show('recognition.'+kind)}/>{/if}
    {#if user.admin}{#each providers.filter(p=>matches(p.name,filter)) as provider (provider.id)}<ExtensionCard id={'provider.'+provider.id} name={provider.name} family="Métadonnées" description="Informations, images et fonctions fournies par cet adaptateur." status={providerStatus(provider)} enabled={provider.enabled&&provider.runtime?.installed!==false} disabled={busy||provider.runtime?.installed===false} toggle={provider.activation?()=>toggleProvider(provider):undefined} open={()=>show('provider.'+provider.id)}/>{/each}{/if}
    {#each visiblePackages as p (p.manifest.id)}<ExtensionCard id={p.manifest.id} name={p.manifest.name} family={p.manifest.api===3?'Interface · pages et menus':'Interface · thème'} description={p.manifest.description} status={appearance.safeMode?'Suspendu':p.enabled?'Actif':'Désactivé'} enabled={p.enabled} disabled={busy||appearance.safeMode&&!p.enabled} toggle={user.admin?()=>toggle(p):undefined} open={()=>show(p.manifest.id)}/>{/each}
    {#if user.admin}{#each native.filter(p=>matches(p.name+' '+p.description,filter)) as item}<ExtensionCard id={item.id} name={item.name} family={item.family} description={item.description} status={bundleStatus[bundles.find(b=>"builtin."+b.id===item.id)?.state??""]??"Intégré"} open={()=>show(item.id)}/>{/each}{/if}
   </div>
   {#if !loading&&!user.admin&&!visiblePackages.length&&(!user.admin||(!native.some(p=>matches(p.name+' '+p.description,filter))&&!providers.some(p=>matches(p.name,filter))))}<div class="extension-empty"><strong>{filter?'Aucune extension ne correspond.':'Aucun thème autorisé pour le moment.'}</strong><p>{filter?'Essayez un autre nom.':'Votre administrateur peut vous donner accès aux thèmes installés.'}</p></div>{/if}
   <div class="extension-native-choice"><div><strong>Apparence Mosaune</strong><p>Le style d’origine reste toujours disponible. Clair ou sombre : rendez-vous dans Mon compte.</p></div><button class="secondary" disabled={busy||!appearance.selected} onclick={()=>choose('')}>{appearance.selected?'Rétablir pour moi':'Choisie pour vous'}</button></div>
   {#if appearance.selected&&!appearance.effective}<p class="folder-help">Votre thème préféré est conservé mais indisponible actuellement. L’apparence d’origine est utilisée.</p>{/if}
  {/if}
 {/if}
</section>
