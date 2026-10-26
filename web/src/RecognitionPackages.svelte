<script lang="ts">
 import {onMount} from 'svelte';
 import {accountRequest} from './accounts';
 import ExtensionCard from './ExtensionCard.svelte';
 export let token:string;
 export let tab:'installed'|'available'='installed';
 export let detail='';
 export let filter='';
 export let refreshKey=0;
 export let select:(kind:string)=>void;
 type Package={kind:string;id:string;version:string;installed:boolean;enabled:boolean;name:string;publisher:string;factory:boolean;selected:boolean;previousVersion:string};
 let packages:Package[]=[];let revision=0;let safeMode=false;let busy=false;let error='';let notice='';let confirming=false;
 let lastRefresh=refreshKey;
 $: if(refreshKey!==lastRefresh){lastRefresh=refreshKey;void load().catch(e=>error=e.message);}
 const names:Record<string,string>={films:'Reconnaissance des films',series:'Reconnaissance des séries',music:'Reconnaissance musicale',audiotags:'Informations des fichiers musicaux','catalogue-films':'Catalogue des films','catalogue-series':'Catalogue des séries','catalogue-music':'Catalogue musical'};
 $: current=packages.find(p=>p.id===detail||p.factory&&p.kind===detail);
 $: visible=packages.filter(p=>p.installed===(tab==='installed')&&(p.name+' '+names[p.kind]).toLocaleLowerCase().includes(filter.toLocaleLowerCase()));
 async function load(){const result=await accountRequest(token,'recognition-packages');packages=result.packages;revision=result.revision;safeMode=result.safeMode;}
 onMount(()=>{void load().catch(e=>error=e.message);const focus=()=>{if(!busy)void load().catch(e=>error=e.message)};window.addEventListener('focus',focus);return()=>window.removeEventListener('focus',focus)});
 async function change(p:Package,operation:string){if(busy)return;busy=true;error='';notice='';try{await accountRequest(token,'recognition-packages/'+encodeURIComponent(p.id),'POST',{operation,revision});confirming=false;await load();const messages:Record<string,string>={install:'Module installé. Activez-le puis choisissez-le pour les scans.',enable:'Module activé. Le choix pour les scans est indiqué sur sa fiche.',disable:'Module désactivé. Votre catalogue est conservé.',select:'Ce module sera utilisé pour les prochains scans. Les analyses non enregistrées de ce média ont été interrompues.',restore:'Version précédente restaurée. Votre choix et votre collection sont conservés.',remove:'Module désinstallé. Votre catalogue est conservé. Choisissez un autre module pour poursuivre les scans.'};notice=messages[operation];if(isCatalogue(p)){notice=operation==='remove'||operation==='disable'?'Organisation désactivée. Les fichiers restent accessibles individuellement ; les données sont conservées.':'Module de catalogue mis à jour. Son choix est indiqué sur cette fiche.';window.dispatchEvent(new Event('mosaune:catalogue-changed'));}if(p.kind==='audiotags'&&operation==='remove')notice='Lecteur retiré. Les données enregistrées sont conservées ; aucun autre lecteur ne sera sélectionné automatiquement.';if(operation==='remove'&&!p.factory)select('');}catch(e){error=e instanceof Error?e.message:'Modification indisponible.';try{await load()}catch{}}finally{busy=false}}
 const isCatalogue=(p:Package)=>p.kind.startsWith('catalogue-');
 function status(p:Package){return safeMode?'Suspendu':!p.enabled?'Désactivé':p.selected?(isCatalogue(p)?'Utilisé pour le catalogue':'Utilisé pour les scans'):'Actif · non sélectionné'}
