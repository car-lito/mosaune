<script lang="ts">
 import {onMount,tick} from 'svelte';
 import ExtensionCard from './ExtensionCard.svelte';
 import RecognitionPackages from './RecognitionPackages.svelte';
 import RecognitionImport from './RecognitionImport.svelte';
 const recognitionNames:Record<string,string>={films:'Reconnaissance des films',series:'Reconnaissance des séries',music:'Reconnaissance musicale'};
 import ExtensionConfiguration from './ExtensionConfiguration.svelte';
 import ProviderPackageImport from './ProviderPackageImport.svelte';
 import ProviderPackagePanel from './ProviderPackagePanel.svelte';
 import ExtensionRepositoryStore from './ExtensionRepositoryStore.svelte';
 import {accountRequest,type Account} from './accounts';
 import {extensionsChanged,type ExtensionPackage,type ExtensionAppearance,type ProviderPackageState} from './extensions';
 export let user:Account;export let token:string;
 export let refresh:()=>Promise<void>;export let openSettings:(id:string)=>void;
 let tab:'installed'|'available'='installed';let selected='';let filter='';let heading:HTMLElement;
 let runtime:{serviceAccess?:boolean;providerEnabled:boolean;enabled:boolean;effective:boolean;loaded:boolean;safeMode:boolean}|null=null;
 let providerPackage:ProviderPackageState|null=null;
 async function providerInstalled(){await load();tab='installed';await show('tmdb');notice='Paquet fournisseur installé. Vos fiches et votre connexion sont conservées.';}
 async function repositoryInstalled(kind:string,id:string){await load();tab='installed';await show(kind==='provider'?'tmdb':(kind==='recognition'||kind==='audio-tags')?'recognition.'+id:id);notice='Paquet installé. Retrouvez ses réglages ci-dessous.';}
 $: selectedPackage=packages.find(p=>p.manifest.id===selected);
 $: selectedNative=native.find(p=>p.id===selected);
 $: visiblePackages=packages.filter(p=>(p.manifest.name+' '+p.manifest.description).toLocaleLowerCase().includes(filter.toLocaleLowerCase()));
 async function show(id:string){selected=id;editing='';removing='';await tick();heading?.focus();heading?.closest('.settings-content')?.scrollTo({top:0});}
 async function back(){const previous=selected;selected='';editing='';removing='';await tick();document.getElementById('extension-card-'+previous)?.focus();}
 async function toggleRuntime(){if(!runtime)return;await perform(async()=>{runtime=await accountRequest(token,'metadata/tmdb-runtime','PUT',{enabled:!runtime?.enabled});notice=runtime?.enabled?'Moteur extension choisi. L’état actif ou désactivé de TMDB est conservé.':'Moteur intégré choisi. L’état actif ou désactivé de TMDB est conservé.';});}
 async function toggleProvider(){if(!runtime)return;await perform(async()=>{const enabled=!runtime!.providerEnabled;await accountRequest(token,'metadata/providers/tmdb/state','PUT',{enabled});notice=enabled?'TMDB activé. Les consultations reprendront si son accès en ligne est autorisé.':'TMDB désactivé. Les recherches et traitements en cours sont arrêtés ; vos fiches et affiches conservées restent disponibles.';});}
 function matches(name:string,query:string){return name.toLocaleLowerCase().includes(query.toLocaleLowerCase())}

 let packages:ExtensionPackage[]=[];let users:Account[]=[];let appearance:ExtensionAppearance={selected:'',effective:'',stamp:'',safeMode:false};
 let recognitionRefresh=0;
 let busy=false;let loading=true;let error='';let notice='';let file:File|null=null;let input:HTMLInputElement;
 let preview:{package:ExtensionPackage;previousChecksum:string;previousVersion:string}|null=null;
 let removing='';let editing='';let grants:string[]=[];
 const native=[
 {id:'builtin.films',name:'Films',family:'Médias',description:'Bibliothèques, fiches et lecture des films.',settings:'libraries'},
 {id:'builtin.series',name:'Séries',family:'Médias',description:'Saisons, épisodes et reprise de lecture.',settings:'libraries'},
 {id:'builtin.music',name:'Musique',family:'Médias',description:'Albums, playlists et file d’écoute.',settings:'libraries'},
 {id:'builtin.musicbrainz',name:'MusicBrainz',family:'Métadonnées',description:'Identification des albums, artistes et morceaux.',settings:'metadata'},
 {id:'builtin.radarr',name:'Radarr',family:'Connecteurs',description:'Demandes et suivi de vos films.',settings:'downloads'},
 {id:'builtin.sonarr',name:'Sonarr',family:'Connecteurs',description:'Demandes et suivi de vos séries.',settings:'downloads'}];
 async function load(){try{const [list,a]=await Promise.all([accountRequest(token,'extensions'),accountRequest(token,'extensions/appearance')]);packages=list.packages;appearance=a;if(user.admin){const result=await Promise.all([accountRequest(token,'accounts'),accountRequest(token,'metadata/tmdb-runtime'),accountRequest(token,'provider-packages/tmdb')]);users=result[0];runtime=result[1];providerPackage=result[2];}if(selected&&!selected.startsWith('builtin.')&&!selected.startsWith('recognition.')&&selected!=='tmdb'&&selected!=='audio-tags'&&!packages.some(p=>p.manifest.id===selected))selected='';}catch(e){error=e instanceof Error?e.message:'Extensions indisponibles.'}finally{loading=false;recognitionRefresh++}}
 onMount(()=>{void load();const focus=()=>{if(!busy)void load()};window.addEventListener('focus',focus);return()=>window.removeEventListener('focus',focus)});
 async function perform(action:()=>Promise<void>){if(busy)return;busy=true;error='';notice='';try{await action();await load();extensionsChanged()}catch(e){error=e instanceof Error?e.message:'Opération indisponible.'}finally{busy=false}}
 async function upload(path:string){if(!file)throw Error('Choisissez un paquet.');const r=await fetch('/api/v1/extensions/'+path,{method:'POST',headers:{'Content-Type':'application/zip','X-Mosaune-Token':token,'X-Mosaune-Package-Checksum':preview?.package.checksum??'','X-Mosaune-Previous-Checksum':preview?.previousChecksum??''},body:file});const result=await r.json();if(!r.ok)throw Error(result.error||'Paquet non accepté.');return result;}
 async function inspect(){file=input.files?.[0]??null;preview=null;notice='';error='';if(!file)return;if(file.size>262144){error='Le paquet doit faire moins de 256 Ko.';file=null;return}await perform(async()=>{preview=await upload('preview')})}
 async function install(){await perform(async()=>{await upload('install');notice='Paquet installé, désactivé par défaut. Activez-le puis choisissez-le pour votre compte.';preview=null;file=null;if(input)input.value='';tab='installed';selected='';})}
 async function toggle(p:ExtensionPackage){await perform(async()=>{await accountRequest(token,'extensions/'+p.manifest.id+'/state','PUT',{enabled:!p.enabled,checksum:p.checksum});notice=p.enabled?'Extension désactivée. Vos choix restent conservés.':'Extension activée. Chaque compte autorisé peut maintenant la choisir.';})}
 async function choose(id:string){await perform(async()=>{await accountRequest(token,'account/extension-theme','PUT',{id});notice=id?'Thème choisi. Le style s’applique à toutes les rubriques.':'Apparence Mosaune rétablie pour votre compte.';})}
 async function remove(p:ExtensionPackage){await perform(async()=>{await accountRequest(token,'extensions/'+p.manifest.id,'DELETE',{checksum:p.checksum});removing='';editing='';selected='';notice='Paquet retiré. Les comptes concernés retrouvent l’apparence Mosaune ; votre collection est conservée.';})}
 function editAccess(p:ExtensionPackage){editing=p.manifest.id;grants=[...p.users];removing=''}
 function changeAccess(id:string,checked:boolean){grants=checked?[...grants,id]:grants.filter(x=>x!==id)}
 async function saveAccess(p:ExtensionPackage){await perform(async()=>{await accountRequest(token,'extensions/'+p.manifest.id+'/access','PUT',{users:grants,checksum:p.checksum});editing='';notice='Accès enregistrés. Les administrateurs gardent accès à toutes les extensions.';})}