</script>
{#if error}<p class="form-error" role="alert">{error}</p>{/if}
{#if notice}<p class="extension-notice" role="status">{notice}</p>{/if}
{#if detail}
 {#if current}
  <article class="extension-detail"><div class="extension-card-body">
   <h3>{current.name}</h3>
   <span class="extension-tag">{isCatalogue(current)?'Catalogue':current.kind==='audiotags'?'Informations musicales':'Reconnaissance'} · {current.version} · {current.installed?status(current):'Non installé'}</span>
   <p>{current.factory?'Fourni avec Mosaune':'Module externe · éditeur déclaré : '+current.publisher+' (non vérifié)'}</p>
   {#if isCatalogue(current)}
   <p>Définit les groupes, leurs titres et l’ordre des fichiers à partir des métadonnées déjà visibles pour votre compte. Aucun accès aux fichiers, chemins, réseau ou clés.</p>
   <p>Un seul module est choisi par catalogue. S’il est désactivé ou retiré, les fichiers s’affichent individuellement. Les informations et progressions restent conservées. La réactivation reforme les groupes sans nouveau scan.</p>
   {:else if current.kind==='audiotags'}
   <p>Un seul lecteur est choisi pour les informations intégrées aux fichiers musicaux. Activer un autre lecteur ne le sélectionne pas automatiquement.</p>
   <p>Accès accordé à l’installation : blocs limités du fichier en cours, en lecture seule. Aucun choix de chemin, parcours des dossiers, accès au réseau ou aux clés.</p>
   <p>Si le lecteur choisi est désactivé ou retiré, les nouveaux fichiers utilisent leur nom. Aucun autre lecteur ne prend sa place automatiquement. Les fiches et pochettes enregistrées sont conservées ; les fichiers inchangés ne sont pas réanalysés.</p>
   {:else}
   <p>Un seul reconnaisseur est choisi par type de média. En désactivant ou retirant le module choisi, les scans sont suspendus jusqu’à sa réactivation ou au choix d’un remplaçant.</p>
   <p>Reconnaît les formats, les titres et les numéros d’épisodes à partir des noms de fichiers. Ce module s’exécute uniquement pendant les analyses, sans accès au réseau.</p>
   {/if}
   {#if !isCatalogue(current)}<p>La désactivation arrête les analyses non enregistrées de ce type de média. Une sauvegarde déjà commencée se termine avant le changement d’état. Les médias, affiches et progressions déjà enregistrés restent disponibles.</p>
   <p>Il s’agit d’un composant d’analyse : les écrans, les lecteurs et les autres fonctions médias sont encore intégrés à Mosaune.</p>{/if}
   <div class="extension-actions">
    {#if current.installed}
     <button class="secondary" disabled={busy||safeMode&&!current.enabled} onclick={()=>current&&change(current,current.enabled?'disable':'enable')}>{current.enabled?'Désactiver':'Activer'}</button>
     {#if !current.selected}<button class="secondary" disabled={busy||safeMode||!current.enabled} onclick={()=>current&&change(current,'select')}>{isCatalogue(current)?'Utiliser pour le catalogue':'Utiliser pour les scans'}</button>{/if}
     {#if current.previousVersion}<button class="secondary" disabled={busy||safeMode} onclick={()=>current&&change(current,'restore')}>Restaurer la version {current.previousVersion}</button>{/if}
     <button class="text-action" disabled={busy} onclick={()=>confirming=true}>Désinstaller</button>
    {:else}<button class="secondary" disabled={busy||safeMode} onclick={()=>current&&change(current,'install')}>Installer la version fournie avec Mosaune</button>{/if}
   </div>
   {#if confirming&&current.installed}<div class="extension-warning"><strong>Désinstaller ce module ?</strong><p>Son programme installé sera retiré. {isCatalogue(current)?'Les fichiers seront affichés individuellement si ce module était choisi.':current.kind==='audiotags'?'Les nouvelles informations internes ne seront plus lues si ce lecteur était choisi.':'Choisissez un autre reconnaisseur pour poursuivre les scans si celui-ci était choisi.'} Vos fichiers et votre collection sont conservés.</p><div class="extension-actions"><button class="secondary" disabled={busy} onclick={()=>current&&change(current,'remove')}>Confirmer la désinstallation</button><button class="text-action" disabled={busy} onclick={()=>confirming=false}>Annuler</button></div></div>{/if}
  </div></article>
 {:else}<p role="status">Chargement du module…</p>{/if}
{:else if tab==='installed'}
 {#each visible as p (p.id)}<ExtensionCard id={'recognition.'+p.id} name={p.name} family={isCatalogue(p)?'Médias · catalogue':p.kind==='audiotags'?'Métadonnées · fichiers locaux':'Médias · reconnaissance'} description={p.factory?(isCatalogue(p)?'Regroupement et ordre des médias à partir des informations du catalogue.':p.kind==='audiotags'?'Titres, artistes, albums et pochettes intégrés aux fichiers.':'Identification locale des noms et formats pendant les scans.'):'Module externe · '+p.publisher+' (éditeur non vérifié)'} status={status(p)} enabled={p.enabled&&!safeMode} disabled={busy||safeMode&&!p.enabled} toggle={()=>change(p,p.enabled?'disable':'enable')} open={()=>select(p.id)}/>{/each}
{:else if visible.length}
 <section><h3>Modules fournis avec Mosaune</h3><p>Ces modules sont disponibles ici sans téléchargement. L’installation ne les active pas automatiquement.</p><div class="extension-grid">{#each visible as p (p.id)}<ExtensionCard id={'recognition.'+p.kind} name={names[p.kind]} family={isCatalogue(p)?'Médias · catalogue':p.kind==='audiotags'?'Métadonnées · fichiers locaux':'Médias · reconnaissance'} description={isCatalogue(p)?'Organisation de votre collection. Aucun accès aux fichiers ou au réseau.':p.kind==='audiotags'?'Lecture limitée aux fichiers musicaux analysés. Aucun accès au réseau.':'Reconnaissance locale des noms et formats. Aucun accès au réseau.'} status="À installer" open={()=>select(p.kind)}/>{/each}</div></section>
{/if}