</script>
<section class="extension-settings" aria-busy={busy}>
 {#if selected}<button class="back-link" onclick={back}>← Toutes les extensions</button>{/if}
 <span class="eyebrow">VOTRE MOSAUNE</span>
 <h2 bind:this={heading} tabindex="-1">{selectedPackage?.manifest.name ?? selectedNative?.name ?? recognitionNames[selected.replace('recognition.','')] ?? (selected==='audio-tags'?'Informations musicales':selected==='tmdb'?'TMDB':'Les extensions')}</h2>
 {#if !selected}<p>Vos médias, vos services et votre apparence. Ouvrez une carte pour retrouver tous ses réglages.</p>{/if}
 {#if appearance.safeMode}<div class="extension-warning" role="status"><strong>Mode de secours</strong><p>Les thèmes externes, les modules de reconnaissance et le moteur extension de TMDB sont suspendus. Le fournisseur TMDB reste désactivé si vous l’avez coupé.</p></div>{/if}
 {#if error}<p class="form-error" role="alert">{error}</p>{/if}{#if notice}<p class="extension-notice" role="status">{notice}</p>{/if}
 {#if selectedPackage}
  {@const p=selectedPackage}{@const m=p.manifest}
  <article class="extension-detail"> <svg class="extension-palette" viewBox="0 0 360 88" role="img" aria-label={'Palettes claire et sombre de '+m.name}><rect width="180" height="88" fill={p.preview.light.canvas}/><rect x="180" width="180" height="88" fill={p.preview.dark.canvas}/><circle cx="42" cy="44" r="19" fill={p.preview.light.accent}/><rect x="77" y="28" width="74" height="8" rx="4" fill={p.preview.light.ink}/><rect x="77" y="48" width="52" height="5" rx="2" fill={p.preview.light.muted}/><circle cx="222" cy="44" r="19" fill={p.preview.dark.accent}/><rect x="257" y="28" width="74" height="8" rx="4" fill={p.preview.dark.ink}/><rect x="257" y="48" width="52" height="5" rx="2" fill={p.preview.dark.muted}/></svg>
 <div class="extension-card-body"><div class="extension-card-heading"><h3>{m.name}</h3><span class="extension-tag">{appearance.safeMode?'Suspendu':p.enabled?'Actif':'Désactivé'}</span></div><p>{m.description}</p><small>{m.publisher} · {m.version}<br/>Import local · identité non vérifiée</small>
 <div class="extension-actions"><button class="secondary" disabled={busy||!p.enabled||appearance.safeMode||appearance.selected===m.id} onclick={()=>choose(m.id)}>{appearance.selected===m.id?'Choisi pour vous':'Utiliser pour moi'}</button>{#if user.admin}<button class="text-action" disabled={busy||appearance.safeMode&&!p.enabled} onclick={()=>toggle(p)}>{p.enabled?'Désactiver':'Activer'}</button>{/if}</div>
 {#if user.admin}<div class="extension-admin-actions"><button class="text-action" disabled={busy} onclick={()=>editAccess(p)}>Accès des utilisateurs</button><button class="text-action" disabled={busy} onclick={()=>{removing=m.id;editing=''}}>Désinstaller</button></div>
 {#if editing===m.id}<fieldset class="extension-access"><legend>Qui peut utiliser {m.name} ?</legend><p>Les administrateurs y ont accès. Les autres comptes doivent être autorisés ici.</p>{#each users.filter(a=>!a.admin) as a}<label><input type="checkbox" checked={grants.includes(a.id)} disabled={busy} onchange={e=>changeAccess(a.id,e.currentTarget.checked)}/>{a.name}{a.disabled?' · compte désactivé':''}</label>{/each}{#if !users.some(a=>!a.admin)}<p>Aucun autre compte. Vous pouvez en créer dans Utilisateurs.</p>{/if}<div class="extension-actions"><button class="secondary" disabled={busy} onclick={()=>saveAccess(p)}>Enregistrer les accès</button><button class="text-action" disabled={busy} onclick={()=>editing=''}>Fermer</button></div></fieldset>{/if}
 {#if removing===m.id}<div class="extension-warning"><strong>Retirer {m.name} ?</strong><p>Le paquet, ses autorisations et les choix associés seront retirés. Les comptes concernés retrouveront l’apparence Mosaune. Vos médias, fiches et playlists sont conservés.</p><div class="extension-actions"><button class="secondary" disabled={busy} onclick={()=>remove(p)}>Confirmer la désinstallation</button><button class="text-action" disabled={busy} onclick={()=>removing=''}>Annuler</button></div></div>{/if}{/if}
 </div></article>
 {:else if selected==='tmdb' && user.admin && runtime}
  <p>Fournisseur de métadonnées pour les films et séries. Le module expérimental gère maintenant les recherches et les fiches détaillées ; les images sont téléchargées par Mosaune.</p>
  <div class="extension-native-choice"><div><strong>{runtime.providerEnabled?(runtime.serviceAccess===false?'TMDB actif · accès en ligne retiré':'TMDB actif'):'TMDB désactivé'}</strong><p>Ce réglage arrête toutes les recherches, suggestions et récupérations d’images de TMDB, quel que soit son moteur. Vos données et votre clé sont conservées.</p></div><button class="secondary" role="switch" aria-label="Activer TMDB" aria-checked={runtime.providerEnabled} disabled={busy} onclick={toggleProvider}>{runtime.providerEnabled?'Désactiver TMDB':'Activer TMDB'}</button></div>
  <details class="extension-import"><summary>Moteur de traitement · avancé</summary><p>{runtime.enabled?'Moteur extension choisi':'Moteur intégré choisi'}. {runtime.safeMode?'Le mode de secours utilise le moteur intégré si TMDB est actif.':''} Changer de moteur ne réactive jamais un fournisseur désactivé.</p><button class="secondary" disabled={busy||runtime.safeMode&&!runtime.enabled} onclick={toggleRuntime}>{runtime.enabled?'Utiliser le moteur intégré':'Utiliser le moteur extension'}</button></details>
  <ProviderPackagePanel {token} state={providerPackage} changed={load} browse={()=>{tab='available';void show('')}}/>
  {#key providerPackage?.package?.checksum}<ExtensionConfiguration id="mosaune.tmdb-search" {token} refresh={async()=>{await load();await refresh();}}/>{/key}
 {:else if selected==='audio-tags' && user.admin}
  <RecognitionPackages {token} detail="audiotags" refreshKey={recognitionRefresh} select={id=>show(id?'recognition.'+id:'')}/>
 {:else if selected.startsWith('recognition.') && user.admin}
  <RecognitionPackages {token} refreshKey={recognitionRefresh} detail={selected.slice(12)} select={id=>show(id?'recognition.'+id:'')}/>
 {:else if selectedNative}
  <span class="extension-tag">{selectedNative.family} · intégré</span><p>{selectedNative.description}</p>
  <p>Cette fonction fait encore partie du programme. Elle sera séparée en extension lors d’une prochaine étape ; elle ne peut pas encore être désactivée ou désinstallée ici.</p>
  <button class="secondary" onclick={()=>openSettings(selectedNative?.settings??'account')}>Ouvrir ses réglages</button>
 {:else}
  <div class="extension-tabs" aria-label="Catalogue des extensions"><button class:selected={tab==='installed'} aria-pressed={tab==='installed'} onclick={()=>tab='installed'}>Sur le serveur</button><button class:selected={tab==='available'} aria-pressed={tab==='available'} onclick={()=>tab='available'}>À télécharger</button></div>
  {#if tab==='available'}
   {#if user.admin}<RecognitionPackages {token} refreshKey={recognitionRefresh} {tab} select={kind=>show('recognition.'+kind)}/>{/if}
   {#if user.admin}<ExtensionRepositoryStore {token} safeMode={appearance.safeMode} installed={repositoryInstalled}/>{/if}
   {#if user.admin}<details class="extension-import"><summary>Importer un fichier depuis cet appareil</summary><RecognitionImport {token} installed={()=>{recognitionRefresh++;}}/><ProviderPackageImport {token} installed={providerInstalled}/>
   <section class="extension-import"><div><span class="eyebrow">UN PAQUET, UNE NOUVELLE POSSIBILITÉ</span><h3>Installer un thème</h3><p>Importez un fichier <strong>.mosaune.zip</strong>. Mosaune vérifie son format et sa compatibilité avant de proposer l’installation.</p></div>
 <label class="extension-file">Paquet à vérifier<input bind:this={input} type="file" accept=".zip,application/zip" disabled={busy} onchange={inspect}/></label>
 <a class="text-action" href="/api/v1/extensions/example" download>Télécharger Ambre 2</a>
 <a class="text-action" href="/api/v1/extensions/theme-template" download>Télécharger le fichier de thème complet (JSON)</a>
 <a class="text-action" href="/api/v1/extensions/theme-contract" download>Télécharger le catalogue des réglages</a>
 <small>Les paquets importés sont des sources locales non vérifiées. Le nom d’éditeur est déclaratif ; l’empreinte vérifie le contenu, pas l’identité de son auteur.</small>
 {#if preview}{@const m=preview.package.manifest}<div class="extension-preview"><span class="extension-tag">{preview.previousVersion?'Mise à jour à vérifier':'Prêt à installer'}</span><h3>{m.name} <small>{preview.previousVersion?preview.previousVersion+' → ':''}{m.version}</small></h3><p>{m.description}</p><dl><dt>Éditeur déclaré</dt><dd>{m.publisher}</dd><dt>Type</dt><dd>Interface · thème déclaratif</dd><dt>Compatibilité</dt><dd>Mosaune {m.minCore} ou plus · API {m.api}</dd><dt>Capacité requise</dt><dd>{m.requires.join(', ')}</dd><dt>Accès demandés</dt><dd>Aucun accès aux médias, fichiers, comptes ou réseau.</dd><dt>Empreinte SHA-256</dt><dd class="extension-hash">{preview.package.checksum}</dd></dl>
 <p>L’installation n’applique pas automatiquement ce thème. {preview.previousVersion?'La mise à jour le désactive jusqu’à votre prochaine activation.':''}</p><div class="extension-actions"><button class="primary" disabled={busy} onclick={install}>{busy?'Installation…':preview.previousVersion?'Installer cette mise à jour':'Installer ce paquet'}</button><button class="text-action" disabled={busy} onclick={()=>{preview=null;file=null;input.value=''}}>Annuler</button></div></div>{/if}
 </section></details>{:else}<p>Votre administrateur peut installer de nouveaux paquets et vous y donner accès.</p>{/if}
  {:else}
   <div class="extension-toolbar"><label>Rechercher une extension<input type="search" bind:value={filter} placeholder="Nom ou description…"/></label><button class="text-action" disabled={busy} onclick={()=>void load()}>Actualiser</button></div>
   {#if loading}<p role="status">Chargement des extensions…</p>{/if}
   <div class="extension-grid">
    {#if user.admin}<RecognitionPackages {token} refreshKey={recognitionRefresh} {filter} select={kind=>show('recognition.'+kind)}/>{/if}
    {#if user.admin&&runtime&&matches('TMDB métadonnées films séries',filter)}<ExtensionCard id="tmdb" name="TMDB" family="Métadonnées · expérimental" description="Recherche et fiches de films et séries, avec le moteur d’extensions." status={runtime.providerEnabled?(runtime.serviceAccess===false?'Accès en ligne retiré':'Actif'):'Désactivé'} enabled={runtime.providerEnabled} disabled={busy} toggle={toggleProvider} open={()=>show('tmdb')}/>{/if}
    {#each visiblePackages as p (p.manifest.id)}<ExtensionCard id={p.manifest.id} name={p.manifest.name} family="Interface · thème" description={p.manifest.description} status={appearance.safeMode?'Suspendu':p.enabled?'Actif':'Désactivé'} enabled={p.enabled} disabled={busy||appearance.safeMode&&!p.enabled} toggle={user.admin?()=>toggle(p):undefined} open={()=>show(p.manifest.id)}/>{/each}
    {#if user.admin}{#each native.filter(p=>matches(p.name+' '+p.description,filter)) as item}<ExtensionCard id={item.id} name={item.name} family={item.family} description={item.description} status="Intégré" open={()=>show(item.id)}/>{/each}{/if}
   </div>
   {#if !loading&&!user.admin&&!visiblePackages.length&&(!user.admin||(!native.some(p=>matches(p.name+' '+p.description,filter))&&(!runtime||!matches('TMDB métadonnées films séries',filter))))}<div class="extension-empty"><strong>{filter?'Aucune extension ne correspond.':'Aucun thème autorisé pour le moment.'}</strong><p>{filter?'Essayez un autre nom.':'Votre administrateur peut vous donner accès aux thèmes installés.'}</p></div>{/if}
   <div class="extension-native-choice"><div><strong>Apparence Mosaune</strong><p>Le style d’origine reste toujours disponible. Clair ou sombre : rendez-vous dans Mon compte.</p></div><button class="secondary" disabled={busy||!appearance.selected} onclick={()=>choose('')}>{appearance.selected?'Rétablir pour moi':'Choisie pour vous'}</button></div>
   {#if appearance.selected&&!appearance.effective}<p class="folder-help">Votre thème préféré est conservé mais indisponible actuellement. L’apparence d’origine est utilisée.</p>{/if}
  {/if}
 {/if}
</section>
